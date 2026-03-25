package main

import (
	"io"
	"text/template"
)

type affineGen struct{ D int64 }

func (gen affineGen) Gen(w io.Writer) error { return affineTmpl.Execute(w, &gen) }

var affineTmpl = template.Must(template.New("affine").Parse(`
{{$AffineD := printf "Affine%d" .D}}

type {{$AffineD}}[T constraints.Float] struct {
	M Mat{{.D}}x{{.D}}f32
	T Vec{{.D}}[T]
}

type (
	Affine{{.D}}f32 = {{$AffineD}}[float32]
	Affine{{.D}}f64 = {{$AffineD}}[float64]
)

// TODO: change constructors of other types to the same naming (i.e. generic and no G prefix)
func Affine{{.D}}One[T constraints.Float]() {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: Mat{{.D}}x{{.D}}One[float32](),
	}
}

/*
func (f {{$AffineD}}[T]) Inv() {{$AffineD}}[T] {
	panic("not implemented")
}
*/

func (f {{$AffineD}}[T]) Mul(g {{$AffineD}}[T]) {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: f.M.Mul(g.M),
		T: f.T.Add(Mat{{.D}}x{{.D}}[T](convert9[T](f.M)).Mulv(g.T)),
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

// TODO: rename TRHS back to TRS when we plop scale and shearing into one object
{{$TRSD := printf "TRHS%d" .D}}

type {{$TRSD}}[T constraints.Float] struct {
	T Vec{{.D}}[T]
	R Rot{{.D}}
	S Shcale{{.D}}
}

type (
	TRHS{{.D}}f32 = {{$TRSD}}[float32]
	TRHS{{.D}}f64 = {{$TRSD}}[float64]
)

func TRHS{{.D}}One[T constraints.Float]() {{$TRSD}}[T] {
	return {{$TRSD}}[T]{
		R: Rot{{.D}}One(),
		S: Shcale{{.D}}One(),
	}
}

func TRHS{{.D}}FromAffine[T constraints.Float](f {{$AffineD}}[T]) {{$TRSD}}[T] {
	panic("not implemented")

	// TODO: f.M into R and S using QR decomp

	return {{$TRSD}}[T]{
		T: f.T,
	}
}

func (trhs {{$TRSD}}[T]) ToAffine() {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		// TODO: special case R.Mat() by S.Mat() for more :b:erf and kill those
		// methods
		M: trhs.R.ToMat().Mul(trhs.S.ToMat()),
		T: trhs.T,
	}
}
`))
