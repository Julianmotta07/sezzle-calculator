package api

// White-box tests for branches that are hard or impossible to reach through
// the public handler: defensive fallbacks that should never fire in practice.

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAPIErrorMessage(t *testing.T) {
	err := badRequest(CodeMissingOperand, `operation "add" requires field "b"`)
	if got := err.Error(); got != `operation "add" requires field "b"` {
		t.Errorf("Error() = %q", got)
	}
}

func TestFromCalcErrorUnknownErrorIsInternal(t *testing.T) {
	got := fromCalcError(errors.New("unexpected"))
	if got.status != http.StatusInternalServerError || got.code != CodeInternal {
		t.Errorf("got %d %s, want 500 %s", got.status, got.code, CodeInternal)
	}
	if strings.Contains(got.message, "unexpected") {
		t.Errorf("internal error details leaked to client: %q", got.message)
	}
}

func TestWriteJSONEncodeFailure(t *testing.T) {
	// Channels cannot be JSON-encoded; status and headers are already sent,
	// so writeJSON can only log the failure.
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, make(chan int))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestDescribeKind(t *testing.T) {
	cases := []struct {
		typ  reflect.Type
		want string
	}{
		{reflect.TypeFor[float64](), "a finite number"},
		{reflect.TypeFor[*float64](), "a finite number"},
		{reflect.TypeFor[float32](), "a finite number"},
		{reflect.TypeFor[string](), "a string"},
		{reflect.TypeFor[bool](), "of type bool"},
		{reflect.TypeFor[**int](), "of type int"},
	}
	for _, tc := range cases {
		if got := describeKind(tc.typ); got != tc.want {
			t.Errorf("describeKind(%v) = %q, want %q", tc.typ, got, tc.want)
		}
	}
}

// failingReader makes every Read fail, simulating a broken connection.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestDecodeErrorFallback(t *testing.T) {
	// A read error that is not EOF, syntax, type or size related.
	_, err := json.NewDecoder(failingReader{}).Token()
	got := decodeError(err)
	if got.status != http.StatusBadRequest || got.code != CodeInvalidJSON {
		t.Errorf("got %d %s, want 400 %s", got.status, got.code, CodeInvalidJSON)
	}
}

// unseekableFS returns files that cannot Seek, which http.ServeContent needs.
type unseekableFS struct{}

func (unseekableFS) Open(string) (fs.File, error) { return unseekableFile{}, nil }

type unseekableFile struct{}

func (unseekableFile) Stat() (fs.FileInfo, error) { return fakeInfo{}, nil }
func (unseekableFile) Read([]byte) (int, error)   { return 0, errors.New("unused") }
func (unseekableFile) Close() error               { return nil }

// statFailFS returns files whose Stat fails.
type statFailFS struct{}

func (statFailFS) Open(string) (fs.File, error) { return statFailFile{}, nil }

type statFailFile struct{ unseekableFile }

func (statFailFile) Stat() (fs.FileInfo, error) { return nil, errors.New("stat failed") }

type fakeInfo struct{}

func (fakeInfo) Name() string       { return "index.html" }
func (fakeInfo) Size() int64        { return 0 }
func (fakeInfo) Mode() fs.FileMode  { return 0o644 }
func (fakeInfo) ModTime() time.Time { return time.Time{} }
func (fakeInfo) IsDir() bool        { return false }
func (fakeInfo) Sys() any           { return nil }

func TestServeFileUnservableFileIsInternalError(t *testing.T) {
	cases := []struct {
		name string
		fsys fs.FS
	}{
		{"file cannot seek", unseekableFS{}},
		{"file cannot stat", statFailFS{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			serveFile(rec, httptest.NewRequest(http.MethodGet, "/", nil), tc.fsys, "index.html")

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code != CodeInternal {
				t.Errorf("body = %s, want %s error", rec.Body.String(), CodeInternal)
			}
		})
	}
}
