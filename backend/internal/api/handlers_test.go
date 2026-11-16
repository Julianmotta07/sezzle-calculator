package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Julianmotta07/sezzle-calculator/backend/internal/api"
)

const testOrigin = "http://localhost:5173"

// newHandler returns the API handler configured as in development.
func newHandler() http.Handler {
	return api.NewHandler(api.Config{AllowedOrigin: testOrigin})
}

// do sends a request through the handler and returns the recorded response.
func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeStrict decodes a JSON response body into v, failing on unknown fields
// so the tests also pin down the exact response shape.
func decodeStrict(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	dec := json.NewDecoder(rec.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
}

// assertError checks status code and the shared error shape.
func assertError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, wantStatus, rec.Body.String())
	}
	var got api.ErrorResponse
	decodeStrict(t, rec, &got)
	if got.Error.Code != wantCode {
		t.Errorf("error code = %q, want %q", got.Error.Code, wantCode)
	}
	if got.Error.Message == "" {
		t.Error("error message is empty")
	}
}

func TestHealth(t *testing.T) {
	rec := do(t, newHandler(), http.MethodGet, "/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string]string
	decodeStrict(t, rec, &got)
	if got["status"] != "ok" || len(got) != 1 {
		t.Errorf("body = %v, want {status: ok}", got)
	}
}

func TestOperations(t *testing.T) {
	rec := do(t, newHandler(), http.MethodGet, "/api/v1/operations", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got struct {
		Operations []struct {
			Name        string `json:"name"`
			Operands    int    `json:"operands"`
			Description string `json:"description"`
		} `json:"operations"`
	}
	decodeStrict(t, rec, &got)

	want := map[string]int{
		"add": 2, "subtract": 2, "multiply": 2, "divide": 2,
		"power": 2, "sqrt": 1, "percentage": 2,
	}
	if len(got.Operations) != len(want) {
		t.Fatalf("got %d operations, want %d", len(got.Operations), len(want))
	}
	for _, op := range got.Operations {
		n, ok := want[op.Name]
		if !ok {
			t.Errorf("unexpected operation %q", op.Name)
			continue
		}
		if op.Operands != n {
			t.Errorf("%s operands = %d, want %d", op.Name, op.Operands, n)
		}
		if op.Description == "" {
			t.Errorf("%s has empty description", op.Name)
		}
	}
}

func TestCalculateSuccess(t *testing.T) {
	cases := []struct {
		name string
		body string
		op   string
		want float64
	}{
		{"add", `{"operation":"add","a":2,"b":3}`, "add", 5},
		{"subtract negative result", `{"operation":"subtract","a":2,"b":5}`, "subtract", -3},
		{"multiply decimals", `{"operation":"multiply","a":1.5,"b":-4}`, "multiply", -6},
		{"divide", `{"operation":"divide","a":10,"b":2}`, "divide", 5},
		{"divide zero dividend", `{"operation":"divide","a":0,"b":7}`, "divide", 0},
		{"power", `{"operation":"power","a":2,"b":10}`, "power", 1024},
		{"sqrt", `{"operation":"sqrt","a":81}`, "sqrt", 9},
		{"sqrt of zero", `{"operation":"sqrt","a":0}`, "sqrt", 0},
		{"percentage", `{"operation":"percentage","a":20,"b":50}`, "percentage", 10},
		{"explicit zero operand", `{"operation":"add","a":0,"b":0}`, "add", 0},
	}
	h := newHandler()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/api/v1/calculate", tc.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
			}
			var got struct {
				Operation string  `json:"operation"`
				Result    float64 `json:"result"`
			}
			decodeStrict(t, rec, &got)
			if got.Operation != tc.op || got.Result != tc.want {
				t.Errorf("got %+v, want {Operation:%s Result:%v}", got, tc.op, tc.want)
			}
		})
	}
}

