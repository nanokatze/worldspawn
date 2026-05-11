// http://yehar.com/blog/wp-content/uploads/2009/08/deip.pdf
package interpolators

func LagrangeP4O3(yminus1, y0, y1, y2, x float32) float32 {
	c0 := y0
	c1 := y1 - 1/3.0*yminus1 - 1/2.0*y0 - 1/6.0*y2
	c2 := 1/2.0*(yminus1+y1) - y0
	c3 := 1/6.0*(y2-yminus1) + 1/2.0*(y0-y1)
	return ((c3*x+c2)*x+c1)*x + c0
}
