package main

import (
	"io"
	"text/template"
)

// TODO: swizzle/permutation objects? These objects would be basically tuples of
// indices and we would have operations to compose and invert them and
// everything, and apply them to vectors. This would be basically a more
// effishent alternative to matrices.

type vecGen struct{ D int64 }

func (gen vecGen) Gen(w io.Writer) error { return vecTmpl.Execute(w, &gen) }

var vecTmpl = template.Must(template.New("vec").Parse(`
{{$VecD := printf "Vec%d" .D}}

type {{$VecD}}[T constraints.Float] [{{.D}}]T

type (
	Vec{{.D}}f32 = {{$VecD}}[float32]
	Vec{{.D}}f64 = {{$VecD}}[float64]
)

func (v {{$VecD}}[From]) Convert[To constraints.Float]() {{$VecD}}[To] {
	return {{$VecD}}[To]{
		{{- range .D}}
		To(v[{{.}}]),
		{{- end}}
	}
}

func Vec{{.D}}Ones[T constraints.Float]() {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		1,
		{{- end}}
	}
}

func (v {{$VecD}}[T]) Add(w {{$VecD}}[T]) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		v[{{.}}] + w[{{.}}],
		{{- end}}
	}
}

func (v {{$VecD}}[T]) Sub(w {{$VecD}}[T]) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		v[{{.}}] - w[{{.}}],
		{{- end}}
	}
}

func (v {{$VecD}}[T]) Scale(a T) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		v[{{.}}] * a,
		{{- end}}
	}
}

func (v {{$VecD}}[T]) Length() T {
	return T(math.Sqrt(float64(v.Dot(v))))
}

func (v {{$VecD}}[T]) Dot(w {{$VecD}}[T]) T {
	return 0 {{- range .D}} + v[{{.}}] * w[{{.}}] {{- end}}
}

func (v {{$VecD}}[T]) Normalize() {{$VecD}}[T] {
	norm := v.Length()
	if !(norm > 0) {
		return {{$VecD}}[T]{}
	}
	return v.Scale(1.0 / norm)
}

{{$MatDxD := printf "Mat%dx%d" .D .D}}

func Matvec{{.D}}[T constraints.Float](A {{$MatDxD}}[T], v {{$VecD}}[T]) {{$VecD}}[T] {
	return {{$VecD}}[T](A.Mul{{.D}}x1(Mat{{.D}}x1[T](v)))
}
`))
