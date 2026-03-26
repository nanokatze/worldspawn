package main

import "text/template"

type rotGen struct{ D int64 }

var rotTmpl = template.Must(template.New("rot").Parse(`
{{$RotD := printf "Rot%d" .D}}

type {{$RotD}} struct {
}

func Rot{{.D}}One() {{$RotD}} {
	panic("not implemented")
}

func Rot{{.D}}InPlane(???, θ float32) {{$RotD}} {
	s, c := math.Sincos(float64(θ / 2))
	return {{$RotD}}{}
}

func Rot{{.D}}FromMat(M Mat{{.D}}x{{.D}}f32) {{$RotD}} {
	panic("not implemented")
}

func (R {{$RotD}}) Renormalize() {{$RotD}} {
	panic("not implemented")
}

func (R {{$RotD}}) Inv() {{$RotD}} {
	panic("not implemented")
}

func (R1 {{$RotD}}) Mul(R2 {{$RotD}}) {{$RotD}} {
	panic("not implemented")
}

// TODO: make this generic once generic methods land
func (R {{$RotD}}) Rotate32(v Vec{{.D}}f32) Vec{{.D}}f32 {
	panic("not implemented")
}
`))
