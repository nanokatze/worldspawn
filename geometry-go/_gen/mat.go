package main

import (
	"io"
	"text/template"
)

type matGen struct{ M, N int64 }

func (mat matGen) Gen(w io.Writer) error { return matTmpl.Execute(w, &mat) }

var matTmpl = template.Must(template.New("mat").Parse(`
{{$matMxN := printf "mat%dx%d" .M .N}}

type Mat{{.N}}x{{.M}} = {{$matMxN}}[float32]

func Mat{{.N}}x{{.M}}One() Mat{{.N}}x{{.M}} { return {{$matMxN}}One[float32]() }

type {{$matMxN}}[T constraints.Float] [{{.N}}][{{.M}}]T

func {{$matMxN}}One[T constraints.Float]() {{$matMxN}}[T] {
	var A {{$matMxN}}[T]
	{{- range .N}}
	A[{{.}}][{{.}}] = 1
	{{- end}}
	return A
}
`))

type matmulGen struct{ M, N, P int64 }

func (matmul matmulGen) Gen(w io.Writer) error { return matmulTmpl.Execute(w, &matmul) }

var matmulTmpl = template.Must(template.New("matmul").Parse(`
{{$matMxN := printf "mat%dx%d" .M .N}}
{{$matNxP := printf "mat%dx%d" .N .P}}
{{$matMxP := printf "mat%dx%d" .M .P}}

func (A {{$matMxN}}[T]) Mul{{.N}}x{{.P}}(B {{$matNxP}}[T]) {{$matMxP}}[T] {
	var C {{$matMxP}}[T]
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
