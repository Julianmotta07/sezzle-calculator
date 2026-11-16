package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

// maxBodyBytes caps the size of a calculate request body. A valid request is
// well under 200 bytes, so 4 KiB is generous while still bounding memory use.
const maxBodyBytes = 4 << 10

// calculateRequest is the body of POST /api/v1/calculate. Operands are
// pointers so a missing field can be told apart from an explicit 0.
type calculateRequest struct {
	Operation string   `json:"operation"`
	A         *float64 `json:"a"`
	B         *float64 `json:"b"`
}

// calculateResponse is the success body of POST /api/v1/calculate.
type calculateResponse struct {
	Operation string  `json:"operation"`
	Result    float64 `json:"result"`
}

// operationsResponse is the body of GET /api/v1/operations.
type operationsResponse struct {
	Operations []operation `json:"operations"`
}

// handleHealth reports that the server is up.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleOperations lists the supported operations and their operand counts.
func handleOperations(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, operationsResponse{Operations: operations})
}

// handleCalculate decodes, validates and runs one calculation.
func handleCalculate(w http.ResponseWriter, r *http.Request) {
	req, apiErr := decodeCalculateRequest(w, r)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	op, apiErr := resolveOperation(req)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}

	var b float64
	if req.B != nil {
		b = *req.B
	}
	result, err := op.apply(*req.A, b)
	if err != nil {
		writeError(w, fromCalcError(err))
		return
	}
	writeJSON(w, http.StatusOK, calculateResponse{Operation: op.Name, Result: result})
}

// decodeCalculateRequest reads exactly one JSON object from a size-limited
// body, rejecting unknown fields and trailing data.
func decodeCalculateRequest(w http.ResponseWriter, r *http.Request) (calculateRequest, *apiError) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req calculateRequest
	if err := dec.Decode(&req); err != nil {
		return req, decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if tooLarge := asTooLarge(err); tooLarge != nil {
			return req, tooLarge
		}
		return req, badRequest(CodeInvalidJSON, "request body must contain a single JSON object")
	}
	return req, nil
}

// decodeError turns a json.Decoder error into a client-facing apiError.
func decodeError(err error) *apiError {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	if tooLarge := asTooLarge(err); tooLarge != nil {
		return tooLarge
	}
	switch {
	case errors.Is(err, io.EOF):
		return badRequest(CodeInvalidJSON, "request body is empty")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return badRequest(CodeInvalidJSON, "request body contains malformed JSON")
	case errors.As(err, &syntaxErr):
		return badRequest(CodeInvalidJSON, fmt.Sprintf("request body contains malformed JSON at position %d", syntaxErr.Offset))
	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return badRequest(CodeInvalidJSON, "request body must be a JSON object")
		}
		return badRequest(CodeInvalidJSON, fmt.Sprintf("field %q must be %s", typeErr.Field, describeKind(typeErr.Type)))
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		// encoding/json has no typed error for this case; the prefix is stable.
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return badRequest(CodeUnknownField, "request body contains unknown field "+field)
	default:
		return badRequest(CodeInvalidJSON, "request body is not valid JSON")
	}
}

// asTooLarge returns a 413 apiError if err came from exceeding maxBodyBytes.
func asTooLarge(err error) *apiError {
	var maxErr *http.MaxBytesError
	if !errors.As(err, &maxErr) {
		return nil
	}
	return &apiError{
		status:  http.StatusRequestEntityTooLarge,
		code:    CodePayloadTooLarge,
		message: fmt.Sprintf("request body must not exceed %d bytes", maxBodyBytes),
	}
}

// describeKind names the JSON type a Go field expects, for error messages.
func describeKind(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Float32, reflect.Float64:
		return "a finite number"
	case reflect.String:
		return "a string"
	default:
		return "of type " + t.String()
	}
}

// resolveOperation checks the operation name and that exactly the operands
// it needs are present.
func resolveOperation(req calculateRequest) (operation, *apiError) {
	if req.Operation == "" {
		return operation{}, badRequest(CodeMissingOperation, `field "operation" is required`)
	}
	op, ok := operationsByName[req.Operation]
	if !ok {
		return operation{}, badRequest(CodeUnknownOperation,
			fmt.Sprintf("unknown operation %q; see GET /api/v1/operations", req.Operation))
	}
	if req.A == nil {
		return operation{}, badRequest(CodeMissingOperand,
			fmt.Sprintf(`operation %q requires field "a"`, op.Name))
	}
	if op.Operands == 2 && req.B == nil {
		return operation{}, badRequest(CodeMissingOperand,
			fmt.Sprintf(`operation %q requires field "b"`, op.Name))
	}
	if op.Operands == 1 && req.B != nil {
		return operation{}, badRequest(CodeUnexpectedOperand,
			fmt.Sprintf(`operation %q takes only field "a"; remove field "b"`, op.Name))
	}
	return op, nil
}
