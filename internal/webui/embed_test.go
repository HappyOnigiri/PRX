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
				`<meta name="prx-version" content="__PRX_VERSION__"><meta name="prx-demo" content="__PRX_DEMO__">` +
					`<meta name="prx-demo-session" content="__PRX_DEMO_SESSION__">`,
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
		if got := response.Body.String(); strings.Contains(got, demoSessionPlaceholder) {
			t.Fatalf("GET %s body = %q", requestPath, got)
		}
		if got := response.Header().Get("Content-Security-Policy"); got == "" {
			t.Fatalf("GET %s has no content security policy", requestPath)
		}
	}
}

// WebUI は閉じた demo の警告をこの ID に結び付けるので、起動ごとに変わることを確かめる。
func TestHandlerInjectsFreshDemoSessionPerStart(t *testing.T) {
	root := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<meta name="prx-demo-session" content="__PRX_DEMO_SESSION__">`),
		},
	}

	bodies := make([]string, 0, 2)
	for range 2 {
		handler := newHandler(root, "1.2.3", true)
		reloads := make([]string, 0, 2)
		for range 2 {
			response := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			handler.ServeHTTP(response, request)
			reloads = append(reloads, response.Body.String())
		}
		if reloads[0] != reloads[1] {
			t.Fatalf("reload changed the demo session: %q and %q", reloads[0], reloads[1])
		}
		bodies = append(bodies, reloads[0])
	}

	if bodies[0] == bodies[1] {
		t.Fatalf("both starts served the same demo session: %q", bodies[0])
	}
	if strings.Contains(bodies[0], demoSessionPlaceholder) {
		t.Fatalf("body = %q", bodies[0])
	}
}

func TestHandlerServesStaticAsset(t *testing.T) {
	root := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(versionPlaceholder + demoPlaceholder + demoSessionPlaceholder),
		},
		"app.js": &fstest.MapFile{Data: []byte("application")},
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
