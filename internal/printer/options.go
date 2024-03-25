package printer

type JsonOption struct {
	Indent        *int
	HideFunctions bool
}

type TextOption struct {
	HideSections bool
	HideMain     bool
	HideStd      bool
}
