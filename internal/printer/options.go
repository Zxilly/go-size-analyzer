package printer

type Option struct {
	HideSections bool
	HideMain     bool
	HideStd      bool

	JsonIndent int

	Output string
}
