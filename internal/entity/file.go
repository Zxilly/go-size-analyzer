package entity

import (
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

type File struct {
	FilePath  string      `json:"file_path"`
	Functions []*Function `json:"functions"`

	Pkg *Package `json:"-"`
}

var FileMarshalerCompact = json.MarshalFuncV2[File](func(encoder *jsontext.Encoder, file File, options json.Options) error {
	err := encoder.WriteToken(jsontext.ObjectStart)
	if err != nil {
		return err
	}

	if err = json.MarshalEncode(encoder, "file_path", options); err != nil {
		return err
	}
	if err = json.MarshalEncode(encoder, file.FilePath, options); err != nil {
		return err
	}
	if err = json.MarshalEncode(encoder, "size", options); err != nil {
		return err
	}
	if err = json.MarshalEncode(encoder, file.FullSize(), options); err != nil {
		return err
	}
	if err = json.MarshalEncode(encoder, "pcln_size", options); err != nil {
		return err
	}
	if err = json.MarshalEncode(encoder, file.PclnSize(), options); err != nil {
		return err
	}

	err = encoder.WriteToken(jsontext.ObjectEnd)
	if err != nil {
		return err
	}
	return nil
})

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
