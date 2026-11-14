package marshaler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func GetFileCompactMarshaler() *json.Marshalers {
	return getFileMarshaler(false)
}

// GetFileDetailMarshaler projects the web payload: file totals, function metrics
// and sampled evidence, plus coverage summaries. The UI does not consume raw
// function ranges, pcln component maps or the complete coverage region ledger.
// Full and compact CLI JSON keep their existing independent representations.
func GetFileDetailMarshaler() *json.Marshalers {
	return json.JoinMarshalers(getFileMarshaler(true), getFunctionWebMarshaler(), getCoverageWebMarshaler())
}

type fileSummary struct {
	FilePath string `json:"file_path"`
	Size     uint64 `json:"size"`
	PclnSize uint64 `json:"pcln_size"`
}

func getFileMarshaler(details bool) *json.Marshalers {
	return json.MarshalToFunc(func(encoder *jsontext.Encoder, file entity.File) error {
		summary := fileSummary{FilePath: file.FilePath, Size: file.FullSize(), PclnSize: file.PclnSize()}
		if details {
			value := struct {
				fileSummary
				Functions []*entity.Function `json:"functions"`
			}{fileSummary: summary, Functions: file.Functions}
			return json.MarshalEncode(encoder, value, encoder.Options())
		}
		return json.MarshalEncode(encoder, summary, encoder.Options())
	})
}
