package main

import "github.com/sirupsen/logrus"

/* power增长函数
	y = a*height + b
	a ~= 6.47289*Math.pow(10,12)
	b ~= -4.951*Math.pow(10, 17)
	6.4815*Math.pow(10,12) * height - 5*Math.pow(10, 17)
 */

/* 24FIL释放函数
	y = a*height + b
	a ~= 0.386516702
	b ~= 90458.99279
*/

type Point struct {
	X float64
	Y float64
}

func Points2Func(p1, p2 Point) (a, b float64) {
	// y = ax + b
	logrus.Infof("p1[%f,%f] p2[%f,%f]", p1.X, p1.Y, p2.X, p2.Y)
	diffY := p2.Y-p1.Y
	diffX := p2.X-p1.X
	a = diffY/diffX
	b = p2.Y - a*p2.X
	return
}