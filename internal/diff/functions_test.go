package diff

import (
	"bytes"
	"encoding/json/v2"
	"math"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func u64(v uint64) *uint64 { return &v }

func diagnosticCodes(notes []diagnostic) []string {
	var codes []string
	for _, note := range notes {
		codes = append(codes, note.Code)
	}
	return codes
}

func functionReport(functions ...commonFunction) *commonResult {
	if functions == nil {
		functions = []commonFunction{}
	}
	files := []commonFile{{Functions: &functions}}
	return &commonResult{
		Size: 1000, Packages: map[string]commonPackage{"p": {Name: "p", Size: 500, Files: &files, SubPackages: map[string]commonPackage{}}},
		Coverage: &commonCoverage{Version: 2}, BuildInfo: &debug.BuildInfo{GoVersion: "go1.27.0"},
	}
}

func measured(name string, code, pcln, refs uint64) commonFunction {
	return commonFunction{Name: name, FullName: "p." + name, Type: "function", CodeSize: u64(code), PclnBytes: u64(pcln), ReferencedBytes: u64(refs)}
}

func TestFunctionChangesSurviveEqualPackageAndFileSize(t *testing.T) {
	old := functionReport(measured("alpha", 50, 20, 100), measured("beta", 50, 20, 300))
	next := functionReport(measured("alpha", 50, 20, 300), measured("beta", 50, 20, 100))
	d := newDiffResult(next, old)
	require.Empty(t, d.Packages)
	require.Len(t, d.Functions, 2)
	assert.Equal(t, int64(200), d.Functions[0].ReferencedBytes.Delta)
	assert.Equal(t, int64(-200), d.Functions[1].ReferencedBytes.Delta)
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "function_metrics_overlap")
}

func TestFunctionDiffKnownAddRemoveAndEmptyLists(t *testing.T) {
	d := newDiffResult(functionReport(measured("added", 12, 2, 0)), functionReport(measured("removed", 10, 3, 8)))
	require.Len(t, d.Functions, 2)
	byName := map[string]diffFunction{}
	for _, f := range d.Functions {
		byName[f.Name] = f
	}
	assert.Equal(t, changeTypeAdd, byName["added"].ChangeType)
	assert.Equal(t, &byteChange{From: 0, To: 12, Delta: 12}, byName["added"].CodeSize)
	assert.Equal(t, changeTypeRemove, byName["removed"].ChangeType)
	assert.Equal(t, int64(-8), byName["removed"].ReferencedBytes.Delta)
	assert.Len(t, newDiffResult(functionReport(), functionReport(measured("f", 2, 1, 0))).Functions, 1)
}

