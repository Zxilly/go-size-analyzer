package entity

type SectionContentType int

const (
	SectionContentOther SectionContentType = iota
	SectionContentText
	SectionContentData
)

type Section struct {
	Name string `json:"name"`

	Size     uint64 `json:"size"`
	FileSize uint64 `json:"file_size"`

	KnownSize uint64 `json:"known_size"`

	Offset uint64 `json:"offset"`
	End    uint64 `json:"end"`

	Addr    uint64 `json:"addr"`
	AddrEnd uint64 `json:"addr_end"`

	OnlyInMemory bool `json:"only_in_memory"`
	Debug        bool `json:"debug"`

	// VirtualSection marks a section that exists only as a virtual address
	// space for analysis (e.g. Wasm linear memory). Unlike OnlyInMemory
	// (BSS-like sections excluded from all caches), VirtualSection sections
	// are included in the data/text address cache so that symbol address
	// lookups work, but are excluded from file-size accounting and from
	// FindSection results.
	VirtualSection bool `json:"-"`

	ContentType SectionContentType `json:"-"`
}
