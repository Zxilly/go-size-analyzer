//go:build embed && !js && !wasm

package printer

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestHTMLKeepsSourcePathsInsideJSON(t *testing.T) {
	const path = "source</script><script>gsaInjected()</script>.go"
	pkg := entity.NewPackage()
	pkg.Name = "main"
	pkg.Files = []*entity.File{{FilePath: path}}
	var output bytes.Buffer
	require.NoError(t, HTML(&result.Result{Name: "example", Packages: entity.PackageMap{"main": pkg}}, &output))
	doc, err := html.Parse(&output)
	require.NoError(t, err)
	var embedded string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			isData := false
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "data" {
					isData = true
				}
			}
			if n.FirstChild != nil {
				if isData {
					embedded = n.FirstChild.Data
				} else {
					require.NotContains(t, n.FirstChild.Data, "gsaInjected()")
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	require.NotEmpty(t, embedded)
	var decoded result.Result
	require.NoError(t, json.Unmarshal([]byte(embedded), &decoded))
	require.Equal(t, path, decoded.Packages["main"].Files[0].FilePath)
	require.False(t, strings.Contains(embedded, "</script>"))
}
