package main

import "text/template"

type rotGen struct{ D int64 }

var rotTmpl = template.Must(template.New("rot").Parse(`
{{$RotD := printf "Rot%d" .D}}

type {{$RotD}} struct {
}

func Rot{{.D}}One() {{$RotD}} {
	panic("not implemented")
}

func Rot{{.D}}AToB(a, b Vec{{.D}}f32) {{$RotD}} {
	panic("not implemented")
}

func Rot{{.D}}FromMat(M Mat{{.D}}x{{.D}}f32) {{$RotD}} {
	panic("not implemented")
}

func (R {{$RotD}}) Renormalize() {{$RotD}} {
	panic("not implemented")
}

func (R {{$RotD}}) Mul(S {{$RotD}}) {{$RotD}} {
	panic("not implemented")
}

func (R {{$RotD}}) Inv() {{$RotD}} {
	panic("not implemented")
}

func (R {{$RotD}}) Mat() Mat{{.D}}x{{.D}}f32 {
	panic("not implemented")
}

func (R {{$RotD}}) Rotate[T constraints.Float](v Vec{{.D}}[T]) Vec{{.D}}[T] {
	panic("not implemented")
}
`))
