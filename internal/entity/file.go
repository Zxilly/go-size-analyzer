package entity

type File struct {
	FilePath  string      `json:"file_path"`
	Functions []*Function `json:"functions"`

	PkgName string `json:"-"`
}

func (f *File) FullSize() uint64 {
	size := uint64(0)
	for _, fn := range f.Functions {
		size += fn.Size()
	}
	return size
}

func (f *File) PclnSize() uint64 {
	size := uint64(0)
	for _, fn := range f.Functions {
		size += fn.PclnSize.Size()
	}
	return size
}
