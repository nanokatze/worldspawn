package main

import (
	"gioui.org/layout"
)

type Navigation struct {
	stack []layout.Widget
}

func (nav *Navigation) Push(w layout.Widget) {
	nav.stack = append(nav.stack, w)
}

func (nav *Navigation) Pop() {
	nav.stack = nav.stack[:len(nav.stack)-1]
}

func (nav *Navigation) Layout(gtx layout.Context) layout.Dimensions {
	w := nav.stack[len(nav.stack)-1]
	return w(gtx)
}

type menu struct {
	_ int
}

func (w *menu) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Dimensions{}
}
