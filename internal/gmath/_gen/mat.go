package main

import (
	"io"
	"text/template"
)

type matGen struct{ M, N int64 }

func (gen matGen) Gen(w io.Writer) error { return matTmpl.Execute(w, &gen) }

var matTmpl = template.Must(template.New("mat").Parse(`
{{$gmatMxN := printf "gmat%dx%d" .M .N}}

type Mat{{.N}}x{{.M}} = {{$gmatMxN}}[float32]

func Mat{{.N}}x{{.M}}One() Mat{{.N}}x{{.M}} { return {{$gmatMxN}}One[float32]() }

// TODO: flatten matrices so that it's [{{.N}}*{{.M}}]T?
type {{$gmatMxN}}[T constraints.Float] [{{.N}}][{{.M}}]T

func {{$gmatMxN}}One[T constraints.Float]() {{$gmatMxN}}[T] {
	var A {{$gmatMxN}}[T]
	{{- range .N}}
	A[{{.}}][{{.}}] = 1
	{{- end}}
	return A
}
`))

type matinvGen struct{ M int64 }

func (gen matinvGen) Gen(w io.Writer) error { return matinvTmpl.Execute(w, &gen) }

var matinvTmpl = template.Must(template.New("matinv").Parse(`
{{$gmatMxM := printf "gmat%dx%d" .M .M}}

func (A {{$gmatMxM}}[T]) Inv() {{$gmatMxM}}[T] {
	panic("not implemented")
}
`))

type matmulGen struct{ M, N, P int64 }

func (gen matmulGen) Gen(w io.Writer) error { return matmulTmpl.Execute(w, &gen) }

var matmulTmpl = template.Must(template.New("matmul").Parse(`
{{$gmatMxN := printf "gmat%dx%d" .M .N}}
{{$gmatNxP := printf "gmat%dx%d" .N .P}}
{{$gmatMxP := printf "gmat%dx%d" .M .P}}

func (A {{$gmatMxN}}[T]) Mul{{.N}}x{{.P}}(B {{$gmatNxP}}[T]) {{$gmatMxP}}[T] {
	var C {{$gmatMxP}}[T]
	{{- range $i := $.M}}
	{{- range $k := $.P}}

	{{- range $j := $.N}}
	C[{{$i}}][{{$k}}] += A[{{$i}}][{{$j}}] * B[{{$j}}][{{$k}}]
	{{- end}}

	{{- end}}
	{{- end}}
	return C
}
`))
