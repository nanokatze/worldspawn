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

// TODO: simplify this so that it's just an ordinary normalize pls
func (a {{$VecD}}[T]) NormalizeOr(b {{$VecD}}[T]) {{$VecD}}[T] {
	norm2 := a.Dot(a)
	if norm2 == 0 {
		return b
	}
	norm := T(math.Sqrt(float64(norm2)))
	return {{$VecD}}[T]{
		{{- range .D}}
		a[{{.}}] / norm,
		{{- end}}
	}
}

// TODO: make this a method once generic methods are in
func Vec{{.D}}Convert[To, From constraints.Float](a {{$VecD}}[From]) {{$VecD}}[To] {
	return {{$VecD}}[To]{
		{{- range .D}}
		To(a[{{.}}]),
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
