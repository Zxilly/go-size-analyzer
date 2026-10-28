package entity_test

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestFileLedgerRandomizedByteOracle(t *testing.T) {
	for seed := range uint64(100) {
		t.Run(fmt.Sprintf("seed-%03d", seed), func(t *testing.T) {
			random := rand.New(rand.NewPCG(seed, 0x5a17)) //nolint:gosec // Deterministic test cases make failures reproducible.
			size := uint64(random.IntN(80))
			owners := []string{"", "a", "a/child", "b", "nogroup", "nilgroup", "emptygroup"}
			sources := []string{"", "symbol", "dwarf", "metadata", "hint"}
			roles := []string{"", "definition", "reference"}
			boundaries := []string{"", "declared", "encoded", "heuristic"}
			groups := ledgerOracleGroups()
			if seed%3 == 0 {
				groups = nil
			}
			claims := make([]entity.FileClaim, random.IntN(80))
			for i := range claims {
				offset := random.Uint64N(size + 1)
				claims[i] = entity.FileClaim{
					FileRange: entity.FileRange{Offset: offset, Size: random.Uint64N(min(12, size-offset) + 1)},
					Class:     random.IntN(3),
					Owner:     owners[random.IntN(len(owners))],
					Source:    sources[random.IntN(len(sources))],
					Role:      roles[random.IntN(len(roles))],
					Boundary:  boundaries[random.IntN(len(boundaries))],
					Metadata:  random.IntN(2) == 0,
				}
				if claims[i].Class != 2 && i%3 == 0 {
					claims[i].Role = "ignored-non-attributed-role"
				}
			}
			checkLedgerByteOracle(t, size, claims, groups)
			shuffled := slices.Clone(claims)
			random.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
			if len(claims) > 0 {
				shuffled = append(shuffled, claims[0], claims[len(claims)/2], claims[len(claims)-1])
			}
			want := ledgerByteOracle(size, claims, groups, true)
			require.Equal(t, want, ledgerByteOracle(size, shuffled, groups, true), "claim order and multiplicity do not change byte ownership")
			checkLedgerByteOracle(t, size, shuffled, groups)
		})
	}
}

func TestFileLedgerByteOracleBlankProvenanceAndGroups(t *testing.T) {
	claims := []entity.FileClaim{
		{FileRange: entity.FileRange{Offset: 0, Size: 2}, Class: 0, Source: "padding"},
		{FileRange: entity.FileRange{Offset: 2, Size: 2}, Class: 1},
		{FileRange: entity.FileRange{Offset: 4, Size: 2}, Class: 2},
		{FileRange: entity.FileRange{Offset: 6, Size: 2}, Class: 2, Source: "symbol"},
		{FileRange: entity.FileRange{Offset: 6, Size: 2}, Class: 2, Owner: "a", Source: "ref", Role: "reference"},
		{FileRange: entity.FileRange{Offset: 8, Size: 2}, Class: 2, Owner: "a", Source: "symbol"},
		{FileRange: entity.FileRange{Offset: 8, Size: 2}, Class: 2, Owner: "b", Source: "ref", Role: "reference", Metadata: true},
		{FileRange: entity.FileRange{Offset: 10, Size: 2}, Class: 2, Owner: "a", Source: "meta", Metadata: true},
		{FileRange: entity.FileRange{Offset: 10, Size: 2}, Class: 2, Owner: "a/child", Source: "meta", Metadata: true},
	}
	groups := ledgerOracleGroups()
	want := ledgerByteOracle(12, claims, groups, true)
	require.Equal(t, uint64(8), want.Attributed)
	require.Equal(t, uint64(2), want.Recognized)
	require.Equal(t, uint64(2), want.Unclassified)
	require.Equal(t, uint64(2), want.Shared)
	require.Equal(t, map[string]uint64{"symbol": 4, "ref": 4, "meta": 2}, want.BySource)
	require.Equal(t, map[string]uint64{"": 6, "root": 6, "a": 6, "a/child": 2}, want.PackageSizes)
	require.Equal(t, map[string]uint64{"a": 2, "a/child": 2}, want.MetadataSizes)
	require.Equal(t, []entity.FileRange{{Offset: 4, Size: 8}}, want.AttributedRanges)
	checkLedgerByteOracle(t, 12, claims, groups)

	claims = []entity.FileClaim{
		{FileRange: entity.FileRange{Offset: 0, Size: 2}, Class: 2, Owner: "nilgroup", Metadata: true},
		{FileRange: entity.FileRange{Offset: 2, Size: 2}, Class: 2, Owner: "emptygroup", Metadata: true},
		{FileRange: entity.FileRange{Offset: 4, Size: 2}, Class: 2, Owner: "nogroup", Metadata: true},
	}
	want = ledgerByteOracle(6, claims, groups, true)
	require.Equal(t, map[string]uint64{"nogroup": 2}, want.PackageSizes)
	require.Equal(t, map[string]uint64{"nilgroup": 2, "emptygroup": 2, "nogroup": 2}, want.MetadataSizes)
	checkLedgerByteOracle(t, 6, claims, groups)
}

