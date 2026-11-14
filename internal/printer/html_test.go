//go:build embed && !js && !wasm

package printer

import (
	"bytes"
	"encoding/json/jsontext"
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

func TestHTMLUsesWebProjectionWhileFullJSONPreservesDetails(t *testing.T) {
	zero := uint64(0)
	fn := &entity.Function{
		Name: "f", FullName: "p.f", CodeSize: 32, ReferencedBytes: &zero,
		PclnSize:               entity.PclnSymbolSize{Name: 4, Header: 8},
		Ranges:                 []entity.AddrPos{{Addr: 100, Size: 32, Type: entity.AddrTypeText}},
		ReferenceEvidenceCount: 11,
		ReferenceEvidence:      []entity.FunctionEvidence{{FileRange: entity.FileRange{Offset: 400, Size: 8}, Source: "static_read", Role: "reference", Boundary: "instruction"}},
	}
	pkg := entity.NewPackage()
	pkg.Name = "p"
	pkg.Files = []*entity.File{{FilePath: "source.go", Functions: []*entity.Function{fn}}}
	r := &result.Result{Packages: entity.PackageMap{"p": pkg}, Coverage: &entity.FileCoverage{
		Version:    2,
		Attributed: 32, Recognized: 5, Unclassified: 1, Shared: 3, BySource: map[string]uint64{"pclntab": 32}, Notes: []string{"kept"},
		Regions: []entity.FileRegion{{FileRange: entity.FileRange{Offset: 10, Size: 32}, Class: "attributed"}},
	}}
	var web, full bytes.Buffer
	require.NoError(t, HTML(r, &web))
	doc, err := html.Parse(&web)
	require.NoError(t, err)
	var payload string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "data" && n.FirstChild != nil {
					payload = n.FirstChild.Data
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	require.NotEmpty(t, payload)
	var value struct {
		Packages map[string]struct {
			Files []struct {
				Functions []map[string]jsontext.Value `json:"functions"`
			} `json:"files"`
		} `json:"packages"`
		Coverage entity.FileCoverage `json:"coverage"`
	}
	require.NoError(t, json.Unmarshal([]byte(payload), &value))
	function := value.Packages["p"].Files[0].Functions[0]
	require.NotContains(t, function, "pcln_size")
	require.NotContains(t, function, "ranges")
	require.JSONEq(t, "12", string(function["pcln_bytes"]))
	require.JSONEq(t, "0", string(function["referenced_bytes"]))
	require.JSONEq(t, "11", string(function["reference_evidence_count"]))
	require.Contains(t, function, "reference_evidence")
	require.Empty(t, value.Coverage.Regions)
	require.Equal(t, r.Coverage.Attributed, value.Coverage.Attributed)
	require.Equal(t, r.Coverage.Notes, value.Coverage.Notes)
	require.NoError(t, JSON(r, &full, &JSONOption{}))
	require.Contains(t, full.String(), `"ranges":`)
	require.Contains(t, full.String(), `"pcln_size":`)
	require.Contains(t, full.String(), `"regions":`)
	require.Len(t, r.Coverage.Regions, 1)
}
