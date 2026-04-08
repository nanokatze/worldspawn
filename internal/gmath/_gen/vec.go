package main

import (
	"io"
	"text/template"
)

type vecGen struct{ D int64 }

func (gen vecGen) Gen(w io.Writer) error { return vecTmpl.Execute(w, &gen) }

var vecTmpl = template.Must(template.New("vec").Parse(`
{{$VecD := printf "Vec%d" .D}}

type {{$VecD}}[T constraints.Float] [{{.D}}]T

type (
	Vec{{.D}}f32 = {{$VecD}}[float32]
	Vec{{.D}}f64 = {{$VecD}}[float64]
)

// TODO: make this a method once generic methods are in
func Vec{{.D}}Convert[To, From constraints.Float](a {{$VecD}}[From]) {{$VecD}}[To] {
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

func (a {{$VecD}}[T]) Scale(lambda T) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		lambda * a[{{.}}],
		{{- end}}
	}
}

func (a {{$VecD}}[T]) Length() T {
	return T(math.Sqrt(float64(a.Dot(a))))
}

func (a {{$VecD}}[T]) Dot(b {{$VecD}}[T]) T {
	return 0 {{- range .D}} + a[{{.}}] * b[{{.}}] {{- end}}
}

func (v {{$VecD}}[T]) Normalize() {{$VecD}}[T] {
	lengthSqr := v.Dot(v)
	if !(lengthSqr > 0) {
		return {{$VecD}}[T]{}
	}
	length := T(math.Sqrt(float64(lengthSqr)))
	return {{$VecD}}[T]{
		{{- range .D}}
		v[{{.}}] / length,
		{{- end}}
	}
}

// TODO: kill this
func (a {{$VecD}}[T]) Lerp(b {{$VecD}}[T], t T) {{$VecD}}[T] {
	return {{$VecD}}[T]{
		{{- range .D}}
		lerp(a[{{.}}], b[{{.}}], t),
		{{- end}}
	}
}
`))
