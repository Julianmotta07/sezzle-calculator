package api

import "github.com/Julianmotta07/sezzle-calculator/backend/internal/calculator"

// operation describes one supported calculation. Name, Operands and
// Description are exposed by GET /api/v1/operations; apply is internal.
type operation struct {
	Name        string `json:"name"`
	Operands    int    `json:"operands"`
	Description string `json:"description"`
	apply       func(a, b float64) (float64, error)
}

// operations is the single source of truth for what the API supports.
// A slice (not a map) keeps the listing order stable.
var operations = []operation{
	{Name: "add", Operands: 2, Description: "a + b", apply: calculator.Add},
	{Name: "subtract", Operands: 2, Description: "a - b", apply: calculator.Subtract},
	{Name: "multiply", Operands: 2, Description: "a * b", apply: calculator.Multiply},
	{Name: "divide", Operands: 2, Description: "a / b", apply: calculator.Divide},
	{Name: "power", Operands: 2, Description: "a raised to the power b", apply: calculator.Power},
	{Name: "sqrt", Operands: 1, Description: "square root of a", apply: func(a, _ float64) (float64, error) {
		return calculator.Sqrt(a)
	}},
	{Name: "percentage", Operands: 2, Description: "a percent of b (a*b/100)", apply: calculator.Percentage},
}

// operationsByName indexes operations for lookup by name.
var operationsByName = func() map[string]operation {
	m := make(map[string]operation, len(operations))
	for _, op := range operations {
		m[op.Name] = op
	}
	return m
}()
