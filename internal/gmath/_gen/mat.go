package main

import (
	"io"
	"text/template"
)

// Move matrix and vector stuff into a separate package and only keep higher
// level objects in this one? E.g. instead of a single VecD we'd have
// TranslationD and ScaleD both with .Mul method for composition.

// TODO: unify multiplication templates into something single so that we can
// reuse them in square matrix mul, general matmul and matvec and maybe even
// runtime.

type matGen struct{ M, N int64 }

func (gen matGen) Gen(w io.Writer) error { return matTmpl.Execute(w, &gen) }

var matTmpl = template.Must(template.New("mat").Parse(`
{{$MatMxN := printf "Mat%dx%d" .M .N}}

type {{$MatMxN}}[T constraints.Float] [{{.M}} * {{.N}}]T

type Mat{{.M}}x{{.N}}f32 = {{$MatMxN}}[float32]

func (A {{$MatMxN}}[From]) Convert[To constraints.Float]() {{$MatMxN}}[To] {
	var B {{$MatMxN}}[To]
	for i, v := range A {
		B[i] = To(v)
	}
	return B
}

func (A *{{$MatMxN}}[T]) Index(i, j int) *T {
	A_i := A[i*{{.N}}:][:{{.N}}]
	A_ij := &A_i[j]
	return A_ij
}

{{if (eq .M .N)}}

{{$MatMxM := printf "Mat%dx%d" .M .M}}

func Mat{{.M}}x{{.M}}One[T constraints.Float]() {{$MatMxM}}[T] {
	var I {{$MatMxM}}[T]
	{{- range .M}}
	*I.Index({{.}}, {{.}}) = 1
	{{- end}}
	return I
}

func Mat{{.M}}x{{.M}}Diag[T constraints.Float](d [{{.M}}]T) {{$MatMxM}}[T] {
	var D {{$MatMxM}}[T]
	{{- range .M}}
	*D.Index({{.}}, {{.}}) = d[{{.}}]
	{{- end}}
	return D
}

{{end}}

func (A {{$MatMxN}}[T]) Add(B {{$MatMxN}}[T]) {{$MatMxN}}[T] {
	for i := range A {
		A[i] += B[i]
	}
	return A
}

func (A {{$MatMxN}}[T]) Scale(a T) {{$MatMxN}}[T] {
	for i := range A {
		A[i] *= a
	}
	return A
}

{{if (eq .M .N)}}

{{$MatMxM := printf "Mat%dx%d" .M .M}}

func (A {{$MatMxM}}[T]) Mul(B {{$MatMxM}}[T]) {{$MatMxM}}[T] {
	return A.Mul{{.M}}x{{.M}}(B)
}

/*
func (A {{$MatMxM}}[T]) Inv() {{$MatMxM}}[T] {
	panic("not implemented")
}
*/

{{end}}
`))

type matmulGen struct{ M, N, P int64 }

func (gen matmulGen) Gen(w io.Writer) error { return matmulTmpl.Execute(w, &gen) }

var matmulTmpl = template.Must(template.New("matmul").Parse(`
{{$MatMxN := printf "Mat%dx%d" .M .N}}
{{$MatNxP := printf "Mat%dx%d" .N .P}}
{{$MatMxP := printf "Mat%dx%d" .M .P}}

func (A {{$MatMxN}}[T]) Mul{{.N}}x{{.P}}(B {{$MatNxP}}[T]) {{$MatMxP}}[T] {
	var AB {{$MatMxP}}[T]
	{{- range $i := $.M}}
	{{- range $k := $.P}}
	{{- range $j := $.N}}
	*AB.Index({{$i}}, {{$k}}) += *A.Index({{$i}}, {{$j}}) * *B.Index({{$j}}, {{$k}})
	{{- end}}
	{{- end}}
	{{- end}}
	return AB
}
`))
