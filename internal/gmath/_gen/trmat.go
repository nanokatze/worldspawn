package main

import (
	"io"
	"text/template"
)

type trmatGen struct{ M int64 }

func (gen trmatGen) Gen(w io.Writer) error { return trmatTmpl.Execute(w, &gen) }

var trmatTmpl = template.Must(template.New("trmat").Parse(`
{{$upmatMxM := printf "Gupmat%dx%d" .M .M}}

type {{$upmatMxM}}[T constraints.Float] [{{.M}} * ({{.M}} + 1) / 2]T

type Upmat{{.M}}x{{.M}} = {{$upmatMxM}}[float32]

func (A *{{$upmatMxM}}[T]) Index(i, j int) *T {
	// TODO: remove this open coded garbage
	switch i {
	case 0:
		return &A[0:3][j]
	case 1:
		return &A[3:5][j-1]
	case 2:
		return &A[5:6][j-2]
	default:
		panic("unreachable")
	}

	A_i := A[0:][:{{.M}}-i]
	A_i_j := &A_i[j-i]
	return A_i_j
}

func Upmat{{.M}}x{{.M}}One[T constraints.Float]() {{$upmatMxM}}[T] {
	var I {{$upmatMxM}}[T]
	{{- range .M}}
	*I.Index({{.}}, {{.}}) = 1
	{{- end}}
	return I
}

func Upmat{{.M}}x{{.M}}Diag[T constraints.Float](d gvec{{.M}}[T]) {{$upmatMxM}}[T] {
	var D {{$upmatMxM}}[T]
	{{- range .M}}
	*D.Index({{.}}, {{.}}) = d[{{.}}]
	{{- end}}
	return D
}

func Upmat{{.M}}x{{.M}}FromMat[T constraints.Float]() {{$upmatMxM}}[T] {
	var I {{$upmatMxM}}[T]
	{{- range .M}}
	*I.Index({{.}}, {{.}}) = 1
	{{- end}}
	return I
}

func (A {{$upmatMxM}}[T]) Inv() {{$upmatMxM}}[T] {
	panic("not implemented")
}

func (A {{$upmatMxM}}[T]) Mul(B {{$upmatMxM}}[T]) {{$upmatMxM}}[T] {
	var AB {{$upmatMxM}}[T]
	for i := range {{.M}} {
		for k := i; k < {{.M}}; k++ {
			for j := k; j < {{.M}}; j++ {
				*AB.Index(i, k) += *A.Index(i, j) * *B.Index(j, k)
			}
		}
	}
	return AB
}

{{$matMxM := printf "gmat%dx%d" .M .M}}

func (U {{$upmatMxM}}[T]) ToMat() {{$matMxM}}[T] {
	var M {{$matMxM}}[T]
	for i := range {{.M}} {
		for j := i; j < {{.M}}; j++ {
			*M.Index(i, j) = *U.Index(i, j)
		}
	}
	return M
}
`))
