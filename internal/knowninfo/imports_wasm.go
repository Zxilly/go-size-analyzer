//go:build js && wasm

package knowninfo

func (m *Dependencies) UpdateImportBy() {
	// No-op for wasm
}
