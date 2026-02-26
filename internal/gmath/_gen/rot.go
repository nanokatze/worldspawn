package main

import "text/template"

type rotGen struct{ D int64 }

var rotTmpl = template.Must(template.New("rot").Parse(`
{{$rotD := printf "Rot%d" .D}}

type {{$rotD}} struct {
}

func {{$rotD}}InPlane(???, θ float32) {{$rotD}} {
	s, c := math.Sincos(float64(θ / 2))
	return {{$rotD}}{}
}

func (a {{$rotD}}) Inverse() {{$rotD}} {
	panic("not implemented")
}

func (a {{$rotD}}) Mul(b {{$rotD}}) {{$rotD}} {
	panic("not implemented")
}

func (a {{$rotD}}) Rotate32(v Vec{{.D}}) Vec{{.D}} {
	panic("not implemented")
}

func (a {{$rotD}}) Rotate64(v DVec{{.D}}) DVec{{.D}} {
	panic("not implemented")
}
`))
