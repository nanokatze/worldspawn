package main

import (
	"io"
	"text/template"
)

type matGen struct{ M, N int64 }

func (gen matGen) Gen(w io.Writer) error { return matTmpl.Execute(w, &gen) }

var matTmpl = template.Must(template.New("mat").Parse(`
{{$gmatMxN := printf "gmat%dx%d" .M .N}}

type {{$gmatMxN}}[T constraints.Float] [{{.M}} * {{.N}}]T

type Mat{{.M}}x{{.N}} = {{$gmatMxN}}[float32]

func (A *{{$gmatMxN}}[T]) Index(i, j int) *T {
	return &A[i*{{.N}} : (i+1)*{{.N}}][j]
}
`))

type matringGen struct{ M int64 }

func (gen matringGen) Gen(w io.Writer) error {
	return matringTmpl.Execute(w, &gen)
}

// TODO: unify multiplication templates into something single so that we can
// reuse them in matring, matmul and matvec

var matringTmpl = template.Must(template.New("matring").Parse(`
{{$gmatMxM := printf "gmat%dx%d" .M .M}}

func {{$gmatMxM}}One[T constraints.Float]() {{$gmatMxM}}[T] {
	var I {{$gmatMxM}}[T]
	{{- range .M}}
	*I.Index({{.}}, {{.}}) = 1
	{{- end}}
	return I
}

func Mat{{.M}}x{{.M}}One() Mat{{.M}}x{{.M}} { return {{$gmatMxM}}One[float32]() }

func Mat{{.M}}x{{.M}}Diag[T constraints.Float](d gvec{{.M}}[T]) {{$gmatMxM}}[T] {
	var D {{$gmatMxM}}[T]
	{{- range .M}}
	*D.Index({{.}}, {{.}}) = d[{{.}}]
	{{- end}}
	return D
}

func (A {{$gmatMxM}}[T]) Inv() {{$gmatMxM}}[T] {
	panic("not implemented")
}

func (A {{$gmatMxM}}[T]) Mul(B {{$gmatMxM}}[T]) {{$gmatMxM}}[T] {
	var C {{$gmatMxM}}[T]
	{{- range $i := $.M}}
	{{- range $k := $.M}}
	*C.Index({{$i}}, {{$k}}) = 0 {{range $j := $.M}} + *A.Index({{$i}}, {{$j}}) * *B.Index({{$j}}, {{$k}}) {{end}}
	{{- end}}
	{{- end}}
	return C
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
	*C.Index({{$i}}, {{$k}}) = 0 {{range $j := $.N}} + *A.Index({{$i}}, {{$j}}) * *B.Index({{$j}}, {{$k}}) {{end}}
	{{- end}}
	{{- end}}
	return C
}
`))

type matvecGen struct{ M, N int64 }

func (gen matvecGen) Gen(w io.Writer) error { return matvecTmpl.Execute(w, &gen) }

var matvecTmpl = template.Must(template.New("matvec").Parse(`
{{$gmatMxN := printf "gmat%dx%d" .M .N}}
{{$gvecN := printf "gvec%d" .N}}
{{$gvecM := printf "gvec%d" .M}}

func (A {{$gmatMxN}}[T]) Mulv(b {{$gvecN}}[T]) {{$gvecM}}[T] {
	var c {{$gvecM}}[T]
	{{- range $i := $.M}}
	c[{{$i}}] = 0 {{range $j := $.N}} + *A.Index({{$i}}, {{$j}}) * b[{{$j}}] {{end}}
	{{- end}}
	return c
}
`))
