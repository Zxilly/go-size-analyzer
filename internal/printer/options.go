package printer

type JsonOption struct {
	Indent     *int
	HideDetail bool
}

type TextOption struct {
	HideSections bool
	HideMain     bool
	HideStd      bool
}
