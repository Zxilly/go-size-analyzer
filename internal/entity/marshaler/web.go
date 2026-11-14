package marshaler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type functionWeb struct {
	Name                   string                    `json:"name"`
	FullName               string                    `json:"full_name,omitempty"`
	Type                   string                    `json:"type"`
	Receiver               string                    `json:"receiver"`
	Addr                   uint64                    `json:"addr"`
	CodeSize               uint64                    `json:"code_size"`
	PclnBytes              uint64                    `json:"pcln_bytes"`
	ReferencedBytes        *uint64                   `json:"referenced_bytes,omitempty"`
	ReferenceEvidence      []entity.FunctionEvidence `json:"reference_evidence,omitempty"`
	ReferenceEvidenceCount int                       `json:"reference_evidence_count,omitzero"`
}

func getFunctionWebMarshaler() *json.Marshalers {
	return json.MarshalToFunc(func(encoder *jsontext.Encoder, f entity.Function) error {
		value := functionWeb{
			Name: f.Name, FullName: f.FullName, Type: f.Type, Receiver: f.Receiver,
			Addr: f.Addr, CodeSize: f.CodeSize, PclnBytes: f.PclnBytes(), ReferencedBytes: f.ReferencedBytes,
			ReferenceEvidence: f.ReferenceEvidence, ReferenceEvidenceCount: f.ReferenceEvidenceCount,
		}
		return json.MarshalEncode(encoder, value, encoder.Options())
	})
}

func getCoverageWebMarshaler() *json.Marshalers {
	return json.MarshalToFunc(func(encoder *jsontext.Encoder, coverage entity.FileCoverage) error {
		// A distinct type avoids recursively applying this marshaler. Copying
		// preserves the full ledger for other outputs in the same invocation.
		type coverageSummary entity.FileCoverage
		summary := coverageSummary(coverage)
		summary.Regions = nil
		return json.MarshalEncode(encoder, summary, encoder.Options())
	})
}
