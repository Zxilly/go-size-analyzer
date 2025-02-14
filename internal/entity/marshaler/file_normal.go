package marshaler

import (
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func GetFileCompactMarshaler() *json.Marshalers {
	return json.MarshalToFunc[entity.File](func(encoder *jsontext.Encoder, file entity.File) error {
		options := encoder.Options()
		utils.Must(encoder.WriteToken(jsontext.BeginObject))

		utils.Must(json.MarshalEncode(encoder, "file_path", options))
		utils.Must(json.MarshalEncode(encoder, file.FilePath, options))
		utils.Must(json.MarshalEncode(encoder, "size", options))
		utils.Must(json.MarshalEncode(encoder, file.FullSize(), options))
		utils.Must(json.MarshalEncode(encoder, "pcln_size", options))
		utils.Must(json.MarshalEncode(encoder, file.PclnSize(), options))

		utils.Must(encoder.WriteToken(jsontext.EndObject))
		return nil
	})
}
