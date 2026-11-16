package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Julianmotta07/sezzle-calculator/backend/internal/calculator"
)

// Error codes returned in the "code" field of every error response.
// They are part of the public API contract, so clients can switch on them.
const (
	CodeInvalidJSON       = "INVALID_JSON"
	CodeUnknownField      = "UNKNOWN_FIELD"
	CodePayloadTooLarge   = "PAYLOAD_TOO_LARGE"
	CodeMissingOperation  = "MISSING_OPERATION"
	CodeUnknownOperation  = "UNKNOWN_OPERATION"
	CodeMissingOperand    = "MISSING_OPERAND"
	CodeUnexpectedOperand = "UNEXPECTED_OPERAND"
	CodeDivisionByZero    = "DIVISION_BY_ZERO"
	CodeNegativeSqrt      = "NEGATIVE_SQRT"
	CodeNonFiniteResult   = "NON_FINITE_RESULT"
	CodeNotFound          = "NOT_FOUND"
	CodeMethodNotAllowed  = "METHOD_NOT_ALLOWED"
	CodeInternal          = "INTERNAL_ERROR"
)

// ErrorResponse is the JSON shape shared by all error responses:
//
//	{"error":{"code":"DIVISION_BY_ZERO","message":"..."}}
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody holds the machine-readable code and human-readable message.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// apiError is an error that already knows its HTTP status and error code.
type apiError struct {
	status  int
	code    string
	message string
}

func (e *apiError) Error() string { return e.message }

// badRequest builds a 400 apiError.
func badRequest(code, message string) *apiError {
	return &apiError{status: http.StatusBadRequest, code: code, message: message}
}

// fromCalcError maps a calculator sentinel error to a 422 apiError.
// Unknown errors become a 500 so they are never silently misreported.
func fromCalcError(err error) *apiError {
	switch {
	case errors.Is(err, calculator.ErrDivisionByZero):
		return &apiError{http.StatusUnprocessableEntity, CodeDivisionByZero, "cannot divide by zero"}
	case errors.Is(err, calculator.ErrNegativeSqrt):
		return &apiError{http.StatusUnprocessableEntity, CodeNegativeSqrt, "cannot take the square root of a negative number"}
	case errors.Is(err, calculator.ErrNonFinite):
		return &apiError{http.StatusUnprocessableEntity, CodeNonFiniteResult, "result is not a finite number (overflow or undefined)"}
	default:
		slog.Error("unexpected calculator error", "err", err)
		return &apiError{http.StatusInternalServerError, CodeInternal, "internal server error"}
	}
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

// writeError writes e using the shared error shape.
func writeError(w http.ResponseWriter, e *apiError) {
	writeJSON(w, e.status, ErrorResponse{Error: ErrorBody{Code: e.code, Message: e.message}})
}
