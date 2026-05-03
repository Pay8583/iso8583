package iso8583

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidMTI       = errors.New("invalid MTI")
	ErrInvalidBitmap    = errors.New("invalid bitmap")
	ErrFieldTooLong     = errors.New("field value exceeds maximum length")
	ErrFieldTooShort    = errors.New("field value too short")
	ErrInvalidEncoding  = errors.New("invalid encoding for field")
	ErrFieldNotFound    = errors.New("field not present in message")
	ErrMissingRequired  = errors.New("required field missing")
	ErrSignatureInvalid = errors.New("signature verification failed")
	ErrTruncated        = errors.New("message truncated")
	ErrUnknownField     = errors.New("unknown field number in bitmap")
)

// Error wraps an operation, optional field number, and underlying error.
type Error struct {
	Op    string // "marshal", "unmarshal", "encode", "decode", "sign", "verify"
	Field int    // field number, or 0 if N/A
	Err   error
}

func (e *Error) Error() string {
	if e.Field != 0 {
		return fmt.Sprintf("iso8583: %s field %d: %v", e.Op, e.Field, e.Err)
	}
	return fmt.Sprintf("iso8583: %s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func newError(op string, field int, err error) error {
	return &Error{Op: op, Field: field, Err: err}
}
