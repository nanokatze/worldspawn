package main

import (
	"io"
	"text/template"
)

type affineGen struct{ D int64 }

func (gen affineGen) Gen(w io.Writer) error { return affineTmpl.Execute(w, &gen) }

var affineTmpl = template.Must(template.New("affine").Parse(`
{{$gaffineD := printf "gaffine%d" .D}}

type {{$gaffineD}}[T constraints.Float] struct {
	A Mat{{.D}}x{{.D}} // rotation, scaling and shearing
	B gvec{{.D}}[T]    // translation
}

func (a {{$gaffineD}}[T]) Inv() {{$gaffineD}}[T] {
	panic("not implemented")
}

func (a {{$gaffineD}}[T]) Mul(b {{$gaffineD}}[T]) {{$gaffineD}}[T] {
	panic("not implemented")
}

// TODO: a method to decompose/factor {{$gaffineD}} into its constituents

// TODO: call interpolation things something else pls
func (a {{$gaffineD}}[T]) Lerp(b {{$gaffineD}}[T], t float32) {{$gaffineD}}[T] {
	panic("not implemented")
}
`))
