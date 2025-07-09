package registry

import "strings"

type CommaSeparatedStrings []string

func (ss *CommaSeparatedStrings) UnmarshalText(text []byte) error {
	*ss = strings.Split(string(text), ",")
	return nil
}

// See https://github.com/KhronosGroup/Vulkan-Docs/blob/main/xml/registry.rnc
type Registry struct {
	Types      []Type      `xml:"types>type"`
	Enums      []Enum      `xml:"enums"`
	Features   []Feature   `xml:"feature"`
	Extensions []Extension `xml:"extensions>extension"`
	Formats    []Format    `xml:"formats>format"`
}

type Type struct {
	Name        *string               `xml:"name,attr"`
	API         CommaSeparatedStrings `xml:"api,attr"`
	Alias       *string               `xml:"alias,attr"`
	Requires    CommaSeparatedStrings `xml:"requires,attr"`
	Category    string                `xml:"category,attr"`
	Deprecated  string                `xml:"deprecated,attr"` // TODO: should be optional?
	ObjTypeEnum string                `xml:"objtypeenum,attr"`

	Members []Member `xml:"member"` // TODO: remove this and have a thing where we parse Inner
	Inner   []byte   `xml:",innerxml"`
}

type Member struct {
	API    CommaSeparatedStrings `xml:"api,attr"`
	Values CommaSeparatedStrings `xml:"values,attr"`
	Inner  []byte                `xml:",innerxml"`
}

type Enum struct {
	Name     *string        `xml:"name,attr"`
	Type     *string        `xml:"type,attr"`
	BitWidth *int64         `xml:"bitwidth,attr"`
	Enums    []RequiredEnum `xml:"enum"`
}

type Format struct {
	Name           string  `xml:"name,attr"`
	Class          string  `xml:"class,attr"`
	BlockSize      string  `xml:"blockSize,attr"`
	TexelsPerBlock string  `xml:"texelsPerBlock,attr"`
	BlockExtent    *string `xml:"blockExtent,attr"`
	Packed         *string `xml:"packed,attr"`
	Compressed     *string `xml:"compressed,attr"`
}

type Feature struct {
	API      CommaSeparatedStrings `xml:"api,attr"`
	Name     string                `xml:"name,attr"`
	Number   string                `xml:"number,attr"`
	Requires []Require             `xml:"require"`
}

type Extension struct {
	Name      string                `xml:"name,attr"`
	Number    int64                 `xml:"number,attr"`
	Platform  CommaSeparatedStrings `xml:"platform,attr"`
	Supported CommaSeparatedStrings `xml:"supported,attr"`
	Requires  []Require             `xml:"require"`
}

type Require struct {
	Comment string         `xml:"comment,attr"`
	Enums   []RequiredEnum `xml:"enum"` // TODO: try to implement a type which will be a list of items and do a xml.Unmarshal into these.
	Types   []RequiredType `xml:"type"`
}

type RequiredCommand struct {
}

type RequiredEnum struct {
	API       CommaSeparatedStrings `xml:"api,attr"`
	Extends   *string               `xml:"extends,attr"`
	Name      string                `xml:"name,attr"`
	Type      *string               `xml:"type,attr"`
	Value     *string               `xml:"value,attr"`
	BitPos    *int64                `xml:"bitpos,attr"`
	ExtNumber *int64                `xml:"extnumber,attr"`
	Offset    *int64                `xml:"offset,attr"`
	Dir       string                `xml:"dir,attr"`
	Alias     *string               `xml:"alias,attr"`
}

type RequiredType struct {
	Name string `xml:"name,attr"`
}
