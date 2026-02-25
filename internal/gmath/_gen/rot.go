package main

import "text/template"

type rotGen struct{ D int64 }

var rotTmpl = template.Must(template.New("rot").Parse(`
{{$rotD := printf "Rot%d" .D}}

type {{$rotD}} struct {

}

`))
