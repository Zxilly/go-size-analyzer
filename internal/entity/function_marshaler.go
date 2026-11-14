//go:build js && wasm

package entity

func (f *Function) MarshalJavaScript() any {
	value := map[string]any{
		"name": f.Name, "addr": f.Addr, "code_size": f.CodeSize,
		"type": f.Type, "receiver": f.Receiver, "pcln_bytes": f.PclnBytes(),
	}
	if f.FullName != "" {
		value["full_name"] = f.FullName
	}
	if f.ReferencedBytes != nil {
		value["referenced_bytes"] = *f.ReferencedBytes
	}
	if f.ReferenceEvidenceCount != 0 {
		value["reference_evidence_count"] = f.ReferenceEvidenceCount
	}
	if len(f.ReferenceEvidence) != 0 {
		evidence := make([]any, 0, len(f.ReferenceEvidence))
		for _, e := range f.ReferenceEvidence {
			evidence = append(evidence, map[string]any{
				"offset": e.Offset, "size": e.Size, "section": e.Section,
				"source": e.Source, "role": e.Role, "boundary": e.Boundary,
			})
		}
		value["reference_evidence"] = evidence
	}
	return value
}
