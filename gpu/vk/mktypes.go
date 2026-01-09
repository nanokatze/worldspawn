//go:build ignore

package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"slices"
	"strings"

	"worldspawn/gpu/vk/registry"
	"worldspawn/gpu/vk/registry/cparser"
)

// TODO: clean up this garbage

// What we have to do is *first* walk over all the type, enum, etc, definitions,
// and collect them. Next, we walk the requires (by core and extensions), filter
// things appropriately, expand enum defs with more entries, etc. We should
// always skip union types and types containing bit fields, and handroll them
// instead (at least for now.)

var api = flag.String("api", "vulkan", "API to generate bindings for")
var platforms = flag.String("platforms", "", "Platforms to generate bindings for")
var output = flag.String("o", "types.go", "b")

func main() {
	log.SetFlags(0)

	flag.Parse()

	// TODO: *alias* Flags and FlagBits enums so that we don't need to cast
	// things

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var reg registry.Registry
	if err := xml.NewDecoder(f).Decode(&reg); err != nil {
		log.Fatal(err)
	}

	// TODO: build typeMap, enumMap etc and then generate things as we iterate over

	out := new(bytes.Buffer)

	fmt.Fprintf(out, "// Code generated\n")

	fmt.Fprintln(out)
	fmt.Fprintf(out, "package vk\n")

	fmt.Fprintln(out)
	fmt.Fprintf(out, "import \"unsafe\"\n")
	fmt.Fprintf(out, "import \"structs\"\n")

	// TODO: we should get rid of this set and instead just consult the required
	// set. If the type is in required set, we print it, one way or another.
	seen := make(map[string]struct{})

	required := make(map[string]struct{})

	handleEnum := func(enum registry.RequiredEnum, extends *string, extNumber *int64) {
		decl := trimNamespacePrefix(enum.Name)
		if enum.Extends != nil {
			extends = enum.Extends
		}
		if enum.ExtNumber != nil {
			extNumber = enum.ExtNumber
		}
		if extends != nil {
			decl += " " + trimNamespacePrefix(*extends)
		}

		fmt.Fprintln(out)
		switch {
		case enum.Value != nil:
			fmt.Fprintf(out, "const %s = %v\n", decl, *enum.Value)
		case enum.BitPos != nil:
			fmt.Fprintf(out, "const %s = 1 << %v\n", decl, *enum.BitPos)
		case enum.Offset != nil:
			value := 1000000000 + (*extNumber-1)*1000 + *enum.Offset
			if enum.Dir == "-" {
				value *= -1
			}
			fmt.Fprintf(out, "const %s = %v\n", decl, value)
		case enum.Alias != nil:
			fmt.Fprintf(out, "const %s = %s\n", decl, trimNamespacePrefix(*enum.Alias))
		default:
			// This can be reached because enums may also appear in types. TODO:
			// look at official bindings generator to see how we can handle this.
			log.Print("warning: ", enum)
		}
	}

	handleRequire := func(require registry.Require, extNumber *int64) {
		for _, enum := range require.Enums {
			if _, ok := seen[enum.Name]; ok {
				continue
			}
			handleEnum(enum, nil, extNumber)
			seen[enum.Name] = struct{}{}
		}
		for _, ty := range require.Types {
			required[ty.Name] = struct{}{}
		}
	}

	for _, feature := range reg.Features {
		if !slices.Contains(feature.API, *api) {
			continue
		}

		for _, require := range feature.Requires {
			handleRequire(require, nil)
		}
	}

	for _, ext := range reg.Extensions {
		// Is this the correct way of handling supported?
		if slices.Contains(ext.Supported, "disabled") {
			continue
		}
		if len(ext.Platform) > 0 {
			continue
		}
		if strings.Contains(ext.Name, "video") {
			continue
		}

		for _, require := range ext.Requires {
			handleRequire(require, &ext.Number)
		}
	}

	for _, ty := range reg.Types {
		// TODO: build typeMap and emit from typemap instead of emitting here

		if len(ty.API) > 0 && !slices.Contains(ty.API, *api) {
			continue
		}

		if ty.Alias != nil {
			if _, ok := required[*ty.Name]; !ok {
				continue
			}
			fmt.Fprintln(out)
			fmt.Fprintf(out, "type %v = %v\n", trimNamespacePrefix(*ty.Name), trimNamespacePrefix(*ty.Alias))
			continue
		}

		switch ty.Category {
		case "":

		case "include":

		case "basetype":
			// TODO: we need C parsing to handle these

		case "bitmask":
			var bitmask struct {
				Type string `xml:"type"`
				Name string `xml:"name"`
			}
			if err := xml.Unmarshal([]byte("<type>"+string(ty.Inner)+"</type>"), &bitmask); err != nil {
				log.Fatal(err)
			}
			if _, ok := required[bitmask.Name]; !ok {
				continue
			}

			fmt.Fprintln(out)
			switch bitmask.Type {
			case "VkFlags":
				fmt.Fprintf(out, "type %v uint32\n", trimNamespacePrefix(bitmask.Name))
			case "VkFlags64":
				fmt.Fprintf(out, "type %v uint64\n", trimNamespacePrefix(bitmask.Name))
			default:
				log.Fatalf("unknown bitmask type '%v'", bitmask.Type)
			}

		case "handle":
			var handle struct {
				Type string `xml:"type"`
				Name string `xml:"name"`
			}
			if err := xml.Unmarshal([]byte("<type>"+string(ty.Inner)+"</type>"), &handle); err != nil {
				log.Fatal(err)
			}
			if _, ok := required[handle.Name]; !ok {
				continue
			}

			fmt.Fprintln(out)
			switch handle.Type {
			case "VK_DEFINE_HANDLE":
				fmt.Fprintf(out, "type %v uintptr\n", trimNamespacePrefix(handle.Name))
			case "VK_DEFINE_NON_DISPATCHABLE_HANDLE":
				fmt.Fprintf(out, "type %v uint64\n", trimNamespacePrefix(handle.Name))
			default:
				log.Fatalf("unknown handle type '%v'", handle.Type)
			}

		case "struct":
			if _, ok := required[*ty.Name]; !ok {
				continue
			}
			fmt.Fprintln(out)
			fmt.Fprintf(out, "type %v struct {\n", trimNamespacePrefix(*ty.Name))
			fmt.Fprintf(out, "_ structs.HostLayout\n")
			type Decl struct {
				Name string
				Type cparser.Node
			}
			var goMembers []Decl
			for _, member := range ty.Members {
				if len(member.API) > 0 && !slices.Contains(member.API, *api) {
					continue
				}
				name, ty := cparser.ParseDecl(plainText(member.Inner))
				goMembers = append(goMembers, Decl{name, ty})
			}
			meh := false
			if len(goMembers) == 3 {
				asd, ok := goMembers[2].Type.(cparser.Name)
				if ok && string(asd) == strings.TrimRight(*ty.Name, "23") {
					for _, m := range goMembers[:2] {
						fmt.Fprintf(out, "%s %s\n", capitalized(m.Name), translateType(m.Type))
					}
					fmt.Fprintf(out, "%s\n", translateType(goMembers[2].Type))
					meh = true
				}
			}
			if !meh {
				for _, m := range goMembers {
					fmt.Fprintf(out, "%s %s\n", capitalized(m.Name), translateType(m.Type))
				}
			}
			fmt.Fprintf(out, "}\n")

		case "union":
			// TODO: generate a struct made out of array of uintN, where N is
			// our alignment requirement, and equip it with methods to set the
			// variants.

		case "enum":
			// This is handled

		default:
			log.Printf("warning: unknown category '%v'", ty.Category)
		}
	}

	for _, consts := range reg.Enums {
		if _, ok := required[*consts.Name]; !ok {
			continue
		}

		decl := trimNamespacePrefix(*consts.Name)
		if consts.BitWidth == nil {
			decl += " int32"
		} else {
			decl += fmt.Sprintf(" int%d", *consts.BitWidth)
		}

		fmt.Fprintln(out)
		fmt.Fprintf(out, "type %s\n", decl)

		for _, enum := range consts.Enums {
			handleEnum(enum, consts.Name, enum.ExtNumber)
		}
	}

	// TODO: generate functions

	formatted, err := format.Source(out.Bytes())
	if err != nil {
		// TODO: don't duplicate this
		if err := os.WriteFile(*output, out.Bytes(), 0644); err != nil {
			log.Fatal(err)
		}
		log.Fatal(err)
	}

	if err := os.WriteFile(*output, formatted, 0644); err != nil {
		log.Fatal(err)
	}
}

