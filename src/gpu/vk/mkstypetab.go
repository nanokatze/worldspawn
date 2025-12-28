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

var api = flag.String("api", "vulkan", "API")
var platforms = flag.String("platforms", "", "Platforms")
var output = flag.String("o", "stypetab.go", "b")

func main() {
	log.SetFlags(0)

	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var reg registry.Registry
	if err := xml.NewDecoder(f).Decode(&reg); err != nil {
		log.Fatal(err)
	}

	required := make(map[string]struct{})
	handleRequire := func(require registry.Require, extNumber *int64) {
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

	out := new(bytes.Buffer)

	fmt.Fprintln(out, "package vk")

	fmt.Fprintln(out)
	fmt.Fprintln(out, "import \"reflect\"")

	fmt.Fprintln(out)
	fmt.Fprintln(out, "var sTypeTable = map[reflect.Type]StructureType{")

	for _, ty := range reg.Types {
		// TODO: build typeMap and emit from typemap instead of emitting here

		if len(ty.API) > 0 && !slices.Contains(ty.API, *api) {
			continue
		}

		if ty.Alias != nil {
			continue
		}

		switch ty.Category {
		case "struct":
			if _, ok := required[*ty.Name]; !ok {
				continue
			}
			member0 := ty.Members[0]
			member0Name, _ := cparser.ParseDecl(plainText(member0.Inner))
			if member0Name != "sType" {
				continue
			}
			// TODO: assert that the type of member0 is VkStructureType
			// Skip VkBase{In,Out}Structure
			if len(member0.Values) == 0 {
				continue
			}
			if len(member0.Values) > 1 {
				log.Fatalf("%s::sType has multiple legal values", *ty.Name)
			}
			fmt.Fprintf(out, "reflect.TypeFor[%s](): %s,\n", trimNamespacePrefix(*ty.Name), trimNamespacePrefix(member0.Values[0]))
		}
	}

	fmt.Fprintf(out, "}\n")

	formatted, err := format.Source(out.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(*output, formatted, 0666); err != nil {
		log.Fatal(err)
	}
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
