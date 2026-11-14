package diff

import (
	"bytes"
	"encoding/json/v2"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func symbolReport(symbols ...commonSymbol) *commonResult {
	if symbols == nil {
		symbols = []commonSymbol{}
	}
	r := functionReport()
	pkg := r.Packages["p"]
	pkg.Symbols = &symbols
	r.Packages["p"] = pkg
	return r
}

func sizedSymbol(name string, addr, size uint64) commonSymbol {
	return commonSymbol{Name: name, Type: "data", Addr: u64(addr), Size: u64(size)}
}

func TestSymbolChangesSurviveEqualPackageSize(t *testing.T) {
	a := symbolReport(sizedSymbol("p.Alpha.string", 100, 2048), sizedSymbol("p.Beta.string", 10000, 8192))
	b := symbolReport(sizedSymbol("p.Alpha.string", 1000, 8192), sizedSymbol("p.Beta.string", 20000, 2048))
	d := newDiffResult(b, a)
	require.Empty(t, d.Packages)
	require.Empty(t, d.Functions)
	require.Len(t, d.Symbols, 2)
	require.Equal(t, int64(6144), d.Symbols[0].Size.Delta)
	require.Equal(t, int64(-6144), d.Symbols[1].Size.Delta)
	require.Contains(t, diagnosticCodes(d.Diagnostics), "symbol_metrics_overlap")
	var output bytes.Buffer
	require.NoError(t, text(&d, &output))
	require.Contains(t, output.String(), "Symbol changes")
	require.Contains(t, output.String(), "p.Alpha.string")
}

func TestSymbolDuplicateEvidenceDoesNotBecomeAmbiguous(t *testing.T) {
	a := sizedSymbol("asset", 100, 20)
	d := newDiffResult(symbolReport(sizedSymbol("asset", 800, 25)), symbolReport(a, a))
	require.Len(t, d.Symbols, 1)
	require.Equal(t, int64(5), d.Symbols[0].Size.Delta)
	require.NotContains(t, diagnosticCodes(d.Diagnostics), "symbol_identity_collision")
}

func TestSymbolCollisionDoesNotHideOtherIdentities(t *testing.T) {
	for _, address := range []*uint64{u64(200), nil} {
		a, b := sizedSymbol("ambiguous", 100, 20), sizedSymbol("ambiguous", 200, 20)
		b.Addr = address
		d := newDiffResult(symbolReport(a, sizedSymbol("unique", 900, 30)), symbolReport(a, b, sizedSymbol("unique", 300, 20)))
		require.Len(t, d.Symbols, 1)
		require.Equal(t, "unique", d.Symbols[0].Name)
		require.Equal(t, int64(10), d.Symbols[0].Size.Delta)
		require.Contains(t, diagnosticCodes(d.Diagnostics), "symbol_identity_collision")
	}
}

func TestSymbolMissingListsDoNotImplyDeletion(t *testing.T) {
	full := symbolReport(sizedSymbol("asset", 100, 20))
	legacy := functionReport()
	d := newDiffResult(legacy, full)
	require.Empty(t, d.Symbols)
	require.Contains(t, diagnosticCodes(d.Diagnostics), "symbol_details_unavailable")
	d = newDiffResult(symbolReport(), full)
	require.Len(t, d.Symbols, 1)
	require.Equal(t, changeTypeRemove, d.Symbols[0].ChangeType)
	require.Equal(t, int64(-20), d.Symbols[0].Size.Delta)
}

func TestSymbolUnknownAndOversizedMetricsAreNotZero(t *testing.T) {
	for _, size := range []*uint64{nil, u64(math.MaxUint64)} {
		a := sizedSymbol("asset", 100, 20)
		a.Size = size
		d := newDiffResult(symbolReport(sizedSymbol("asset", 100, 30)), symbolReport(a))
		require.Empty(t, d.Symbols)
		require.Contains(t, diagnosticCodes(d.Diagnostics), "symbol_metric_unavailable")
	}
}

func TestSymbolsCompareRecursiveAndMissingPackagesConservatively(t *testing.T) {
	a, b := symbolReport(), symbolReport()
	child := commonPackage{Name: "p/child", SubPackages: map[string]commonPackage{}, Symbols: &[]commonSymbol{sizedSymbol("asset", 100, 20)}}
	a.Packages["p"].SubPackages["p/child"] = child
	d := newDiffResult(b, a)
	require.Len(t, d.Symbols, 1)
	require.Equal(t, "p/child", d.Symbols[0].Package)
	require.Equal(t, changeTypeRemove, d.Symbols[0].ChangeType)
	b.Packages["legacy"] = commonPackage{Name: "legacy"}
	d = newDiffResult(b, a)
	require.Empty(t, d.Symbols, "an incomplete tree must not imply missing package inventory is empty")
}

func TestSymbolUnknownAddressDuplicatesStayAmbiguous(t *testing.T) {
	a := sizedSymbol("asset", 100, 20)
	a.Addr = nil
	d := newDiffResult(symbolReport(sizedSymbol("asset", 500, 40)), symbolReport(a, a))
	require.Empty(t, d.Symbols)
	require.Contains(t, diagnosticCodes(d.Diagnostics), "symbol_identity_collision")
}

func TestSymbolMissingTypeDoesNotBecomeRemoveAndAdd(t *testing.T) {
	a := sizedSymbol("asset", 100, 20)
	a.Type = ""
	d := newDiffResult(symbolReport(sizedSymbol("asset", 500, 30), sizedSymbol("valid", 600, 40)),
		symbolReport(a, sizedSymbol("valid", 200, 30)))
	require.Len(t, d.Symbols, 1)
	require.Equal(t, "valid", d.Symbols[0].Name)
	require.Contains(t, diagnosticCodes(d.Diagnostics), "symbol_details_unavailable")
}

func TestTextBoundsRepeatedCollisionsWithoutDroppingJSON(t *testing.T) {
	r := diffResult{}
	for range 100 {
		r.Diagnostics = append(r.Diagnostics, diagnostic{Code: "symbol_identity_collision", Identity: "asset", Message: "ambiguous"})
	}
	var output bytes.Buffer
	require.NoError(t, text(&r, &output))
	require.Equal(t, 10, strings.Count(output.String(), "asset: ambiguous"))
	require.Contains(t, output.String(), "90 additional ambiguous identities")
	data, err := json.Marshal(r)
	require.NoError(t, err)
	var decoded diffResult
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Len(t, decoded.Diagnostics, 100)
}