var cTypes = map[string]string{
	"char":     "byte",
	"int8_t":   "int8",
	"uint8_t":  "uint8",
	"int16_t":  "int16",
	"uint16_t": "uint16",
	"int32_t":  "int32",
	"uint32_t": "uint32",
	"int64_t":  "int64",
	"uint64_t": "uint64",
	"size_t":   "int",
	"float":    "float32",
	"double":   "float64",
}

func translateType(n cparser.Node) string {
	switch n := n.(type) {
	case cparser.Name:
		if a, ok := cTypes[string(n)]; ok {
			return a
		}
		return trimNamespacePrefix(string(n))
	case cparser.Pointer:
		if n.Elem == cparser.Name("void") {
			return "unsafe.Pointer"
		}
		return "*" + translateType(n.Elem)
	case cparser.Array:
		return "[" + translateType(n.Count) + "]" + translateType(n.Elem)
	default:
		panic("unreachable")
	}
}

func capitalized(s string) string {
	i := 1
	if s[0] == 'p' {
		for ; i < len(s); i++ {
			if s[i] != s[0] {
				break
			}
		}
	}
	return strings.ToUpper(s[:i]) + s[i:]
}

func trimNamespacePrefix(name string) string {
	name = strings.TrimPrefix(name, "VK_")
	name = strings.TrimPrefix(name, "Vk")
	name = strings.TrimPrefix(name, "vk")
	return name
}

func plainText(data []byte) []byte {
	d := xml.NewDecoder(bytes.NewReader(data))

	var buf []byte
	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return buf
			}
			panic("invalid XML")
		}

		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Local == "comment" {
				// Any errors will be handled on the next iteration
				d.Skip()
			}

		case xml.CharData:
			buf = append(buf, t...)
		}
	}
}
