package tools

import (
	"math/rand"
)

type IntRange struct {
	Start int
	End   int
}

func NewIntRange(start, end int) IntRange {
	return IntRange{start, end}
}

func (r IntRange) Random() int {
	return r.Start + rand.Intn(r.End-r.Start)
}

func RandomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Distance(x, y, x2, y2 int) int {
	return Abs(x-x2) + Abs(y-y2)
}

func Distance3D(x, y, z, x2, y2, z2 int) int {
	return Abs(x-x2) + Abs(y-y2) + Abs(z-z2)
}

func FloatDistance(x, y, x2, y2 float64) float64 {
	return AbsFloat(x-x2) + AbsFloat(y-y2)
}

func AbsFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func FloatDistance3D(x, y, z, x2, y2, z2 float64) float64 {
	return AbsFloat(x-x2) + AbsFloat(y-y2) + AbsFloat(z-z2)
}

func LinearProgressionAt(min_y, max_y, min_x, max_x, x int) int {
	return min_y + (max_y-min_y)*(x-min_x)/(max_x-min_x)
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
