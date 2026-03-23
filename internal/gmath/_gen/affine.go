package main

import (
	"io"
	"text/template"
)

type affineGen struct{ D int64 }

func (gen affineGen) Gen(w io.Writer) error { return affineTmpl.Execute(w, &gen) }

var affineTmpl = template.Must(template.New("affine").Parse(`
{{$affineD := printf "GAffine%d" .D}}

type {{$affineD}}[T constraints.Float] struct {
	M Mat{{.D}}x{{.D}} // rotation * shearing * scaling
	T gvec{{.D}}[T]    // translation
}

type Affine{{.D}} = {{$affineD}}[float32]

type DAffine{{.D}} = {{$affineD}}[float64]

// TODO: change constructors of other types to the same naming (i.e. generic and no G prefix)
func Affine{{.D}}One[T constraints.Float]() {{$affineD}}[T] {
	return {{$affineD}}[T]{
		M: Mat{{.D}}x{{.D}}One(),
	}
}

/*
func (f {{$affineD}}[T]) Inv() {{$affineD}}[T] {
	panic("not implemented")
}
*/

func (f {{$affineD}}[T]) Mul(g {{$affineD}}[T]) {{$affineD}}[T] {
	return {{$affineD}}[T]{
		M: f.M.Mul(g.M),
		T: f.T.Add(gmat{{.D}}x{{.D}}[T](convert9[T](f.M)).Mulv(g.T)),
	}
}

// TODO: just introduce MatMxMConvert or whatever pls
func convert9[To, From constraints.Float](x [9]From) [9]To {
	return [9]To{
		{{- range 9}}
		To(x[{{.}}]),
		{{- end}}
	}
}

{{$trhsD := printf "GTRHS%d" .D}}

type {{$trhsD}}[T constraints.Float] struct {
	T gvec{{.D}}[T]
	R Rot{{.D}}
	H Vec{{.D}} // TODO: should be a bivector; for now must be zero
	S Vec{{.D}}
}

type TRHS{{.D}} = {{$trhsD}}[float32]

type DTRHS{{.D}} = {{$trhsD}}[float64]

func TRHS{{.D}}One[T constraints.Float]() {{$trhsD}}[T] {
	return {{$trhsD}}[T]{
		R: Rot{{.D}}One(),
		S: Vec{{.D}}Ones(),
	}
}

func TRHS{{.D}}FromAffine[T constraints.Float](f {{$affineD}}[T]) {{$trhsD}}[T] {
	panic("not implemented")
}

func (trhs {{$trhsD}}[T]) ToAffine() {{$affineD}}[T] {
	// TODO: include shearing
	return {{$affineD}}[T]{
		M: trhs.R.Mat().Mul(Mat{{.D}}x{{.D}}Diag(trhs.S)),
		T: trhs.T,
	}
}
`))