func TestFileLedgerByteOracleCoalescesHiddenBoundaries(t *testing.T) {
	claims := []entity.FileClaim{
		{FileRange: entity.FileRange{Size: 8}, Class: 2, Owner: "a", Source: "symbol"},
		{FileRange: entity.FileRange{Offset: 1, Size: 6}, Class: 2, Owner: "a", Source: "symbol"},
		{FileRange: entity.FileRange{Offset: 2, Size: 2}, Class: 2, Owner: "a", Source: "symbol", Metadata: true},
		{FileRange: entity.FileRange{Offset: 3, Size: 2}, Class: 1, Source: "hidden"},
		{FileRange: entity.FileRange{Offset: 4, Size: 2}, Class: 0, Source: "also-hidden"},
	}
	want := ledgerByteOracle(8, claims, nil, true)
	require.Len(t, want.Regions, 1, "metadata flags and lower-priority claims do not split identical visible evidence")
	require.Equal(t, map[string]uint64{"a": 2}, want.MetadataSizes)
	checkLedgerByteOracle(t, 8, claims, nil)
	checkLedgerByteOracle(t, 0, nil, nil)
	checkLedgerByteOracle(t, 8, nil, nil)
}

func TestFileLedgerAtMaximumOffset(t *testing.T) {
	ledger := entity.NewFileLedger(math.MaxUint64)
	ledger.SetPackageGroups(map[string][]string{"same": {"", "same", "same"}})
	require.NoError(t, ledger.Add(entity.FileClaim{
		FileRange: entity.FileRange{Offset: math.MaxUint64 - 2, Size: 2},
		Class:     2, Owner: "same", Source: "same", Metadata: true,
	}))
	want := &entity.FileCoverage{
		Version: 2, Attributed: 2, Unclassified: math.MaxUint64 - 2,
		BySource:         map[string]uint64{"same": 2},
		PackageSizes:     map[string]uint64{"": 2, "same": 2},
		MetadataSizes:    map[string]uint64{"same": 2},
		AttributedRanges: []entity.FileRange{{Offset: math.MaxUint64 - 2, Size: 2}},
		Regions: []entity.FileRegion{
			{FileRange: entity.FileRange{Size: math.MaxUint64 - 2}, Class: "unclassified", Sources: []string{"unmapped-file-range"}},
			{
				FileRange: entity.FileRange{Offset: math.MaxUint64 - 2, Size: 2}, Class: "attributed",
				Owners: []string{"same"}, DefinitionOwners: []string{"same"}, Referrers: []string{}, Sources: []string{"same"},
				Evidence: []entity.FileEvidence{{Owner: "same", Source: "same", Role: "definition"}},
			},
		},
	}
	require.Equal(t, want, ledger.Finish(true))
	want.Regions = nil
	require.Equal(t, want, ledger.Finish(false))
}

