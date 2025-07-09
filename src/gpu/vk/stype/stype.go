//go:generate go run gen.go -o stype_table.go /usr/share/vulkan/registry/vk.xml

package stype

import (
	"fmt"
	"reflect"

	"worldspawn/gpu/vk"
)

func SType(v any) vk.StructureType {
	t := reflect.TypeOf(v)

	sType, ok := sTypeTable[t]
	if !ok {
		panic(fmt.Sprintf("unknown struct %v", t))
	}
	return sType
}
