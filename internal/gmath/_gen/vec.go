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

func (a {{$VecD}}[From]) Convert[To constraints.Float]() {{$VecD}}[To] {
	return {{$VecD}}[To]{
		{{- range .D}}
		To(a[{{.}}]),
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

func (a {{$VecD}}[T]) Add(b {{$VecD}}[T]) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		a[{{.}}] + b[{{.}}],
		{{- end}}
	}
}

func (a {{$VecD}}[T]) Sub(b {{$VecD}}[T]) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		a[{{.}}] - b[{{.}}],
		{{- end}}
	}
}

func (a {{$VecD}}[T]) Scale(λ T) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		λ * a[{{.}}],
		{{- end}}
	}
}

func (a {{$VecD}}[T]) Length() T {
	return T(math.Sqrt(float64(a.Dot(a))))
}

func (a {{$VecD}}[T]) Dot(b {{$VecD}}[T]) T {
	return 0 {{- range .D}} + a[{{.}}] * b[{{.}}] {{- end}}
}

func (a {{$VecD}}[T]) Normalize() {{$VecD}}[T] {
	lengthSqr := a.Dot(a)
	if !(lengthSqr > 0) {
		return {{$VecD}}[T]{}
	}
	length := T(math.Sqrt(float64(lengthSqr)))
	return {{$VecD}}[T]{
		{{- range .D}}
		a[{{.}}] / length,
		{{- end}}
	}
}

{{$MatDxD := printf "Mat%dx%d" .D .D}}

func Matvec{{.D}}[T constraints.Float](A {{$MatDxD}}[T], b {{$VecD}}[T]) {{$VecD}}[T] {
	return {{$VecD}}[T](A.Mul{{.D}}x1(Mat{{.D}}x1[T](b)))
}
`))