func checkLedgerByteOracle(t *testing.T, size uint64, claims []entity.FileClaim, groups map[string][]string) {
	t.Helper()
	ledger := entity.NewFileLedger(size)
	ledger.SetPackageGroups(groups)
	for _, claim := range claims {
		require.NoError(t, ledger.Add(claim))
	}
	wantDetails := ledgerByteOracle(size, claims, groups, true)
	wantSummary := ledgerByteOracle(size, claims, groups, false)
	first := ledger.Finish(true)
	require.Equal(t, wantDetails, first)
	require.Equal(t, wantSummary, ledger.Finish(false))
	require.Equal(t, wantDetails, ledger.Finish(true))
	require.Equal(t, wantSummary, ledger.Finish(false))

	// Finishing must not freeze the claims or group projection for later calls.
	later := entity.FileClaim{FileRange: entity.FileRange{Offset: size / 2, Size: size - size/2}, Class: 2, Owner: "later", Source: "later", Metadata: true}
	require.NoError(t, ledger.Add(later))
	more := append(slices.Clone(claims), later)
	require.Equal(t, ledgerByteOracle(size, more, groups, true), ledger.Finish(true))
	require.Equal(t, ledgerByteOracle(size, more, groups, false), ledger.Finish(false))
	newGroups := map[string][]string{"later": {"", "new-parent", "new-parent"}, "a": nil}
	ledger.SetPackageGroups(newGroups)
	require.Equal(t, ledgerByteOracle(size, more, newGroups, true), ledger.Finish(true))
	require.Equal(t, ledgerByteOracle(size, more, newGroups, false), ledger.Finish(false))
	require.Equal(t, wantDetails, first, "later Finish calls must not mutate previously returned coverage")
}

func ledgerOracleGroups() map[string][]string {
	return map[string][]string{
		"":           {"ignored-empty-owner"},
		"a":          {"", "root", "a", "root"},
		"a/child":    {"", "root", "a", "a/child"},
		"b":          {"root", "b"},
		"nilgroup":   nil,
		"emptygroup": {},
	}
}

type ledgerOracleCell struct {
	region   entity.FileRegion
	packages []string
	metadata []string
}

// ledgerByteOracle independently evaluates every physical byte. It uses no
// ledger events, interned IDs, counters, or production coalescing helpers.
func ledgerByteOracle(size uint64, claims []entity.FileClaim, groups map[string][]string, details bool) *entity.FileCoverage {
	result := &entity.FileCoverage{
		Version: 2, BySource: map[string]uint64{}, PackageSizes: map[string]uint64{}, MetadataSizes: map[string]uint64{},
	}
	cells := make([]ledgerOracleCell, size)
	for offset := range size {
		cell := ledgerOracleAt(offset, claims, groups)
		cells[offset] = cell
		switch cell.region.Class {
		case "attributed":
			result.Attributed++
			if len(cell.region.Owners) > 1 {
				result.Shared++
			}
			for _, source := range cell.region.Sources {
				result.BySource[source]++
			}
			for _, name := range cell.packages {
				result.PackageSizes[name]++
			}
			for _, name := range cell.metadata {
				result.MetadataSizes[name]++
			}
		case "recognized":
			result.Recognized++
		default:
			result.Unclassified++
		}
	}
	for start := 0; start < len(cells); {
		if cells[start].region.Class != "attributed" {
			start++
			continue
		}
		end := start + 1
		for end < len(cells) && cells[end].region.Class == "attributed" {
			end++
		}
		result.AttributedRanges = append(result.AttributedRanges, entity.FileRange{Offset: uint64(start), Size: uint64(end - start)})
		start = end
	}
	if details {
		for start := 0; start < len(cells); {
			end := start + 1
			for end < len(cells) && reflect.DeepEqual(cells[start].region, cells[end].region) {
				end++
			}
			region := cells[start].region
			region.FileRange = entity.FileRange{Offset: uint64(start), Size: uint64(end - start)}
			result.Regions = append(result.Regions, region)
			start = end
		}
	}
	return result
}

