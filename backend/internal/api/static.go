package api

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
)

// staticHandler serves the built single-page app from dir:
//   - existing files are served as-is; Vite's hashed files under assets/
//     are cached for a year, since their names change when their content does;
//   - any other path (including directories) gets index.html, so client-side
//     routes work on reload and directory listings are never exposed;
//   - only GET and HEAD are allowed.
func staticHandler(dir string) http.Handler {
	fsys := os.DirFS(dir)
	wrongMethod := methodNotAllowed("GET, HEAD")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			wrongMethod.ServeHTTP(w, r)
			return
		}

		// path.Clean on a rooted path removes any "..", so name stays inside dir
		// (os.DirFS also rejects names that escape it).
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if info, err := fs.Stat(fsys, name); err == nil && !info.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			serveFile(w, r, fsys, name)
			return
		}

		// index.html must always be revalidated so new deploys are picked up.
		w.Header().Set("Cache-Control", "no-cache")
		serveFile(w, r, fsys, "index.html")
	})
}

// serveFile writes one file from fsys. http.ServeContent sets Content-Type
// from the extension and handles HEAD, Range and conditional requests.
func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) {
	f, err := fsys.Open(name)
	if err != nil {
		handleNotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	content, seekable := f.(io.ReadSeeker)
	if err != nil || !seekable {
		slog.Error("cannot serve static file", "name", name, "err", err)
		writeError(w, &apiError{http.StatusInternalServerError, CodeInternal, "internal server error"})
		return
	}
	http.ServeContent(w, r, name, info.ModTime(), content)
}
