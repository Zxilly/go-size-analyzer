package internal

type ResultSection struct {
	Name      string `json:"name"`
	KnownSize uint64 `json:"known_size"`
	Size      uint64 `json:"size"`
}

type Result struct {
	Name     string     `json:"name"`
	Size     uint64     `json:"size"`
	Packages PackageMap `json:"packages"`
}