func ledgerOracleAt(offset uint64, claims []entity.FileClaim, groups map[string][]string) ledgerOracleCell {
	class := 0
	var active []entity.FileClaim
	for _, claim := range claims {
		if offset >= claim.Offset && offset-claim.Offset < claim.Size {
			active = append(active, claim)
			class = max(class, claim.Class)
		}
	}
	sources, definitions, references, metadata := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	evidence := map[entity.FileEvidence]bool{}
	for _, claim := range active {
		if claim.Class != class {
			continue
		}
		if claim.Source != "" {
			sources[claim.Source] = true
		}
		if class != 2 {
			continue
		}
		role := claim.Role
		if role == "" {
			role = "definition"
		}
		evidence[entity.FileEvidence{Owner: claim.Owner, Source: claim.Source, Role: role, Boundary: claim.Boundary}] = true
		if claim.Owner == "" {
			continue
		}
		if role == "reference" {
			references[claim.Owner] = true
		} else {
			definitions[claim.Owner] = true
		}
		if claim.Metadata {
			metadata[claim.Owner] = true
		}
	}
	cell := ledgerOracleCell{region: entity.FileRegion{Class: [3]string{"unclassified", "recognized", "attributed"}[class], Sources: ledgerOracleNames(sources)}}
	if class == 0 && len(sources) == 0 {
		cell.region.Sources = []string{"unmapped-file-range"}
	}
	if class != 2 {
		return cell
	}
	owners := definitions
	if len(owners) == 0 {
		owners = references
	}
	cell.region.Owners = ledgerOracleNames(owners)
	cell.region.DefinitionOwners = ledgerOracleNames(definitions)
	cell.region.Referrers = ledgerOracleNames(references)
	for item := range evidence {
		cell.region.Evidence = append(cell.region.Evidence, item)
	}
	slices.SortFunc(cell.region.Evidence, func(a, b entity.FileEvidence) int {
		for _, pair := range [][2]string{{a.Owner, b.Owner}, {a.Source, b.Source}, {a.Role, b.Role}, {a.Boundary, b.Boundary}} {
			if order := cmp.Compare(pair[0], pair[1]); order != 0 {
				return order
			}
		}
		return 0
	})
	packageNames, metadataNames := map[string]bool{}, map[string]bool{}
	for owner := range owners {
		if metadata[owner] {
			metadataNames[owner] = true
		}
		ancestors, exists := groups[owner]
		if !exists {
			ancestors = []string{owner}
		}
		for _, ancestor := range ancestors {
			packageNames[ancestor] = true
		}
	}
	cell.packages, cell.metadata = ledgerOracleNames(packageNames), ledgerOracleNames(metadataNames)
	return cell
}

func ledgerOracleNames(values map[string]bool) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func BenchmarkFileLedgerRealistic(b *testing.B) {
	const fileSize, claimCount, ownerCount = uint64(8 << 20), 50_000, 200
	random := rand.New(rand.NewPCG(17, 43)) //nolint:gosec // Both benchmark variants need identical generated claims.
	sources := []string{"symbol", "dwarf", "pclntab", "typelink", "itab", "static_read", "symbol_table", "symbol_name"}
	owners := make([]string, ownerCount)
	groups := make(map[string][]string, ownerCount)
	for i := range owners {
		owners[i] = fmt.Sprintf("pkg/%03d", i)
		groups[owners[i]] = []string{"root", fmt.Sprintf("group/%02d", i%10), owners[i]}
	}
	claims := make([]entity.FileClaim, claimCount)
	for i := range claims {
		claims[i] = entity.FileClaim{
			FileRange: entity.FileRange{Offset: random.Uint64N(fileSize-2048) &^ 7, Size: 16 + random.Uint64N(512)},
			Class:     2,
			Owner:     owners[random.IntN(ownerCount)],
			Source:    sources[i%len(sources)],
			Boundary:  "encoded",
			Metadata:  i%7 == 0,
		}
		if i%4 == 0 {
			claims[i].Role = "reference"
		}
		if i%17 == 0 {
			claims[i].Class = 1
		}
		if i%53 == 0 {
			claims[i].Class = 0
		}
		if i%41 == 0 {
			claims[i].Size += 1024
		}
		if i > 0 && i%97 == 0 {
			claims[i] = claims[i-1]
		}
	}
	for _, details := range []bool{false, true} {
		name := "summary"
		if details {
			name = "details"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				ledger := entity.NewFileLedger(fileSize)
				ledger.SetPackageGroups(groups)
				for _, claim := range claims {
					if err := ledger.Add(claim); err != nil {
						b.Fatal(err)
					}
				}
				if ledger.Finish(details).Attributed == 0 {
					b.Fatal("fixture must attribute bytes")
				}
			}
			b.ReportMetric(claimCount, "claims/op")
		})
	}
}
