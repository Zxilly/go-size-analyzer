package entity

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFunctionReferenceSummaryUsesPhysicalUnion(t *testing.T) {
	fn := &Function{CodeSize: 32}
	fn.Init()
	fn.disasm.Insert(&Addr{AddrPos: &AddrPos{Addr: 100, Size: 20}, Role: AddrRoleReference, SourceType: AddrSourceStaticRead, Boundary: "instruction"})
	fn.disasm.Insert(&Addr{AddrPos: &AddrPos{Addr: 110, Size: 20}, Role: AddrRoleReference, SourceType: AddrSourceDisasm, Boundary: "heuristic"})
	fn.disasm.Insert(&Addr{AddrPos: &AddrPos{Addr: 200, Size: 20}, Role: AddrRoleDefinition})
	fn.disasm.Insert(&Addr{AddrPos: &AddrPos{Addr: 300, Size: 20}, Role: AddrRoleReference})
	sections := []*Section{
		{Name: ".data", Addr: 100, Size: 100, Offset: 10, FileSize: 25, ContentType: SectionContentData},
		{Name: ".bss", Addr: 300, Size: 100, OnlyInMemory: true, ContentType: SectionContentData},
		{Name: "virtual", Addr: 300, Size: 100, FileSize: 100, VirtualSection: true, ContentType: SectionContentData},
		{Name: ".debug", Addr: 300, Size: 100, FileSize: 100, Debug: true, ContentType: SectionContentData},
	}
	fn.SetReferenceSummary(sections, true, true)
	require.NotNil(t, fn.ReferencedBytes)
	require.Equal(t, uint64(25), *fn.ReferencedBytes, "overlap counted once, zero-fill tail excluded")
	require.Len(t, fn.ReferenceEvidence, 2)
	require.Equal(t, FileRange{Offset: 10, Size: 20}, fn.ReferenceEvidence[0].FileRange)
	require.Equal(t, FileRange{Offset: 20, Size: 15}, fn.ReferenceEvidence[1].FileRange)
	require.Equal(t, uint64(32), fn.Size(), "references must not inflate code totals")
	fn.SetReferenceSummary(sections, false, true)
	require.Equal(t, uint64(25), *fn.ReferencedBytes)
	require.Empty(t, fn.ReferenceEvidence)
	require.Zero(t, fn.ReferenceEvidenceCount)
	fn.SetReferenceSummary(sections, true, false)
	require.Nil(t, fn.ReferencedBytes, "unavailable must differ from measured zero")
	fn.Wasm = true
	fn.SetReferenceSummary(sections, true, true)
	require.Nil(t, fn.ReferencedBytes, "native mapping must not fabricate wasm reference totals")
}

func TestFunctionReferenceEvidenceBoundedDeterministicAndDeduplicated(t *testing.T) {
	fn := &Function{}
	fn.Init()
	for i := uint64(0); i < 20; i++ {
		a := &Addr{AddrPos: &AddrPos{Addr: 100 + i*4, Size: 2}, Role: AddrRoleReference, SourceType: AddrSourceStaticRead}
		fn.disasm.Insert(a)
		fn.disasm.Insert(a)
	}
	sections := []*Section{{Name: ".data", Addr: 100, Size: 100, Offset: 20, FileSize: 100, ContentType: SectionContentData}}
	fn.SetReferenceSummary(sections, true, true)
	require.Equal(t, uint64(40), *fn.ReferencedBytes)
	require.Equal(t, 20, fn.ReferenceEvidenceCount)
	require.Len(t, fn.ReferenceEvidence, 8)
	for i, e := range fn.ReferenceEvidence {
		require.Equal(t, uint64(20+i*4), e.Offset)
	}
	first := append([]FunctionEvidence(nil), fn.ReferenceEvidence...)
	fn.SetReferenceSummary(sections, true, true)
	require.Equal(t, first, fn.ReferenceEvidence)
}

func TestFunctionReferenceJSONUnknownAndZero(t *testing.T) {
	fn := &Function{}
	data, err := json.Marshal(fn)
	require.NoError(t, err)
	require.NotContains(t, string(data), "referenced_bytes")
	require.NotContains(t, string(data), "reference_evidence_count")
	fn.SetReferenceSummary(nil, false, true)
	data, err = json.Marshal(fn)
	require.NoError(t, err)
	require.Contains(t, string(data), `"referenced_bytes":0`)
}
