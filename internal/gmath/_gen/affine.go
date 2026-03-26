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

{{$TRSD := printf "TRS%d" .D}}

type {{$TRSD}}[T constraints.Float] struct {
	T Vec{{.D}}[T]
	R Rot{{.D}}
	S Shcale{{.D}}
}

type (
	TRS{{.D}}f32 = {{$TRSD}}[float32]
	TRS{{.D}}f64 = {{$TRSD}}[float64]
)

func TRS{{.D}}One[T constraints.Float]() {{$TRSD}}[T] {
	return {{$TRSD}}[T]{
		R: Rot{{.D}}One(),
		S: Shcale{{.D}}One(),
	}
}

// TODO: rename to something like "decompose", e.g. AffineDecomposeTRS?
func TRS{{.D}}FromAffine[T constraints.Float](f {{$AffineD}}[T]) {{$TRSD}}[T] {
	// TODO: don't assume this is just translation * rotation, properly extract
	// shcale too or at least scale with scale.
	Q := f.M
	R := Mat{{.D}}x{{.D}}UOne[float32]()

	return {{$TRSD}}[T]{
		T: f.T,
		R: Rot{{.D}}FromMat(Q),
		S: Shcale{{.D}}(R),
	}
}

// TODO: rename this to "Compose"?
func (trs {{$TRSD}}[T]) ToAffine() {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		// TODO: special case R.Mat() by S.Mat() for more :b:erf and kill those
		// methods
		M: trs.R.ToMat().Mul(trs.S.ToMat()),
		T: trs.T,
	}
}
`))
