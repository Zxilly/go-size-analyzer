package entity

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
}
