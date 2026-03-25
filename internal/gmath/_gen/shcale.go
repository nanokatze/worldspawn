package main

import (
	"io"
	"text/template"
)

type shcaleGen struct{ D int64 }

func (gen shcaleGen) Gen(w io.Writer) error { return shcaleTmpl.Execute(w, &gen) }

var shcaleTmpl = template.Must(template.New("shcale").Parse(`
{{$shcaleD := printf "Shcale%d" .D}}

type {{$shcaleD}} Upmat{{.D}}x{{.D}}

func Shcale{{.D}}One() Shcale{{.D}} { return Shcale{{.D}}(Upmat{{.D}}x{{.D}}One[float32]()) }

func Shcale{{.D}}FromScale(s Vec{{.D}}) Shcale{{.D}} { return Shcale{{.D}}(Upmat{{.D}}x{{.D}}Diag(s)) }

func (A Shcale{{.D}}) Mul(B Shcale{{.D}}) Shcale{{.D}} {
	A_ := Upmat{{.D}}x{{.D}}(A)
	B_ := Upmat{{.D}}x{{.D}}(B)
	return Shcale{{.D}}(A_.Mul(B_))
}
`))
