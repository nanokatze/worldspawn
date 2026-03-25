package main

import (
	"io"
	"text/template"
)

type vecGen struct{ D int64 }

func (gen vecGen) Gen(w io.Writer) error { return vecTmpl.Execute(w, &gen) }

var vecTmpl = template.Must(template.New("vec").Parse(`
{{$vecD := printf "gvec%d" .D}}

type {{$vecD}}[T constraints.Float] [{{.D}}]T

type (
	Vec{{.D}}  = {{$vecD}}[float32]
	DVec{{.D}} = {{$vecD}}[float64]
)

func Vec{{.D}}Ones[T constraints.Float]() {{$vecD}}[T] {
	return {{$vecD}}[T]{
		{{- range .D}}
		1,
		{{- end}}
	}
}

func (a {{$vecD}}[T]) Add(b {{$vecD}}[T]) {{$vecD}}[T] {
	return {{$vecD}}[T]{
		{{- range .D}}
		a[{{.}}] + b[{{.}}],
		{{- end}}
	}
}

func (a {{$vecD}}[T]) Sub(b {{$vecD}}[T]) {{$vecD}}[T] {
	return {{$vecD}}[T]{
		{{- range .D}}
		a[{{.}}] - b[{{.}}],
		{{- end}}
	}
}

func (a {{$vecD}}[T]) Scale(lambda T) {{$vecD}}[T] {
	return {{$vecD}}[T]{
		{{- range .D}}
		lambda * a[{{.}}],
		{{- end}}
	}
}

func (a {{$vecD}}[T]) Length() T {
	return T(math.Sqrt(float64(a.Dot(a))))
}

func (a {{$vecD}}[T]) Dot(b {{$vecD}}[T]) T {
	return 0 {{- range .D}} + a[{{.}}] * b[{{.}}] {{- end}}
}

// TODO: simplify this so that it's just an ordinary normalize pls
func (a {{$vecD}}[T]) NormalizeOr(b {{$vecD}}[T]) {{$vecD}}[T] {
	norm2 := a.Dot(a)
	if norm2 == 0 {
		return b
	}
	norm := T(math.Sqrt(float64(norm2)))
	return {{$vecD}}[T]{
		{{- range .D}}
		a[{{.}}] / norm,
		{{- end}}
	}
}

// TODO: make this a method once generic methods are in
func Vec{{.D}}Convert[To, From constraints.Float](a {{$vecD}}[From]) {{$vecD}}[To] {
	return {{$vecD}}[To]{
		{{- range .D}}
		To(a[{{.}}]),
		{{- end}}
	}
}

// TODO: kill this
func (a {{$vecD}}[T]) Lerp(b {{$vecD}}[T], t T) {{$vecD}}[T] {
	return {{$vecD}}[T]{
		{{- range .D}}
		lerp(a[{{.}}], b[{{.}}], t),
		{{- end}}
	}
}
`))
