package http

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderTemplateEscapesContexts(t *testing.T) {
	for _, tt := range []struct{ name, template, value, want string }{
		{"text", "<p>{{.}}</p>", "<script>alert(1)</script>", "&lt;script&gt;"},
		{"attribute", `<p title="{{.}}">ok</p>`, `" onmouseover="alert(1)`, "&#34;"},
		{"url", `<a href="{{.}}">link</a>`, "javascript:alert(1)", "#ZgotmplZ"},
		{"script", `<script>const value = {{.}};</script>`, "</script><script>alert(1)</script>", `\u003c/script\u003e`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "page.html"), []byte(tt.template), 0600))
			rec := httptest.NewRecorder()
			RenderTemplate(rec, dir, "page", tt.value)
			require.Equal(t, 200, rec.Code)
			require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
			require.Contains(t, rec.Body.String(), tt.want)
			require.False(t, strings.Contains(rec.Body.String(), tt.value))
		})
	}
}
