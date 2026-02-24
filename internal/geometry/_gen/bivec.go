package main

import (
	"io"
	"slices"
	"text/template"
)

type bivecGen struct{ D int64 }

func (gen bivecGen) Gen(w io.Writer) error {
	return bivecTmpl.Execute(w, &gen)
}

var bivecTmpl = template.Must(template.New("bivec").Funcs(bivecTmplFuncs).Parse(`
{{$vecD := printf "vec%d" .D}}
{{$bivecD := printf "bivec%d" .D}}

// {{.D}}-dimensional bivector.
type Bivec{{.D}} = {{$bivecD}}[float32]

type {{$bivecD}}[T constraints.Float] [{{binomial .D 2}}]T

{{/* TODO: make wedge generator be a separate template and generalize it to different grade operands */}}
func (a {{$vecD}}[T]) Wedge(b {{$vecD}}[T]) {{$bivecD}}[T] {
	return {{$bivecD}}[T]{
		{{- range $e := (bivecBasis .D)}}
		a[{{index $e 0}}] * b[{{index $e 1}}] - a[{{index $e 1}}] * b[{{index $e 0}}], // {{index $e 0}} {{index $e 1}}
		{{- end}}
	}
}
`))

var bivecTmplFuncs = template.FuncMap{
	"binomial":   binomial,
	"bivecBasis": bivecBasis,
}

func bivecBasis(d int64) [][2]int64 {
	// TODO: could we come up with a generalization that hits this?
	if d == 3 {
		return [][2]int64{
			{1, 2},
			{2, 0},
			{0, 1},
		}
	}

	return slices.Collect(func(yield func([2]int64) bool) {
		for i := range d {
			for j := i + 1; j < d; j++ {
				yield([2]int64{i, j})
			}
		}
	})
}
