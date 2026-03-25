package main

import (
	"io"
	"text/template"
)

type shcaleGen struct{ D int64 }

func (gen shcaleGen) Gen(w io.Writer) error { return shcaleTmpl.Execute(w, &gen) }

var shcaleTmpl = template.Must(template.New("shcale").Parse(`
{{$ShcaleD := printf "Shcale%d" .D}}

{{$MatDxDUf32 := printf "Mat%dx%dUf32" .D .D}}

type {{$ShcaleD}} {{$MatDxDUf32}}

func Shcale{{.D}}One() Shcale{{.D}} { return Shcale{{.D}}(Mat{{.D}}x{{.D}}UOne[float32]()) }

func Shcale{{.D}}FromScale(s Vec{{.D}}f32) Shcale{{.D}} { return Shcale{{.D}}(Mat{{.D}}x{{.D}}UDiag(s)) }

func (A Shcale{{.D}}) Mul(B Shcale{{.D}}) Shcale{{.D}} {
	return Shcale{{.D}}({{$MatDxDUf32}}(A).Mul({{$MatDxDUf32}}(B)))
}
`))
