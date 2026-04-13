package printer

const (
	FormatText = "text"
	FormatJSON = "json"
	FormatHTML = "html"
	FormatSVG  = "svg"
)

// SupportedFormats lists every format accepted by the printer package, in the
// canonical order used by help text and test matrices.
var SupportedFormats = []string{FormatText, FormatJSON, FormatHTML, FormatSVG}

// IsSupportedFormat reports whether name is one of SupportedFormats.
func IsSupportedFormat(name string) bool {
	for _, f := range SupportedFormats {
		if f == name {
			return true
		}
	}
	return false
}
