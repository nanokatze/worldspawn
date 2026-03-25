package main

import (
	"io"
	"text/template"
)

// Move matrix and vector stuff into a separate package and only keep higher
// level objects in this one? E.g. instead of a single VecD we'd have
// TranslationD and ScaleD both with .Mul method for composition.

// TODO: unify multiplication templates into something single so that we can
// reuse them in square matrix mul, general matmul and matvec

type matGen struct{ M, N int64 }

func (gen matGen) Gen(w io.Writer) error { return matTmpl.Execute(w, &gen) }

var matTmpl = template.Must(template.New("mat").Parse(`
{{$matMxN := printf "gmat%dx%d" .M .N}}

type {{$matMxN}}[T constraints.Float] [{{.M}} * {{.N}}]T

type Mat{{.M}}x{{.N}} = {{$matMxN}}[float32]

func (A *{{$matMxN}}[T]) Index(i, j int) *T {
	A_i := A[i*{{.N}}:][:{{.N}}]
	A_i_j := &A_i[j]
	return A_i_j
}

{{if (eq .M .N)}}

{{$matMxM := printf "gmat%dx%d" .M .M}}

func Mat{{.M}}x{{.M}}One[T constraints.Float]() {{$matMxM}}[T] {
	var I {{$matMxM}}[T]
	{{- range .M}}
	*I.Index({{.}}, {{.}}) = 1
	{{- end}}
	return I
}

func Mat{{.M}}x{{.M}}Diag[T constraints.Float](d gvec{{.M}}[T]) {{$matMxM}}[T] {
	var D {{$matMxM}}[T]
	{{- range .M}}
	*D.Index({{.}}, {{.}}) = d[{{.}}]
	{{- end}}
	return D
}

func (A {{$matMxM}}[T]) Inv() {{$matMxM}}[T] {
	panic("not implemented")
}

func (A {{$matMxM}}[T]) Mul(B {{$matMxM}}[T]) {{$matMxM}}[T] {
	var AB {{$matMxM}}[T]
	{{- range $i := $.M}}
	{{- range $k := $.M}}
	*AB.Index({{$i}}, {{$k}}) = 0 {{range $j := $.M}} + *A.Index({{$i}}, {{$j}}) * *B.Index({{$j}}, {{$k}}) {{end}}
	{{- end}}
	{{- end}}
	return AB
}

{{end}}
`))

type matmulGen struct{ M, N, P int64 }

func (gen matmulGen) Gen(w io.Writer) error { return matmulTmpl.Execute(w, &gen) }

var matmulTmpl = template.Must(template.New("matmul").Parse(`
{{$matMxN := printf "gmat%dx%d" .M .N}}
{{$matNxP := printf "gmat%dx%d" .N .P}}
{{$matMxP := printf "gmat%dx%d" .M .P}}

func (A {{$matMxN}}[T]) Mul{{.N}}x{{.P}}(B {{$matNxP}}[T]) {{$matMxP}}[T] {
	var AB {{$matMxP}}[T]
	{{- range $i := $.M}}
	{{- range $k := $.P}}
	*AB.Index({{$i}}, {{$k}}) = 0 {{range $j := $.N}} + *A.Index({{$i}}, {{$j}}) * *B.Index({{$j}}, {{$k}}) {{end}}
	{{- end}}
	{{- end}}
	return AB
}
`))

type matvecGen struct{ M, N int64 }

func (gen matvecGen) Gen(w io.Writer) error { return matvecTmpl.Execute(w, &gen) }

var matvecTmpl = template.Must(template.New("matvec").Parse(`
{{$matMxN := printf "gmat%dx%d" .M .N}}
{{$vecN := printf "gvec%d" .N}}
{{$vecM := printf "gvec%d" .M}}

func (A {{$matMxN}}[T]) Mulv(b {{$vecN}}[T]) {{$vecM}}[T] {
	var Ab {{$vecM}}[T]
	{{- range $i := $.M}}
	Ab[{{$i}}] = 0 {{range $j := $.N}} + *A.Index({{$i}}, {{$j}}) * b[{{$j}}] {{end}}
	{{- end}}
	return Ab
}
`))
