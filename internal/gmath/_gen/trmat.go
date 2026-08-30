package main

import (
	"io"
	"text/template"
)

type trmatGen struct{ M int64 }

func (gen trmatGen) Gen(w io.Writer) error { return trmatTmpl.Execute(w, &gen) }

var trmatTmpl = template.Must(template.New("trmat").Parse(`
{{$MatMxMU := printf "Mat%dx%dU" .M .M}}

type {{$MatMxMU}}[T constraints.Float] [{{.M}} * ({{.M}} + 1) / 2]T

type Mat{{.M}}x{{.M}}Uf32 = {{$MatMxMU}}[float32]

func (A *{{$MatMxMU}}[T]) Index(i, j int) *T {
	A_i := A[len(A)-triangularNumber({{.M}}-i):][:{{.M}}-i]
	A_ij := &A_i[j-i]
	return A_ij
}

func Mat{{.M}}x{{.M}}UOne[T constraints.Float]() {{$MatMxMU}}[T] {
	var I {{$MatMxMU}}[T]
	{{- range .M}}
	*I.Index({{.}}, {{.}}) = 1
	{{- end}}
	return I
}

func Mat{{.M}}x{{.M}}UDiag[T constraints.Float](d [{{.M}}]T) {{$MatMxMU}}[T] {
	var D {{$MatMxMU}}[T]
	{{- range .M}}
	*D.Index({{.}}, {{.}}) = d[{{.}}]
	{{- end}}
	return D
}

func (A {{$MatMxMU}}[T]) Add(B {{$MatMxMU}}[T]) {{$MatMxMU}}[T] {
	for i := range A {
		A[i] += B[i]
	}
	return A
}

func (A {{$MatMxMU}}[T]) Scale(λ T) {{$MatMxMU}}[T] {
	for i := range A {
		A[i] *= λ
	}
	return A
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

func (A {{$MatMxMU}}[T]) Inv() {{$MatMxMU}}[T] {
	panic("not implemented")
}

{{$MatMxM := printf "Mat%dx%d" .M .M}}

func (U {{$MatMxMU}}[T]) Mat() {{$MatMxM}}[T] {
	var M {{$MatMxM}}[T]
	for i := range {{.M}} {
		for j := i; j < {{.M}}; j++ {
			*M.Index(i, j) = *U.Index(i, j)
		}
	}
	return M
}
`))
