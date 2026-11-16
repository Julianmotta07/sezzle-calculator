package calculator

import (
	"errors"
	"math"
	"testing"
)

// epsilon is the tolerance used when comparing floating-point results.
const epsilon = 1e-9

var (
	maxF = math.MaxFloat64
	inf  = math.Inf(1)
	nan  = math.NaN()
)

// binaryCase describes one test for a two-operand operation.
type binaryCase struct {
	name    string
	a, b    float64
	want    float64
	wantErr error
}

// runBinary runs a table of cases against a two-operand operation.
func runBinary(t *testing.T, op func(a, b float64) (float64, error), cases []binaryCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := op(tc.a, tc.b)
			check(t, got, err, tc.want, tc.wantErr)
		})
	}
}

// check asserts the result of an operation against the expected value or error.
func check(t *testing.T, got float64, err error, want float64, wantErr error) {
	t.Helper()
	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
		if got != 0 {
			t.Errorf("result on error = %v, want 0", got)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approxEqual(got, want) {
		t.Errorf("result = %v, want %v", got, want)
	}
}

// approxEqual reports whether a and b are equal within a relative tolerance.
func approxEqual(a, b float64) bool {
	diff := math.Abs(a - b)
	if diff <= epsilon {
		return true
	}
	return diff <= epsilon*math.Max(math.Abs(a), math.Abs(b))
}

func TestAdd(t *testing.T) {
	runBinary(t, Add, []binaryCase{
		{name: "positives", a: 2, b: 3, want: 5},
		{name: "negatives", a: -2, b: -3, want: -5},
		{name: "mixed signs", a: -7, b: 3, want: -4},
		{name: "decimals", a: 0.1, b: 0.2, want: 0.3},
		{name: "zero", a: 0, b: 0, want: 0},
		{name: "identity", a: 42.5, b: 0, want: 42.5},
		{name: "overflow", a: maxF, b: maxF, wantErr: ErrNonFinite},
		{name: "infinite input", a: inf, b: 1, wantErr: ErrNonFinite},
		{name: "NaN input", a: nan, b: 1, wantErr: ErrNonFinite},
	})
}

func TestSubtract(t *testing.T) {
	runBinary(t, Subtract, []binaryCase{
		{name: "positives", a: 10, b: 4, want: 6},
		{name: "negative result", a: 4, b: 10, want: -6},
		{name: "negatives", a: -5, b: -3, want: -2},
		{name: "decimals", a: 1.5, b: 0.25, want: 1.25},
		{name: "zero", a: 0, b: 0, want: 0},
		{name: "self", a: 3.3, b: 3.3, want: 0},
		{name: "overflow", a: -maxF, b: maxF, wantErr: ErrNonFinite},
		{name: "inf minus inf", a: inf, b: inf, wantErr: ErrNonFinite},
	})
}

func TestMultiply(t *testing.T) {
	runBinary(t, Multiply, []binaryCase{
		{name: "positives", a: 6, b: 7, want: 42},
		{name: "negative and positive", a: -6, b: 7, want: -42},
		{name: "negatives", a: -6, b: -7, want: 42},
		{name: "decimals", a: 0.5, b: 0.5, want: 0.25},
		{name: "by zero", a: 123.45, b: 0, want: 0},
		{name: "by one", a: -9.75, b: 1, want: -9.75},
		{name: "overflow", a: maxF, b: 2, wantErr: ErrNonFinite},
		{name: "inf times zero", a: inf, b: 0, wantErr: ErrNonFinite},
	})
}

func TestDivide(t *testing.T) {
	runBinary(t, Divide, []binaryCase{
		{name: "exact", a: 10, b: 2, want: 5},
		{name: "fractional result", a: 1, b: 3, want: 1.0 / 3.0},
		{name: "negative divisor", a: 9, b: -3, want: -3},
		{name: "negatives", a: -9, b: -3, want: 3},
		{name: "decimals", a: 0.75, b: 0.25, want: 3},
		{name: "zero dividend", a: 0, b: 5, want: 0},
		{name: "by zero", a: 1, b: 0, wantErr: ErrDivisionByZero},
		{name: "zero by zero", a: 0, b: 0, wantErr: ErrDivisionByZero},
		{name: "by negative zero", a: 1, b: math.Copysign(0, -1), wantErr: ErrDivisionByZero},
		{name: "overflow", a: maxF, b: 0.5, wantErr: ErrNonFinite},
		{name: "NaN input", a: nan, b: 2, wantErr: ErrNonFinite},
	})
}

func TestPower(t *testing.T) {
	runBinary(t, Power, []binaryCase{
		{name: "square", a: 3, b: 2, want: 9},
		{name: "cube of negative", a: -2, b: 3, want: -8},
		{name: "even power of negative", a: -2, b: 2, want: 4},
		{name: "negative exponent", a: 2, b: -2, want: 0.25},
		{name: "fractional exponent", a: 9, b: 0.5, want: 3},
		{name: "decimal base", a: 1.5, b: 2, want: 2.25},
		{name: "zero exponent", a: 7, b: 0, want: 1},
		{name: "zero to zero", a: 0, b: 0, want: 1},
		{name: "zero base", a: 0, b: 5, want: 0},
		{name: "zero to negative power", a: 0, b: -1, wantErr: ErrNonFinite},
		{name: "negative base fractional exponent", a: -8, b: 1.0 / 3.0, wantErr: ErrNonFinite},
		{name: "overflow", a: 10, b: 400, wantErr: ErrNonFinite},
	})
}

func TestSqrt(t *testing.T) {
	cases := []struct {
		name    string
		x       float64
		want    float64
		wantErr error
	}{
		{name: "perfect square", x: 16, want: 4},
		{name: "irrational", x: 2, want: math.Sqrt2},
		{name: "decimal", x: 0.25, want: 0.5},
		{name: "zero", x: 0, want: 0},
		{name: "negative zero", x: math.Copysign(0, -1), want: 0},
		{name: "large", x: 1e300, want: 1e150},
		{name: "negative", x: -4, wantErr: ErrNegativeSqrt},
		{name: "small negative", x: -0.0001, wantErr: ErrNegativeSqrt},
		{name: "infinite input", x: inf, wantErr: ErrNonFinite},
		{name: "NaN input", x: nan, wantErr: ErrNonFinite},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Sqrt(tc.x)
			check(t, got, err, tc.want, tc.wantErr)
		})
	}
}

func TestPercentage(t *testing.T) {
	runBinary(t, Percentage, []binaryCase{
		{name: "20 percent of 50", a: 20, b: 50, want: 10},
		{name: "100 percent", a: 100, b: 80, want: 80},
		{name: "over 100 percent", a: 150, b: 40, want: 60},
		{name: "decimal percent", a: 12.5, b: 80, want: 10},
		{name: "negative percent", a: -10, b: 200, want: -20},
		{name: "percent of negative", a: 10, b: -200, want: -20},
		{name: "zero percent", a: 0, b: 999, want: 0},
		{name: "percent of zero", a: 50, b: 0, want: 0},
		{name: "overflow", a: maxF, b: maxF, wantErr: ErrNonFinite},
		{name: "NaN input", a: nan, b: 10, wantErr: ErrNonFinite},
	})
}
