package printer

type JsonOption struct {
	Indent     *int
	hideDetail bool
}

type TextOption struct {
	HideSections bool
	HideMain     bool
	HideStd      bool
}
