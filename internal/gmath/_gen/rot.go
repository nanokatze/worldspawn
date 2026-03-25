package main

import "text/template"

type rotGen struct{ D int64 }

var rotTmpl = template.Must(template.New("rot").Parse(`
{{$RotD := printf "Rot%d" .D}}

type {{$RotD}} struct {
}

func {{$RotD}}InPlane(???, θ float32) {{$RotD}} {
	s, c := math.Sincos(float64(θ / 2))
	return {{$RotD}}{}
}

func (a {{$RotD}}) Inverse() {{$RotD}} {
	panic("not implemented")
}

func (a {{$RotD}}) Mul(b {{$RotD}}) {{$RotD}} {
	panic("not implemented")
}

func (a {{$RotD}}) Rotate32(v Vec{{.D}}) Vec{{.D}} {
	panic("not implemented")
}

func (a {{$RotD}}) Rotate64(v DVec{{.D}}) DVec{{.D}} {
	panic("not implemented")
}
`))
