package marshaler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"strings"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestWebFunctionProjectionPreservesMetricsAndSampledEvidence(t *testing.T) {
	zero := uint64(0)
	fn := entity.Function{
		Name: "f", FullName: "p.f", Type: "method", Receiver: "T", Addr: 100, CodeSize: 32,
		ReferencedBytes: &zero, ReferenceEvidenceCount: 19,
		PclnSize:          entity.PclnSymbolSize{Name: 4, Header: 12, PCData: map[string]int{"pc": 8}},
		Ranges:            []entity.AddrPos{{Addr: 100, Size: 32, Type: entity.AddrTypeText}},
		ReferenceEvidence: []entity.FunctionEvidence{{FileRange: entity.FileRange{Offset: 500, Size: 16}, Section: ".rodata", Source: "static_read", Role: "reference", Boundary: "instruction"}},
	}
	full, err := json.Marshal(fn)
	require.NoError(t, err)
	encoded, err := json.Marshal(fn, json.WithMarshalers(GetFileDetailMarshaler()))
	require.NoError(t, err)
	var value map[string]jsontext.Value
	require.NoError(t, json.Unmarshal(encoded, &value))
	require.NotContains(t, value, "pcln_size")
	require.NotContains(t, value, "ranges")
	require.JSONEq(t, "0", string(value["referenced_bytes"]))
	require.JSONEq(t, "19", string(value["reference_evidence_count"]))
	var pcln uint64
	require.NoError(t, json.Unmarshal(value["pcln_bytes"], &pcln))
	require.Equal(t, fn.PclnBytes(), pcln, "legacy pcln components must be projected to the same total")
	var evidence []entity.FunctionEvidence
	require.NoError(t, json.Unmarshal(value["reference_evidence"], &evidence))
	require.Equal(t, fn.ReferenceEvidence, evidence)
	for _, field := range []string{"name", "full_name", "type", "receiver", "addr", "code_size"} {
		require.Contains(t, value, field)
	}
	unchanged, err := json.Marshal(fn)
	require.NoError(t, err)
	require.Equal(t, full, unchanged, "web serialization must not mutate full CLI data")
	require.Contains(t, string(full), `"pcln_size":`)
	require.Contains(t, string(full), `"ranges":`)
	fn.ReferencedBytes = nil
	encoded, err = json.Marshal(fn, json.WithMarshalers(GetFileDetailMarshaler()))
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"referenced_bytes":`)
}

func TestWebCoverageProjectionLeavesFullAndCompactLedgerIntact(t *testing.T) {
	coverage := entity.FileCoverage{
		Version: 2, Attributed: 12, Recognized: 4, Unclassified: 8, Shared: 2,
		BySource: map[string]uint64{"symbol": 12}, Notes: []string{"test note"},
		Regions: []entity.FileRegion{{FileRange: entity.FileRange{Offset: 0, Size: 12}, Class: "attributed", Owners: []string{"p"}}},
	}
	full, err := json.Marshal(coverage)
	require.NoError(t, err)
	web, err := json.Marshal(coverage, json.WithMarshalers(GetFileDetailMarshaler()))
	require.NoError(t, err)
	var projected entity.FileCoverage
	require.NoError(t, json.Unmarshal(web, &projected))
	expected := coverage
	expected.Regions = nil
	require.Equal(t, expected, projected)
	require.Len(t, coverage.Regions, 1)
	compact, err := json.Marshal(coverage, json.WithMarshalers(GetFileCompactMarshaler()))
	require.NoError(t, err)
	require.JSONEq(t, string(full), string(compact), "compact CLI omits function details, not requested coverage regions")
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestFileMarshalersPropagateWriterFailure(t *testing.T) {
	failure := errors.New("write failed")
	file := entity.File{FilePath: strings.Repeat("source", 10000), Functions: []*entity.Function{{Name: "f"}}}
	for _, marshalers := range []*json.Marshalers{GetFileCompactMarshaler(), GetFileDetailMarshaler()} {
		require.NotPanics(t, func() {
			err := json.MarshalWrite(failingWriter{err: failure}, file, json.WithMarshalers(marshalers))
			require.ErrorIs(t, err, failure)
		})
	}
}
