//go:generate go run mkstypetab.go -o stypetab.go /usr/share/vulkan/registry/vk.xml

package vk

import (
	"fmt"
	"reflect"
)

func StructureTypeOf(t reflect.Type) StructureType {
	sType, ok := sTypeTable[t]
	if !ok {
		panic(fmt.Sprintf("unknown struct %v", t))
	}
	return sType
}
