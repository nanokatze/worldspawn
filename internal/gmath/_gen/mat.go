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
{{$MatMxN := printf "Mat%dx%d" .M .N}}

type {{$MatMxN}}[T constraints.Float] [{{.M}} * {{.N}}]T

type Mat{{.M}}x{{.N}}f32 = {{$MatMxN}}[float32]

func (M {{$MatMxN}}[From]) Convert[To constraints.Float]() {{$MatMxN}}[To] {
	var M2 {{$MatMxN}}[To]
	for i, v := range M {
		M2[i] = To(v)
	}
	return M2
}

func (A *{{$MatMxN}}[T]) Index(i, j int) *T {
	A_i := A[i*{{.N}}:][:{{.N}}]
	A_i_j := &A_i[j]
	return A_i_j
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

func Mat{{.M}}x{{.M}}Diag[T constraints.Float](d Vec{{.M}}[T]) {{$MatMxM}}[T] {
	var D {{$MatMxM}}[T]
	{{- range .M}}
	*D.Index({{.}}, {{.}}) = d[{{.}}]
	{{- end}}
	return D
}

func (A {{$MatMxM}}[T]) Inv() {{$MatMxM}}[T] {
	panic("not implemented")
}

func (A {{$MatMxM}}[T]) Mul(B {{$MatMxM}}[T]) {{$MatMxM}}[T] {
	var AB {{$MatMxM}}[T]
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
{{$MatMxN := printf "Mat%dx%d" .M .N}}
{{$MatNxP := printf "Mat%dx%d" .N .P}}
{{$MatMxP := printf "Mat%dx%d" .M .P}}

func (A {{$MatMxN}}[T]) Mul{{.N}}x{{.P}}(B {{$MatNxP}}[T]) {{$MatMxP}}[T] {
	var AB {{$MatMxP}}[T]
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
{{$MatMxN := printf "Mat%dx%d" .M .N}}
{{$VecN := printf "Vec%d" .N}}
{{$VecM := printf "Vec%d" .M}}

func (A {{$MatMxN}}[T]) Mulv(b {{$VecN}}[T]) {{$VecM}}[T] {
	var Ab {{$VecM}}[T]
	{{- range $i := $.M}}
	Ab[{{$i}}] = 0 {{range $j := $.N}} + *A.Index({{$i}}, {{$j}}) * b[{{$j}}] {{end}}
	{{- end}}
	return Ab
}
`))

type trmatGen struct{ M int64 }

func (gen trmatGen) Gen(w io.Writer) error { return trmatTmpl.Execute(w, &gen) }

var trmatTmpl = template.Must(template.New("trmat").Parse(`
{{$MatMxMU := printf "Mat%dx%dU" .M .M}}

type {{$MatMxMU}}[T constraints.Float] [{{.M}} * ({{.M}} + 1) / 2]T

type Mat{{.M}}x{{.M}}Uf32 = {{$MatMxMU}}[float32]

func (A *{{$MatMxMU}}[T]) Index(i, j int) *T {
	A_i := A[len(A)-triangularNumber({{.M}}-i):][:{{.M}}-i]
	A_i_j := &A_i[j-i]
	return A_i_j
}

func Mat{{.M}}x{{.M}}UOne[T constraints.Float]() {{$MatMxMU}}[T] {
	var I {{$MatMxMU}}[T]
	{{- range .M}}
	*I.Index({{.}}, {{.}}) = 1
	{{- end}}
	return I
}

func Mat{{.M}}x{{.M}}UDiag[T constraints.Float](d Vec{{.M}}[T]) {{$MatMxMU}}[T] {
	var D {{$MatMxMU}}[T]
	{{- range .M}}
	*D.Index({{.}}, {{.}}) = d[{{.}}]
	{{- end}}
	return D
}

func (A {{$MatMxMU}}[T]) Inv() {{$MatMxMU}}[T] {
	panic("not implemented")
}

func (A {{$MatMxMU}}[T]) Mul(B {{$MatMxMU}}[T]) {{$MatMxMU}}[T] {
	var AB {{$MatMxMU}}[T]
	for i := range {{.M}} {
		for k := i; k < {{.M}}; k++ {
			for j := k; j < {{.M}}; j++ {
				*AB.Index(i, k) += *A.Index(i, j) * *B.Index(j, k)
			}
		}
	}
	return AB
}

{{$MatMxM := printf "Mat%dx%d" .M .M}}

func (U {{$MatMxMU}}[T]) ToMat() {{$MatMxM}}[T] {
	var M {{$MatMxM}}[T]
	for i := range {{.M}} {
		for j := i; j < {{.M}}; j++ {
			*M.Index(i, j) = *U.Index(i, j)
		}
	}
	return M
}
`))
