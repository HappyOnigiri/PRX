package webui

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed all:dist
var assets embed.FS

const (
	versionPlaceholder = "__PRX_VERSION__"
	demoPlaceholder    = "__PRX_DEMO__"
)

func Handler(version string, demo bool) http.Handler {
	root, _ := fs.Sub(assets, "dist")
	return newHandler(root, version, demo)
}

func newHandler(root fs.FS, version string, demo bool) http.Handler {
	files := http.FileServer(http.FS(root))
	index, indexErr := fs.ReadFile(root, "index.html")
	index = bytes.ReplaceAll(index, []byte(versionPlaceholder), []byte(html.EscapeString(version)))
	index = bytes.ReplaceAll(index, []byte(demoPlaceholder), []byte(fmt.Sprintf("%t", demo)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; connect-src 'self'",
		)
		if indexErr != nil {
			http.Error(w, "web UI is not built; run make web-build", http.StatusServiceUnavailable)
			return
		}
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean == "." || clean == "index.html" {
			http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
			return
		}
		if _, err := fs.Stat(root, clean); err != nil {
			http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
			return
		}
		files.ServeHTTP(w, r)
	})
}
