package webui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandlerInjectsVersionIntoIndexAndRouteFallback(t *testing.T) {
	root := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(
				`<meta name="prx-version" content="__PRX_VERSION__"><meta name="prx-demo" content="__PRX_DEMO__">`,
			),
		},
		"app.js": &fstest.MapFile{Data: []byte("application")},
	}
	handler := newHandler(root, `1.2.3&test`, true)

	for _, requestPath := range []string{"/", "/features/example"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, requestPath, nil)
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d", requestPath, response.Code)
		}
		if got := response.Body.String(); !strings.Contains(got, `content="1.2.3&amp;test"`) {
			t.Fatalf("GET %s body = %q", requestPath, got)
		}
		if got := response.Body.String(); !strings.Contains(got, `name="prx-demo" content="true"`) {
			t.Fatalf("GET %s body = %q", requestPath, got)
		}
		if got := response.Header().Get("Content-Security-Policy"); got == "" {
			t.Fatalf("GET %s has no content security policy", requestPath)
		}
	}
}

func TestHandlerServesStaticAsset(t *testing.T) {
	root := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(versionPlaceholder + demoPlaceholder)},
		"app.js":     &fstest.MapFile{Data: []byte("application")},
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/app.js", nil)
	newHandler(root, "1.2.3", false).ServeHTTP(response, request)

	if got, want := response.Body.String(), "application"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestHandlerReportsMissingBuild(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	newHandler(fstest.MapFS{}, "1.2.3", false).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
	body, err := io.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(body), "web UI is not built; run make web-build\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
