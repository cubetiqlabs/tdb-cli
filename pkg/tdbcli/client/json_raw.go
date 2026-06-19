package client

// jsonRaw represents a pre-encoded JSON payload.
// It exists to allow skipping JSON encoding overhead when the payload is already serialized.
type jsonRaw []byte

// MarshalJSON returns the raw bytes, allowing it to satisfy interfaces that might check for it.
func (r jsonRaw) MarshalJSON() ([]byte, error) {
	return r, nil
}
