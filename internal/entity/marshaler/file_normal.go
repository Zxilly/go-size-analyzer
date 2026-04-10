package marshaler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func GetFileCompactMarshaler() *json.Marshalers {
	return json.MarshalToFunc(func(encoder *jsontext.Encoder, file entity.File) error {
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
