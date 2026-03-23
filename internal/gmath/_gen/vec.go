package main

import (
	"io"
	"text/template"
)

type vecGen struct{ D int64 }

func (gen vecGen) Gen(w io.Writer) error { return vecTmpl.Execute(w, &gen) }

var vecTmpl = template.Must(template.New("vec").Parse(`
{{$gvecD := printf "gvec%d" .D}}

type {{$gvecD}}[T constraints.Float] [{{.D}}]T

type (
	Vec{{.D}}  = {{$gvecD}}[float32]
	DVec{{.D}} = {{$gvecD}}[float64]
)

func {{$gvecD}}Ones[T constraints.Float]() {{$gvecD}}[T] {
	return {{$gvecD}}[T]{
		{{- range .D}}
		1,
		{{- end}}
	}
}

func Vec{{.D}}Ones() Vec{{.D}} { return {{$gvecD}}Ones[float32]() }
func DVec{{.D}}Ones() DVec{{.D}} { return {{$gvecD}}Ones[float64]() }

// TODO: make this a method once generic methods are in?
func Vec{{.D}}Convert[To, From constraints.Float](a {{$gvecD}}[From]) {{$gvecD}}[To] {
	return {{$gvecD}}[To]{
		{{- range .D}}
		To(a[{{.}}]),
		{{- end}}
	}
}

func (a {{$gvecD}}[T]) Add(b {{$gvecD}}[T]) {{$gvecD}}[T] {
	return {{$gvecD}}[T]{
		{{- range .D}}
		a[{{.}}] + b[{{.}}],
		{{- end}}
	}
}

func (a {{$gvecD}}[T]) Sub(b {{$gvecD}}[T]) {{$gvecD}}[T] {
	return {{$gvecD}}[T]{
		{{- range .D}}
		a[{{.}}] - b[{{.}}],
		{{- end}}
	}
}

func (a {{$gvecD}}[T]) Scale(lambda T) {{$gvecD}}[T] {
	return {{$gvecD}}[T]{
		{{- range .D}}
		lambda * a[{{.}}],
		{{- end}}
	}
}

func (a {{$gvecD}}[T]) Length() T {
	return T(math.Sqrt(float64(a.Dot(a))))
}

func (a {{$gvecD}}[T]) Dot(b {{$gvecD}}[T]) T {
	return 0 {{- range .D}} + a[{{.}}] * b[{{.}}] {{- end}}
}

// TODO: simplify this so that it's just an ordinary normalize pls
func (a {{$gvecD}}[T]) NormalizeOr(b {{$gvecD}}[T]) {{$gvecD}}[T] {
	norm2 := a.Dot(a)
	if norm2 == 0 {
		return b
	}
	norm := T(math.Sqrt(float64(norm2)))
	return {{$gvecD}}[T]{
		{{- range .D}}
		a[{{.}}] / norm,
		{{- end}}
	}
}

{{/* kill this in favor of global lerp that operates on random float arrays? */}}
func (a {{$gvecD}}[T]) Lerp(b {{$gvecD}}[T], t T) {{$gvecD}}[T] {
	return {{$gvecD}}[T]{
		{{- range .D}}
		lerp(a[{{.}}], b[{{.}}], t),
		{{- end}}
	}
}
`))
