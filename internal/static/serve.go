package static

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// Dist is populated before release builds via scripts/sync-frontend-dist.mjs.
//
//go:embed all:dist
var Dist embed.FS

// Handler serves the embedded Vite production bundle with SPA fallback.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(Dist, "dist")
	if err != nil {
		return nil, err
	}
	return &fileServer{fs: http.FS(sub)}, nil
}

type fileServer struct {
	fs http.FileSystem
}

func (s *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	clean := path.Clean(r.URL.Path)
	if clean == "/" {
		s.serveIndex(w, r)
		return
	}

	file, err := s.fs.Open(strings.TrimPrefix(clean, "/"))
	if err == nil {
		defer file.Close()
		if stat, statErr := file.Stat(); statErr == nil && !stat.IsDir() {
			if strings.HasPrefix(clean, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.FileServer(s.fs).ServeHTTP(w, r)
			return
		}
	}

	s.serveIndex(w, r)
}

func (s *fileServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, mustOpen(s.fs, "index.html"))
}

func mustOpen(fs http.FileSystem, name string) http.File {
	file, err := fs.Open(name)
	if err != nil {
		panic("embedded frontend missing index.html: run pnpm build and sync-frontend-dist")
	}
	return file
}

// APIPrefix reports whether the path belongs to backend routes (not static).
func APIPrefix(urlPath string) bool {
	return strings.HasPrefix(urlPath, "/v1/") ||
		urlPath == "/v1" ||
		strings.HasPrefix(urlPath, "/api/") ||
		urlPath == "/api" ||
		strings.HasPrefix(urlPath, "/diagnostics/") ||
		urlPath == "/healthz" ||
		urlPath == "/readyz"
}

// Wrap serves embedded static assets for non-API GET requests and delegates API
// traffic to the gateway handler.
func Wrap(api http.Handler, static http.Handler) http.Handler {
	if static == nil {
		return api
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if APIPrefix(r.URL.Path) {
			api.ServeHTTP(w, r)
			return
		}

		static.ServeHTTP(w, r)
	})
}
