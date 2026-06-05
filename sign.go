package iso8583

import "github.com/Pay8583/iso8583/spec"

// Signer produces and verifies cryptographic signatures on packed ISO 8583 messages.
type Signer interface {
	Sign(data []byte) ([]byte, error)
	Verify(data []byte, signature []byte) error
	Algorithm() string
}

// SignOption is a functional option for SignMessage / VerifyMessage /
// ComputeMAC / CheckMAC.
type SignOption func(*spec.SignConfig)

// WithMTI includes the MTI bytes in the signed payload.
func WithMTI() SignOption { return func(o *spec.SignConfig) { o.IncludeMTI = true } }

// WithBitmap includes the bitmap bytes in the signed payload.
func WithBitmap() SignOption { return func(o *spec.SignConfig) { o.IncludeBitmap = true } }

// WithoutBitmap excludes the bitmap bytes from the signed payload.
func WithoutBitmap() SignOption { return func(o *spec.SignConfig) { o.IncludeBitmap = false } }

// ExcludeField excludes a specific field number from the signed payload.
func ExcludeField(n int) SignOption {
	return func(o *spec.SignConfig) { o.ExcludeFields = append(o.ExcludeFields, n) }
}

// WithMACLength sets the MAC byte length for ComputeMAC / CheckMAC.
func WithMACLength(n int) SignOption { return func(o *spec.SignConfig) { o.MACLength = n } }

// WithMACField sets which field holds the MAC (64 or 128).
func WithMACField(n int) SignOption { return func(o *spec.SignConfig) { o.MACField = n } }

// applySignOpts builds a SignConfig from a protocol's default (may be nil)
// and a slice of SignOption overrides. The default config includes the bitmap
// but not the MTI.
func applySignOpts(proto *spec.SignConfig, opts []SignOption) *spec.SignConfig {
	o := &spec.SignConfig{IncludeBitmap: true}
	if proto != nil {
		*o = *proto
	}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// SignMessage computes a signature over the given message bytes.
// By default, the payload starts at the bitmap (excluding the MTI).
// Use WithMTI(), WithBitmap(), ExcludeField(), etc. to adjust.
func SignMessage(data []byte, signer Signer, opts ...SignOption) ([]byte, error) {
	o := applySignOpts(nil, opts)
	payload := extractPayload(data, o)
	return signer.Sign(payload)
}

// VerifyMessage verifies a signature over the given message bytes.
func VerifyMessage(data []byte, signer Signer, signature []byte, opts ...SignOption) error {
	o := applySignOpts(nil, opts)
	payload := extractPayload(data, o)
	return signer.Verify(payload, signature)
}

// extractPayload returns the portion of the message to be signed.
func extractPayload(data []byte, o *spec.SignConfig) []byte {
	if o == nil {
		if len(data) > 4 {
			return data[4:]
		}
		return data
	}

	start := 0
	if o.IncludeMTI {
		start = 0
	} else {
		start = 4
	}
	if start >= len(data) {
		return nil
	}
	payload := data[start:]

	if !o.IncludeBitmap {
		skip := 8
		if len(payload) >= 16 && (payload[0]&0x80) != 0 {
			skip = 16
		}
		if skip < len(payload) {
			payload = payload[skip:]
		} else {
			return nil
		}
	}
	return payload
}
