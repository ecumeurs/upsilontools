package tools

import (
	"math/rand"
	"time"
)

// @spec-link [[mechanic_randomization_helpers]]

// Seed initializes the global random seed using the current system time in nanoseconds.
// This function should be called at application startup to ensure that subsequent
// random number generations are non-deterministic and vary between sessions.
func Seed() {
	rand.Seed(time.Now().UnixNano())
}

// SeedWith initializes the global random seed using a specific 64-bit integer value.
// This is used for deterministic random number generation, allowing for reproducible
// simulations, test cases, and procedural generation sequences.
func SeedWith(s int64) {
	rand.Seed(s)
}

var randInt = rand.Intn

// IntRange represents a numeric interval with a start and end value.
// It provides a convenient way to encapsulate a range of integers and perform
// operations like generating a random value within that specific boundary.
type IntRange struct {
	Start int
	End   int
}

// NewIntRange creates a new IntRange instance with the specified boundaries.
// The start and end values define the inclusive and exclusive limits of the range.
// It returns an initialized IntRange struct ready for use in randomization logic.
func NewIntRange(start, end int) IntRange {
	return IntRange{start, end}
}

// TesterRand allows overriding the default random integer generator for testing purposes.
// This is critical for injecting mock implementations or predictable sequences during
// unit testing, ensuring that the engine behavior can be verified deterministically.
func TesterRand(rdm func(int) int) {
	randInt = rdm
}

// Random returns a random integer within the receiver's range [Start, End).
// It utilizes the current global randInt generator. If the start and end values
// are identical, it returns the start value directly, avoiding division by zero.
func (r IntRange) Random() int {
	if r.Start == r.End {
		return r.Start
	}
	return r.Start + randInt(r.End-r.Start)
}

// RandomInt returns a random integer between min (inclusive) and max (exclusive).
// This is a standalone utility function that does not require an IntRange instance.
// It handles edge cases where min might be greater than or equal to max.
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return min + randInt(max-min)
}

// @spec-link [[mechanic_math_core_utils]]

// Abs returns the absolute value of the provided integer x.
// If x is negative, it returns -x; otherwise, it returns x.
// This is a standard mathematical utility for calculating magnitudes.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// @spec-link [[mechanic_spatial_distance_calculations]]

// Distance calculates the Manhattan distance between two 2D points (x, y) and (x2, y2).
// Manhattan distance is defined as the sum of the absolute differences of their coordinates.
// This is commonly used in grid-based pathfinding and tactical positioning logic.
func Distance(x, y, x2, y2 int) int {
	return Abs(x-x2) + Abs(y-y2)
}

// Distance3D calculates the Manhattan distance between two 3D points (x, y, z) and (x2, y2, z2).
// Similar to the 2D version, it sums the absolute differences across all three spatial dimensions.
// This is useful for volumetric grid calculations or multi-level battle arena logic.
func Distance3D(x, y, z, x2, y2, z2 int) int {
	return Abs(x-x2) + Abs(y-y2) + Abs(z-z2)
}

// FloatDistance calculates the Manhattan distance between two 2D float coordinates.
// It provides a floating-point precision alternative to the standard integer Distance function.
// This is typically used for continuous space calculations or sub-pixel positioning.
func FloatDistance(x, y, x2, y2 float64) float64 {
	return AbsFloat(x-x2) + AbsFloat(y-y2)
}

// AbsFloat returns the absolute value of the provided float64 x.
// It mirrors the behavior of the integer Abs function but for floating-point values.
// This ensures that the result is always a non-negative float64 value.
func AbsFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// FloatDistance3D calculates the Manhattan distance between two 3D float coordinates.
// It combines floating-point precision with three-dimensional spatial distance calculation.
// This is essential for high-fidelity physics or complex movement simulations in 3D.
func FloatDistance3D(x, y, z, x2, y2, z2 float64) float64 {
	return AbsFloat(x-x2) + AbsFloat(y-y2) + AbsFloat(z-z2)
}

// LinearProgressionAt computes a linear interpolation of y based on x within defined bounds.
// It calculates the y-coordinate on a straight line between (min_x, min_y) and (max_x, max_y) at x.
// This is frequently used for scaling game mechanics, stat progressions, or UI gradients.
func LinearProgressionAt(min_y, max_y, min_x, max_x, x int) int {
	return min_y + (max_y-min_y)*(x-min_x)/(max_x-min_x)
}

// Min returns the smaller of the two provided integers x and y.
// This is a basic utility for constraining values within an upper bound or selection logic.
// It compares the two values and returns the one that is numerically smaller.
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Max returns the larger of the two provided integers x and y.
// This is a basic utility for constraining values within a lower bound or selection logic.
// It compares the two values and returns the one that is numerically larger.
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
