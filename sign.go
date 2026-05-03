package iso8583

import "errors"

// Signer produces and verifies cryptographic signatures on packed ISO 8583 messages.
type Signer interface {
	Sign(data []byte) ([]byte, error)
	Verify(data []byte, signature []byte) error
	Algorithm() string
}

// SignOptions controls which portions of a message are included when signing.
type SignOptions struct {
	IncludeMTI    bool
	IncludeBitmap bool
	ExcludeFields []int // fields to omit (e.g., the field that holds the signature itself)
}

// SignMessage computes a signature over the given message bytes according to opts.
// By default, the entire message starting from the bitmap is signed.
func SignMessage(data []byte, signer Signer, opts *SignOptions) ([]byte, error) {
	payload := extractPayload(data, opts)
	return signer.Sign(payload)
}

// VerifyMessage verifies a signature over the given message bytes.
func VerifyMessage(data []byte, signer Signer, signature []byte, opts *SignOptions) error {
	payload := extractPayload(data, opts)
	return signer.Verify(payload, signature)
}

// extractPayload returns the portion of the message that should be signed.
func extractPayload(data []byte, opts *SignOptions) []byte {
	if opts == nil {
		// Default: sign everything after MTI (starting from bitmap).
		if len(data) > 4 {
			return data[4:]
		}
		return data
	}

	start := 0
	if opts.IncludeMTI {
		start = 0
	} else {
		start = 4
	}
	if start >= len(data) {
		return nil
	}
	payload := data[start:]

	if !opts.IncludeBitmap {
		// Skip bitmap: 8 bytes if no secondary, 16 if secondary.
		if len(payload) >= 8 {
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
	}
	return payload
}

// ErrSigning is returned when a signing or verification operation fails.
var ErrSigning = errors.New("signing operation failed")
