package main

import (
	"math"
	"testing"
)

func TestPoints2Func(t *testing.T) {
	type args struct {
		p1 Point
		p2 Point
	}
	tests := []struct {
		name  string
		args  args
		wantA float64
		wantB float64
	}{
		{
			name: "t1",
			args: args{
				p1: Point{
					X: 0,
					Y: 44076857,
				},
				p2: Point{
					X: 82,
					Y: 66874171,
				},
			},
			wantA: 0,
			wantB: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA, gotB := Points2Func(tt.args.p1, tt.args.p2)
			// y = gotA*x+
			t.Logf("y=x*%f+%f", gotA, gotB)
			for i := float64(0); i < 50; i++ {
				val := gotA*i+gotB
				val = val * 130 / 100
				t.Logf("i:%f, %f", i, val)
				t.Logf("%f %f %f %f %f", val*math.Pow(10, -9)*1, val*math.Pow(10, -9)*2,
					val*math.Pow(10, -9)*3, val*math.Pow(10, -9)*4, val*math.Pow(10, -9)*5)
			}
		})
	}

}
