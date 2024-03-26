package printer

type JsonOption struct {
	Indent *int
	minify bool
}

type TextOption struct {
	HideSections bool
	HideMain     bool
	HideStd      bool
}
