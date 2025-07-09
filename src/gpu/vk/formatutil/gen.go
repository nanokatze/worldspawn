//go:build ignore

package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"log"
	"maps"
	"os"
	"slices"
	"strings"

	"worldspawn/gpu/vk/registry"
)

var api = flag.String("api", "vulkan", "API")
var platforms = flag.String("platforms", "", "Platforms")
var output = flag.String("o", "format_table.go", "b")

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

	out := new(bytes.Buffer)

	fmt.Fprintln(out, "package formatutil")

	fmt.Fprintln(out)
	fmt.Fprintln(out, `import "worldspawn/gpu/vk"`)

	{
		fmt.Fprintln(out)
		fmt.Fprintln(out, "const (")
		fmt.Fprintln(out, "_ Class = iota")
		classes := make(map[string]struct{})
		for _, format := range reg.Formats {
			classes[format.Class] = struct{}{}
		}
		for _, class := range slices.Sorted(maps.Keys(classes)) {
			fmt.Fprintln(out, formatClassName(class))
		}
		fmt.Fprintln(out, ")")
	}

	{
		fmt.Fprintln(out)
		fmt.Fprintln(out, "var formatTable = map[vk.Format]Description{")
		for _, format := range reg.Formats {
			fmt.Fprintf(out, "vk.%v: {\n", trimNamespacePrefix(format.Name))
			fmt.Fprintf(out, "Class: %v,\n", formatClassName(format.Class))
			fmt.Fprintf(out, "BlockSize: %v,\n", format.BlockSize)
			blockExtent := strings.Split(or(format.BlockExtent, "1,1,1"), ",")
			fmt.Fprintf(out, "BlockExtent: vk.Extent3D{Width: %v, Height: %v, Depth: %v},\n", blockExtent[0], blockExtent[1], blockExtent[2])
			// format.Compressed
			fmt.Fprintf(out, "},\n")
		}
		fmt.Fprintln(out, "}")
	}

	formatted, err := format.Source(out.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(*output, formatted, 0666); err != nil {
		log.Fatal(err)
	}
}

// :frog:
func or[T any](p *T, dv T) T {
	if p != nil {
		return *p
	}
	return dv
}

func formatClassName(class string) string {
	name := class
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, " ", "_")
	name = "CLASS_" + strings.ToUpper(name)
	return name
}

func trimNamespacePrefix(name string) string {
	name = strings.TrimPrefix(name, "VK_")
	name = strings.TrimPrefix(name, "Vk")
	name = strings.TrimPrefix(name, "vk")
	return name
}
