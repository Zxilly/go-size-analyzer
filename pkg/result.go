package pkg

const (
	ResultUnknownPackage = "unknown"
	ResultStdPackage     = "std"
	ResultVendorPackage  = "vendor"
	ResultSelfPackage    = "main"
	ResultGenerated      = "generated"
)

type Result struct {
	Name     string          `json:"name"`
	Size     uint64          `json:"size"`
	Packages []ResultPackage `json:"packages"`
	// only include size not counted in packages
	SectionSize []ResultSection `json:"section_size"`
	Padding     []ResultSection `json:"padding"`
}

type ResultPackage struct {
	Name     string
	Size     uint64
	Type     string
	Sections []ResultSection
}

type ResultSection struct {
	Name string
	Size uint64
}

type ResultFile struct {
}
