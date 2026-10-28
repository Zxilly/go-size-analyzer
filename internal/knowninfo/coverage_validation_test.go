package knowninfo

import (
	"math"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

type validationClaim struct {
	pkg        string
	addr, size uint64
	typ        entity.AddrType
	source     entity.AddrSourceType
}

func TestValidateNativeCoverageDoesNotBuildCoverageCaches(t *testing.T) {
	k := validationInfo([]validationClaim{
		{pkg: "p", addr: 100, size: 20, typ: entity.AddrTypeData},
		{pkg: "p/child", addr: 110, size: 20, typ: entity.AddrTypeData},
		{pkg: "text", addr: 0, size: 50, typ: entity.AddrTypeText},
	})
	previous := &entity.CoveragePart{Pos: &entity.AddrPos{Addr: 500, Size: 1}}
	k.Coverage = entity.AddrCoverage{previous}
	require.NoError(t, k.ValidateNativeCoverage())
	require.Len(t, k.Coverage, 1)
	require.Same(t, previous, k.Coverage[0], "the fast path must not replace global coverage")

	// A prior GetPackageCoverage call would freeze the lazy cache and hide
	// this later addition in the child, its parent, or both.
	child := k.Deps.Trie.Get("p/child")
	addValidationClaim(child, validationClaim{addr: 200, size: 10, typ: entity.AddrTypeData})
	parent := k.Deps.Trie.Get("p")
	for _, pkg := range []*entity.Package{parent, child} {
		coverage, err := pkg.GetPackageCoverage()
		require.NoError(t, err)
		found := false
		for _, part := range coverage {
			found = found || part.Pos.Addr == 200 && part.Pos.Size == 10
		}
		require.True(t, found, "%s coverage was cached by validation", pkg.Name)
	}
}

func TestValidateNativeCoveragePreservesLegacyFallback(t *testing.T) {
	for _, tc := range []struct {
		name    string
		claims  []validationClaim
		outcome string
	}{
		{
			name: "different types in separate packages",
			claims: []validationClaim{
				{pkg: "a", addr: 0, size: 10, typ: entity.AddrTypeData},
				{pkg: "b", addr: 5, size: 10, typ: entity.AddrTypeText},
			},
			outcome: "error",
		},
		{
			name: "parent child type conflict",
			claims: []validationClaim{
				{pkg: "p", addr: 0, size: 10, typ: entity.AddrTypeData},
				{pkg: "p/child", addr: 5, size: 10, typ: entity.AddrTypeText},
			},
			outcome: "error",
		},
		{
			name: "disasm extension taints package union before global merge",
			claims: []validationClaim{
				{pkg: "a", addr: 0, size: 10, typ: entity.AddrTypeData},
				{pkg: "a", addr: 5, size: 15, typ: entity.AddrTypeData, source: entity.AddrSourceDisasm},
				{pkg: "b", addr: 3, size: 1, typ: entity.AddrTypeText},
			},
			outcome: "success",
		},
		{
			name: "contained disasm does not taint package union",
			claims: []validationClaim{
				{pkg: "a", addr: 0, size: 20, typ: entity.AddrTypeData},
				{pkg: "a", addr: 5, size: 5, typ: entity.AddrTypeData, source: entity.AddrSourceDisasm},
				{pkg: "b", addr: 8, size: 1, typ: entity.AddrTypeText},
			},
			outcome: "error",
		},
		{
			name: "incoming disasm is dropped",
			claims: []validationClaim{
				{pkg: "a", addr: 0, size: 20, typ: entity.AddrTypeText},
				{pkg: "b", addr: 5, size: 5, typ: entity.AddrTypeData, source: entity.AddrSourceDisasm},
			},
			outcome: "success",
		},
		{
			name: "two disasm types are dropped",
			claims: []validationClaim{
				{pkg: "a", addr: 0, size: 10, typ: entity.AddrTypeText, source: entity.AddrSourceDisasm},
				{pkg: "b", addr: 5, size: 10, typ: entity.AddrTypeData, source: entity.AddrSourceDisasm},
			},
			outcome: "success",
		},
		{
			name: "static reads are not the legacy disasm exception",
			claims: []validationClaim{
				{pkg: "a", addr: 0, size: 10, typ: entity.AddrTypeData, source: entity.AddrSourceStaticRead},
				{pkg: "b", addr: 5, size: 10, typ: entity.AddrTypeText},
			},
			outcome: "error",
		},
		{
			name: "empty range inside another type",
			claims: []validationClaim{
				{pkg: "a", addr: 0, size: 10, typ: entity.AddrTypeData},
				{pkg: "b", addr: 5, size: 0, typ: entity.AddrTypeText},
			},
			outcome: "error",
		},
		{
			name: "overflow retains legacy handling",
			claims: []validationClaim{
				{pkg: "a", addr: math.MaxUint64 - 4, size: 8, typ: entity.AddrTypeData},
			},
			outcome: "success",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original, optimized := validationInfo(tc.claims), validationInfo(tc.claims)
			originalPanic, originalErr := validationOutcome(original.CollectCoverage)
			optimizedPanic, optimizedErr := validationOutcome(optimized.ValidateNativeCoverage)
			require.Nil(t, originalPanic)
			require.Nil(t, optimizedPanic)
			switch tc.outcome {
			case "error":
				var originalConflict, optimizedConflict *entity.ErrAddrCoverageConflict
				require.ErrorAs(t, originalErr, &originalConflict)
				require.ErrorAs(t, optimizedErr, &optimizedConflict)
				require.Equal(t, originalErr.Error(), optimizedErr.Error())
			case "success":
				require.NoError(t, originalErr)
				require.NoError(t, optimizedErr)
				require.NotNil(t, optimized.Coverage, "ambiguous ranges must use the legacy collector")
				require.Len(t, optimized.Coverage, len(original.Coverage))
				for i, part := range original.Coverage {
					require.Equal(t, part.Pos, optimized.Coverage[i].Pos)
					require.Equal(t, part.HasDisasm(), optimized.Coverage[i].HasDisasm())
				}
			default:
				t.Fatalf("unknown validation outcome %q", tc.outcome)
			}
		})
	}
}

