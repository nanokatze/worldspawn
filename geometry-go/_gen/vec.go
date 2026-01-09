package main

import (
	"io"
	"text/template"
)

type vecGen struct{ D int64 }

func (vec vecGen) Gen(w io.Writer) error { return vecTmpl.Execute(w, &vec) }

var vecTmpl = template.Must(template.New("vec").Parse(`
{{- $vecD := printf "vec%d" .D}}

type Vec{{.D}} = {{$vecD}}[float32]

func Vec{{.D}}Ones() Vec{{.D}} { return {{$vecD}}Ones[float32]() }

type DVec{{.D}} = {{$vecD}}[float64]

func DVec{{.D}}Ones() DVec{{.D}} { return {{$vecD}}Ones[float64]() }

type {{$vecD}}[T constraints.Float] [{{.D}}]T

func {{$vecD}}Ones[T constraints.Float]() {{$vecD}}[T] {
	return {{$vecD}}[T]{
		{{- range .D}}
		1,
		{{- end}}
	}
}

func Vec{{.D}}Convert[To, From constraints.Float](a {{$vecD}}[From]) {{$vecD}}[To] {
	return {{$vecD}}[To]{
		{{- range .D}}
		To(a[{{.}}]),
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

func (a {{$vecD}}[T]) Dot(b {{$vecD}}[T]) T {
	return 0 {{- range .D}} + a[{{.}}] * b[{{.}}] {{- end}}
}

func (a {{$vecD}}[T]) LengthSq() T {
	return a.Dot(a)
}

func (a {{$vecD}}[T]) Length() T {
	return T(math.Sqrt(float64(a.LengthSq())))
}

// TODO: simplify this so that it's just an ordinary normalize pls
func (a {{$vecD}}[T]) NormalizedOr(b {{$vecD}}[T]) {{$vecD}}[T] {
	norm := a.Length()
	if norm == 0 {
		return b
	}
	return {{$vecD}}[T]{
		{{- range .D}}
		a[{{.}}] / norm,
		{{- end}}
	}
}

func (a {{$vecD}}[T]) Lerp(b {{$vecD}}[T], t T) {{$vecD}}[T] {
	return {{$vecD}}[T]{
		{{- range .D}}
		lerp(a[{{.}}], b[{{.}}], t),
		{{- end}}
	}
}
`))
