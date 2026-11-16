// Package calculator provides pure arithmetic operations on float64 values.
//
// Every operation either returns a finite result and a nil error, or returns
// 0 and one of the sentinel errors declared below. Callers should compare
// errors with errors.Is.
package calculator

import (
	"errors"
	"math"
)

// Sentinel errors returned by the operations in this package.
var (
	// ErrDivisionByZero is returned when the divisor is zero.
	ErrDivisionByZero = errors.New("division by zero")

	// ErrNegativeSqrt is returned when taking the square root of a negative number.
	ErrNegativeSqrt = errors.New("square root of a negative number")

	// ErrNonFinite is returned when a result is NaN or ±Inf, for example on
	// overflow or when an input is already non-finite.
	ErrNonFinite = errors.New("result is not a finite number")
)

// Add returns a + b.
func Add(a, b float64) (float64, error) {
	return finite(a + b)
}

// Subtract returns a - b.
func Subtract(a, b float64) (float64, error) {
	return finite(a - b)
}

// Multiply returns a * b.
func Multiply(a, b float64) (float64, error) {
	return finite(a * b)
}

// Divide returns a / b. It returns ErrDivisionByZero when b is zero.
func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, ErrDivisionByZero
	}
	return finite(a / b)
}

// Power returns base raised to exp. Results that are not real and finite,
// such as a negative base with a fractional exponent or 0 raised to a
// negative power, return ErrNonFinite.
func Power(base, exp float64) (float64, error) {
	return finite(math.Pow(base, exp))
}

// Sqrt returns the square root of x. It returns ErrNegativeSqrt when x < 0.
func Sqrt(x float64) (float64, error) {
	if x < 0 {
		return 0, ErrNegativeSqrt
	}
	return finite(math.Sqrt(x))
}

// Percentage returns a percent of b, that is a*b/100.
// For example, Percentage(20, 50) returns 10.
func Percentage(a, b float64) (float64, error) {
	return finite(a * b / 100)
}

// finite returns v unchanged if it is a finite number, or ErrNonFinite otherwise.
func finite(v float64) (float64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, ErrNonFinite
	}
	return v, nil
}