func TestValidateNativeCoveragePropagatesCachedChildErrors(t *testing.T) {
	k := validationInfo([]validationClaim{
		{pkg: "p", addr: 100, size: 10, typ: entity.AddrTypeData},
		{pkg: "p/child", addr: 0, size: 10, typ: entity.AddrTypeData},
		{pkg: "p/child", addr: 5, size: 10, typ: entity.AddrTypeText},
	})
	var previous error
	for range 2 {
		panicValue, err := validationOutcome(k.ValidateNativeCoverage)
		require.Nil(t, panicValue)
		var conflict *entity.ErrAddrCoverageConflict
		require.ErrorAs(t, err, &conflict)
		require.ErrorContains(t, err, `package "p/child" coverage`)
		if previous != nil {
			require.Same(t, previous, err, "the parent must propagate its child's cached error")
		}
		previous = err
		require.Nil(t, k.Coverage)
		require.Zero(t, k.Deps.Trie.Get("p").Size)
		require.Zero(t, k.Deps.Trie.Get("p/child").Size)
	}
}

func TestValidateNativeCoverageEmptyBoundaryDoesNotConflict(t *testing.T) {
	for _, claims := range [][]validationClaim{
		{
			{pkg: "a", addr: 0, size: 10, typ: entity.AddrTypeData},
			{pkg: "b", addr: 10, size: 0, typ: entity.AddrTypeText},
		},
		{
			{pkg: "a", addr: 10, size: 0, typ: entity.AddrTypeData},
			{pkg: "b", addr: 10, size: 10, typ: entity.AddrTypeText},
		},
		{
			{pkg: "a", addr: 10, size: 0, typ: entity.AddrTypeData},
			{pkg: "b", addr: 10, size: 0, typ: entity.AddrTypeText},
		},
	} {
		original, optimized := validationInfo(claims), validationInfo(claims)
		require.NoError(t, original.CollectCoverage())
		require.NoError(t, optimized.ValidateNativeCoverage())
		require.Nil(t, optimized.Coverage, "empty ranges at boundaries can use the fast path")
	}
}

func TestValidateNativeCoverageInterleavedTypeBounds(t *testing.T) {
	// The overall data envelope surrounds text, but its actual ranges leave
	// a gap. The detailed fallback must still avoid freezing package caches.
	k := validationInfo([]validationClaim{
		{pkg: "data", addr: 0, size: 10, typ: entity.AddrTypeData},
		{pkg: "data", addr: 40, size: 10, typ: entity.AddrTypeData},
		{pkg: "text", addr: 20, size: 10, typ: entity.AddrTypeText},
	})
	require.NoError(t, k.ValidateNativeCoverage())
	require.Nil(t, k.Coverage)
}

func BenchmarkValidateNativeCoverageSeparatedTypes(b *testing.B) {
	claims := make([]validationClaim, 50000)
	for i := range claims {
		claims[i] = validationClaim{pkg: "data", addr: uint64(i) * 8, size: 16, typ: entity.AddrTypeData}
	}
	claims = append(claims, validationClaim{pkg: "text", addr: 500000, size: 100000, typ: entity.AddrTypeText})
	k := validationInfo(claims)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := k.ValidateNativeCoverage(); err != nil {
			b.Fatal(err)
		}
	}
}

func validationInfo(claims []validationClaim) *KnownInfo {
	k := &KnownInfo{}
	k.Deps = NewDependencies(k)
	for _, claim := range claims {
		p := k.Deps.Trie.Get(claim.pkg)
		if p == nil {
			p = entity.NewPackage()
			p.Name = claim.pkg
			k.Deps.Trie.Put(p.Name, p)
		}
		addValidationClaim(p, claim)
	}
	k.Deps.FinishLoad(false)
	return k
}

func addValidationClaim(p *entity.Package, claim validationClaim) {
	source := claim.source
	if source == "" {
		source = entity.AddrSourceSymbol
	}
	p.AddSymbolCoverage(&entity.Addr{
		AddrPos:    &entity.AddrPos{Addr: claim.addr, Size: claim.size, Type: claim.typ},
		Pkg:        p,
		SourceType: source,
	})
}

func validationOutcome(validate func() error) (panicValue any, err error) {
	defer func() { panicValue = recover() }()
	err = validate()
	return nil, err
}
