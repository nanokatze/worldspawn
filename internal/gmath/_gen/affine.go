package main

import (
	"io"
	"text/template"
)

type affineGen struct{ D int64 }

func (gen affineGen) Gen(w io.Writer) error {
	if err := affineTmpl.Execute(w, &gen); err != nil {
		return err
	}
	if err := affineTRSTmpl.Execute(w, &gen); err != nil {
		return err
	}
	return nil
}

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

func (A {{$AffineD}}[T]) Scale(a T) {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: A.M.Scale(float32(a)),
		T: A.T.Scale(a),
	}
}

func (A {{$AffineD}}[T]) Mul(B {{$AffineD}}[T]) {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		M: A.M.Mul(B.M),
		T: A.T.Add(Matvec{{.D}}(A.M.Convert[T](), B.T)),
	}
}

/*
func (A {{$AffineD}}[T]) Inv() {{$AffineD}}[T] {
	panic("not implemented")
}
*/
`))

var affineTRSTmpl = template.Must(template.New("affineTRS").Parse(`
{{$AffineD := printf "Affine%d" .D}}
{{$AffineDTRS := printf "Affine%dTRS" .D}}

type {{$AffineDTRS}}[T constraints.Float] struct {
	T Vec{{.D}}[T]
	R Rot{{.D}}
	S Mat{{.D}}x{{.D}}Uf32
}

type (
	Affine{{.D}}TRSf32 = {{$AffineDTRS}}[float32]
	Affine{{.D}}TRSf64 = {{$AffineDTRS}}[float64]
)

func Affine{{.D}}TRSOne[T constraints.Float]() {{$AffineDTRS}}[T] {
	return {{$AffineDTRS}}[T]{
		R: Rot{{.D}}One(),
		S: Mat{{.D}}x{{.D}}UOne[float32](),
	}
}

func Affine{{.D}}DecomposeTRS[T constraints.Float](A {{$AffineD}}[T]) {{$AffineDTRS}}[T] {
	// TODO: don't assume the linear part is only rotation, properly extract the
	// scaling and shearing too.
	Q := A.M
	R := Mat{{.D}}x{{.D}}UOne[float32]()

	return {{$AffineDTRS}}[T]{
		T: A.T,
		R: Rot{{.D}}FromMat(Q),
		S: R,
	}
}

func (trs {{$AffineDTRS}}[T]) Affine() {{$AffineD}}[T] {
	return {{$AffineD}}[T]{
		// TODO: special case R.Mat() by S.Mat() for more :b:erf?
		M: trs.R.Mat().Mul(trs.S.Mat()),
		T: trs.T,
	}
}
`))