func TestFunctionDiffMissingMetricsAreNotZero(t *testing.T) {
	a := measured("f", 20, 5, 9)
	b := measured("f", 25, 5, 9)
	a.PclnBytes, a.ReferencedBytes = nil, nil
	d := newDiffResult(functionReport(b), functionReport(a))
	require.Len(t, d.Functions, 1)
	assert.Equal(t, int64(5), d.Functions[0].CodeSize.Delta)
	assert.Nil(t, d.Functions[0].PclnBytes)
	assert.Nil(t, d.Functions[0].ReferencedBytes)
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "function_metric_unavailable")
	data, err := json.Marshal(d)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"referenced_bytes":`)
}

func TestFunctionDiffMissingDetailsSuppressPresenceClaims(t *testing.T) {
	for _, raw := range []string{
		`{"packages":{"p":{"name":"p","size":500}}}`,
		`{"packages":{"p":{"name":"p","size":500,"subPackages":{},"files":[{"size":50}]}}}`,
		`{"packages":{"p":{"name":"p","size":500,"subPackages":{},"files":null}}}`,
		`{}`,
	} {
		var legacy commonResult
		require.NoError(t, json.Unmarshal([]byte(raw), &legacy))
		current := functionReport(measured("f", 50, 12, 10))
		assert.Empty(t, newDiffResult(current, &legacy).Functions, raw)
		assert.Empty(t, newDiffResult(&legacy, current).Functions, raw)
	}
}

func TestFunctionDiffMatchesFullNameAcrossLegacyUpgrade(t *testing.T) {
	a := measured("f", 20, 5, 0)
	a.FullName = ""
	b := measured("f", 30, 5, 0)
	d := newDiffResult(functionReport(b), functionReport(a))
	require.Len(t, d.Functions, 1)
	assert.Equal(t, changeTypeChange, d.Functions[0].ChangeType)
	assert.Equal(t, "p.f", d.Functions[0].FullName)
	assert.Equal(t, int64(10), d.Functions[0].CodeSize.Delta)
}

func TestFunctionDiffFullNamesDistinguishGenericInstances(t *testing.T) {
	a, b := measured("generic", 20, 5, 0), measured("generic", 30, 5, 0)
	a.FullName, b.FullName = "p.generic[go.shape.int]", "p.generic[go.shape.string]"
	nextA := a
	nextA.CodeSize = u64(40)
	d := newDiffResult(functionReport(nextA, b), functionReport(a, b))
	require.Len(t, d.Functions, 1)
	assert.Equal(t, a.FullName, d.Functions[0].FullName)
	assert.Equal(t, int64(20), d.Functions[0].CodeSize.Delta)
}

func TestFunctionDiffIdentityCollisionsAreExplicit(t *testing.T) {
	a, b := measured("f", 20, 5, 0), measured("f", 30, 5, 0)
	for _, legacy := range []bool{false, true} {
		x, y := a, b
		if legacy {
			x.FullName, y.FullName = "", ""
		}
		d := newDiffResult(functionReport(b), functionReport(x, y))
		assert.Empty(t, d.Functions, "collisions must not become false additions")
		assert.Contains(t, diagnosticCodes(d.Diagnostics), "function_identity_collision")
	}
	a.FullName = ""
	b.FullName = "p.f[shape.one]"
	c := b
	c.FullName = "p.f[shape.two]"
	d := newDiffResult(functionReport(b, c), functionReport(a))
	assert.Empty(t, d.Functions)
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "function_identity_collision")
}

func TestFunctionDiffCollisionPreservesDistinctFullNameMatch(t *testing.T) {
	intA, intB := measured("g", 10, 0, 0), measured("g", 11, 0, 0)
	intA.FullName, intB.FullName = "p.g[int]", "p.g[int]"
	oldString, newString := measured("g", 20, 0, 0), measured("g", 40, 0, 0)
	oldString.FullName, newString.FullName = "p.g[string]", "p.g[string]"
	d := newDiffResult(functionReport(intA, newString), functionReport(intA, intB, oldString))
	require.Len(t, d.Functions, 1, "ambiguous int instances must not hide the unique string identity")
	assert.Equal(t, "p.g[string]", d.Functions[0].FullName)
	assert.Equal(t, changeTypeChange, d.Functions[0].ChangeType)
	assert.Equal(t, int64(20), d.Functions[0].CodeSize.Delta)
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "function_identity_collision")
}

func TestFunctionDiffRecursesSubpackages(t *testing.T) {
	a, b := functionReport(measured("f", 20, 5, 0)), functionReport(measured("f", 25, 5, 0))
	emptyFiles := []commonFile{}
	for _, r := range []*commonResult{a, b} {
		child := r.Packages["p"]
		child.Name = "module/p"
		r.Packages = map[string]commonPackage{"module": {Name: "module", Size: 500, Files: &emptyFiles, SubPackages: map[string]commonPackage{"module/p": child}}}
	}
	d := newDiffResult(b, a)
	require.Len(t, d.Functions, 1)
	assert.Equal(t, "module/p", d.Functions[0].Package)
}

func TestFunctionDiffAvoidsSignedOverflow(t *testing.T) {
	a, b := measured("f", 20, 5, 0), measured("f", math.MaxUint64, 6, 0)
	d := newDiffResult(functionReport(b), functionReport(a))
	require.Len(t, d.Functions, 1)
	assert.Nil(t, d.Functions[0].CodeSize)
	assert.Equal(t, int64(1), d.Functions[0].PclnBytes.Delta)
}

func TestBuildContextComparisonsAndLegacyInformation(t *testing.T) {
	a, b := functionReport(), functionReport()
	a.BuildInfo.Settings = []debug.BuildSetting{{Key: "GOARCH", Value: "amd64"}, {Key: "vcs.revision", Value: "old"}}
	b.BuildInfo.GoVersion = "go1.28.0"
	b.BuildInfo.Settings = []debug.BuildSetting{{Key: "GOARCH", Value: "arm64"}, {Key: "vcs.revision", Value: "new"}}
	b.BuildInfo.Deps = []*debug.Module{{Path: "example.org/lib", Version: "v1.2.0"}}
	b.Coverage.Version = 3
	d := newDiffResult(b, a)
	require.Len(t, d.BuildChanges, 4)
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "build_context_changed")
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "coverage_version_changed")
	a.BuildInfo = nil
	d = newDiffResult(b, a)
	assert.Empty(t, d.BuildChanges, "missing build info is not an old empty configuration")
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "build_info_unavailable")
}

func TestEmptyGoVersionsRemainUnknown(t *testing.T) {
	a, b := functionReport(), functionReport()
	a.BuildInfo.GoVersion, b.BuildInfo.GoVersion = "", ""
	d := newDiffResult(b, a)
	assert.Empty(t, d.BuildChanges)
	assert.Contains(t, diagnosticCodes(d.Diagnostics), "build_info_incomplete")
}

func TestFunctionTextShowsUnknownComponentsAndNotes(t *testing.T) {
	a, b := measured("f", 20, 5, 1), measured("f", 30, 5, 1)
	a.ReferencedBytes = nil
	d := newDiffResult(functionReport(b), functionReport(a))
	var buf bytes.Buffer
	require.NoError(t, text(&d, &buf))
	assert.Contains(t, buf.String(), "Function changes")
	assert.Contains(t, buf.String(), "p.f")
	assert.Contains(t, buf.String(), "unavailable")
	assert.Contains(t, buf.String(), "function_metric_unavailable")
}
