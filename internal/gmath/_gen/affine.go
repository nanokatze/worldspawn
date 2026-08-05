package main

import (
	"io"
	"text/template"
)

type affineGen struct{ D int64 }

func (gen affineGen) Gen(w io.Writer) error { return affineTmpl.Execute(w, &gen) }

var affineTmpl = template.Must(template.New("affine").Parse(`
{{$AffineD := printf "Affine%d" .D}}

{{$TRSD := printf "TRS%d" .D}}

type {{$AffineD}}[T constraints.Float] struct {
	M Mat{{.D}}x{{.D}}f32
	T Vec{{.D}}[T]
}

type (
	Affine{{.D}}f32 = {{$AffineD}}[float32]
	Affine{{.D}}f64 = {{$AffineD}}[float64]
)

func (A {{$AffineD}}[From]) Convert[To constraints.Float]() {{$AffineD}}[To] {
	return {{$AffineD}}[To]{
		M: A.M,
		T: A.T.Convert[To](),
	}
}

func Affine{{.D}}One[T constraints.Float]() {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: Mat{{.D}}x{{.D}}One[float32](),
	}
}

func (A {{$AffineD}}[T]) Add(B {{$AffineD}}[T]) {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: A.M.Add(B.M),
		T: A.T.Add(B.T),
	}
}

func (A {{$AffineD}}[T]) Scale(λ T) {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: A.M.Scale(float32(λ)),
		T: A.T.Scale(λ),
	}
}

func (A {{$AffineD}}[T]) Mul(B {{$AffineD}}[T]) {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: A.M.Mul(B.M),
		T: A.T.Add(A.M.Convert[T]().Mulv(B.T)),
	}
}

/*
func (A {{$AffineD}}[T]) Inv() {{$AffineD}}[T] {
	panic("not implemented")
}
*/

func (A {{$AffineD}}[T]) TRS() {{$TRSD}}[T] {
	// TODO: don't assume this is just translation * rotation, properly extract
	// shcale too or at least scale with scale.
	Q := A.M
	R := Mat{{.D}}x{{.D}}UOne[float32]()

	return {{$TRSD}}[T]{
		T: A.T,
		R: Rot{{.D}}FromMat(Q),
		S: R,
	}
}

type {{$TRSD}}[T constraints.Float] struct {
	T Vec{{.D}}[T]
	R Rot{{.D}}
	S Mat{{.D}}x{{.D}}Uf32
}

type (
	TRS{{.D}}f32 = {{$TRSD}}[float32]
	TRS{{.D}}f64 = {{$TRSD}}[float64]
)

func TRS{{.D}}One[T constraints.Float]() {{$TRSD}}[T] {
	return {{$TRSD}}[T]{
		R: Rot{{.D}}One(),
		S: Mat{{.D}}x{{.D}}UOne[float32](),
	}
}

func (trs {{$TRSD}}[T]) Affine() {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		// TODO: special case R.Mat() by S.Mat() for more :b:erf
		M: trs.R.Mat().Mul(trs.S.Mat()),
		T: trs.T,
	}
}
`))