func TestCalculateErrors(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		// Malformed or unacceptable bodies: 400.
		{"empty body", ``, 400, api.CodeInvalidJSON},
		{"truncated JSON", `{"operation":"add",`, 400, api.CodeInvalidJSON},
		{"syntax error", `{operation: add}`, 400, api.CodeInvalidJSON},
		{"not an object", `[1, 2]`, 400, api.CodeInvalidJSON},
		{"operand is a string", `{"operation":"add","a":"ten","b":2}`, 400, api.CodeInvalidJSON},
		{"operation is a number", `{"operation":5,"a":1,"b":2}`, 400, api.CodeInvalidJSON},
		{"operand out of float64 range", `{"operation":"add","a":1e400,"b":2}`, 400, api.CodeInvalidJSON},
		{"trailing data", `{"operation":"add","a":1,"b":2} {}`, 400, api.CodeInvalidJSON},
		{"unknown field", `{"operation":"add","a":1,"b":2,"c":3}`, 400, api.CodeUnknownField},
		{"missing operation", `{"a":1,"b":2}`, 400, api.CodeMissingOperation},
		{"unknown operation", `{"operation":"modulo","a":1,"b":2}`, 400, api.CodeUnknownOperation},
		{"operation is case-sensitive", `{"operation":"ADD","a":1,"b":2}`, 400, api.CodeUnknownOperation},
		{"missing a", `{"operation":"add","b":2}`, 400, api.CodeMissingOperand},
		{"missing b", `{"operation":"divide","a":2}`, 400, api.CodeMissingOperand},
		{"null b", `{"operation":"divide","a":2,"b":null}`, 400, api.CodeMissingOperand},
		{"sqrt missing a", `{"operation":"sqrt"}`, 400, api.CodeMissingOperand},
		{"sqrt with b", `{"operation":"sqrt","a":4,"b":2}`, 400, api.CodeUnexpectedOperand},

		// Body over the size limit: 413.
		{"body too large", `{"operation":"add","a":1,"b":2,"pad":"` + strings.Repeat("x", 5000) + `"}`,
			413, api.CodePayloadTooLarge},
		{"trailing data too large", `{"operation":"add","a":1,"b":2}` + strings.Repeat(" ", 5000) + `{}`,
			413, api.CodePayloadTooLarge},

		// Valid requests the math rejects: 422.
		{"division by zero", `{"operation":"divide","a":1,"b":0}`, 422, api.CodeDivisionByZero},
		{"sqrt of negative", `{"operation":"sqrt","a":-9}`, 422, api.CodeNegativeSqrt},
		{"multiply overflow", `{"operation":"multiply","a":1e308,"b":10}`, 422, api.CodeNonFiniteResult},
		{"zero to negative power", `{"operation":"power","a":0,"b":-1}`, 422, api.CodeNonFiniteResult},
		{"negative base fractional power", `{"operation":"power","a":-8,"b":0.5}`, 422, api.CodeNonFiniteResult},
	}
	h := newHandler()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/api/v1/calculate", tc.body)
			assertError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestRoutingErrors(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
		wantAllow  string
	}{
		{"unknown path", http.MethodGet, "/nope", 404, api.CodeNotFound, ""},
		{"unknown api path", http.MethodPost, "/api/v1/calculate/extra", 404, api.CodeNotFound, ""},
		{"GET calculate", http.MethodGet, "/api/v1/calculate", 405, api.CodeMethodNotAllowed, "POST"},
		{"POST health", http.MethodPost, "/health", 405, api.CodeMethodNotAllowed, "GET, HEAD"},
		{"DELETE operations", http.MethodDelete, "/api/v1/operations", 405, api.CodeMethodNotAllowed, "GET, HEAD"},
	}
	h := newHandler()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, "")
			assertError(t, rec, tc.wantStatus, tc.wantCode)
			if got := rec.Header().Get("Allow"); got != tc.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tc.wantAllow)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	cases := []struct {
		name          string
		allowed       string
		method        string
		origin        string
		preflight     bool
		wantStatus    int
		wantACAO      string
		wantACMethods bool
	}{
		{"preflight from allowed origin", testOrigin, http.MethodOptions, testOrigin, true, 204, testOrigin, true},
		{"preflight from other origin", testOrigin, http.MethodOptions, "http://evil.example", true, 204, "", false},
		{"request from allowed origin", testOrigin, http.MethodGet, testOrigin, false, 200, testOrigin, false},
		{"request from other origin", testOrigin, http.MethodGet, "http://evil.example", false, 200, "", false},
		{"request without origin", testOrigin, http.MethodGet, "", false, 200, "", false},
		{"wildcard allows any origin", "*", http.MethodGet, "http://anything.example", false, 200, "*", false},
		{"plain OPTIONS is not a preflight", testOrigin, http.MethodOptions, testOrigin, false, 405, testOrigin, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := api.NewHandler(api.Config{AllowedOrigin: tc.allowed})
			req := httptest.NewRequest(tc.method, "/health", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.preflight {
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
				req.Header.Set("Access-Control-Request-Headers", "Content-Type")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tc.wantACAO {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tc.wantACAO)
			}
			if got := rec.Header().Get("Access-Control-Allow-Methods") != ""; got != tc.wantACMethods {
				t.Errorf("Access-Control-Allow-Methods present = %v, want %v", got, tc.wantACMethods)
			}
			if got := rec.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want Origin", got)
			}
		})
	}
}
