package spec

import "strings"

// Validator checks whether a string value conforms to a field's content rules.
// It is applied before encoding (on marshal) and after decoding (on unmarshal).
type Validator interface {
	// Ok reports whether s is valid according to this rule.
	Ok(s string) bool
	// String returns a human-readable name for this validator.
	String() string
}

// ── Built-in validators ───────────────────────────────────────────────────────

// N validates that every character is an ASCII digit [0-9].
var N Validator = numericValidator{}

// AN validates that every character is alphanumeric [A-Za-z0-9].
var AN Validator = alphaNumValidator{}

// ANS validates that every character is printable ASCII (0x20–0x7E).
// Control characters and DEL (0x7F) are rejected.
var ANS Validator = alphaNumSpecValidator{}

// B accepts any byte sequence (binary data). Always valid.
var B Validator = binValidator{}

// Z validates track data characters: printable ASCII (0x20–0x7E).
// Named distinctly from ANS so call-sites document track-data intent.
var Z Validator = trackValidator{}

// NS validates numeric-special: digits [0-9] plus space, dot, and dash.
var NS Validator = numSpecValidator{}

// XN validates signed numeric: an optional leading sign [+-] followed by
// one or more digits [0-9].
var XN Validator = signedNumValidator{}

// ── Implementations ────────────────────────────────────────────────────────────

type numericValidator struct{}

func (numericValidator) Ok(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (numericValidator) String() string { return "N" }

// ────────────────────────────────────────────────────────────────────────────────

type alphaNumValidator struct{}

func (alphaNumValidator) Ok(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := range len(s) {
		c := s[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func (alphaNumValidator) String() string { return "AN" }

// ────────────────────────────────────────────────────────────────────────────────

type alphaNumSpecValidator struct{}

func (alphaNumSpecValidator) Ok(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := range len(s) {
		c := s[i]
		if c < 0x20 || c > 0x7E {
			return false
		}
	}
	return true
}

func (alphaNumSpecValidator) String() string { return "ANS" }

// ────────────────────────────────────────────────────────────────────────────────

type binValidator struct{}

func (binValidator) Ok(s string) bool {
	return true // binary data is never invalid
}

func (binValidator) String() string { return "B" }

// ────────────────────────────────────────────────────────────────────────────────

type trackValidator struct{}

func (trackValidator) Ok(s string) bool {
	if len(s) == 0 {
		return true
	}
	// Track data in ISO 8583 is a mix of digits, hex chars, sentinels,
	// and printable characters. We accept printable ASCII.
	for i := range len(s) {
		c := s[i]
		if c < 0x20 || c > 0x7E {
			return false
		}
	}
	return true
}

func (trackValidator) String() string { return "Z" }

// ────────────────────────────────────────────────────────────────────────────────

type numSpecValidator struct{}

func (numSpecValidator) Ok(s string) bool {
	if len(s) == 0 {
		return true
	}
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c == ' ' || c == '.' || c == '-':
		default:
			return false
		}
	}
	return true
}

func (numSpecValidator) String() string { return "NS" }

// ────────────────────────────────────────────────────────────────────────────────

type signedNumValidator struct{}

func (signedNumValidator) Ok(s string) bool {
	if len(s) == 0 {
		return true
	}
	v := s
	// Optional leading sign.
	if len(v) > 0 && (v[0] == '+' || v[0] == '-') {
		v = v[1:]
	}
	// Must have at least one digit after optional sign.
	if len(v) == 0 {
		return false
	}
	// Remainder must be all digits.
	for i := range len(v) {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
	}
	return true
}

func (signedNumValidator) String() string { return "XN" }

// ── Helpers ─────────────────────────────────────────────────────────────────────

// ValidatorByName returns the built-in Validator for a case-insensitive name,
// or nil if the name is not recognized.
func ValidatorByName(name string) Validator {
	switch strings.ToUpper(name) {
	case "N":
		return N
	case "AN":
		return AN
	case "ANS":
		return ANS
	case "B":
		return B
	case "Z":
		return Z
	case "NS":
		return NS
	case "XN":
		return XN
	default:
		return nil
	}
}

// AllValidators returns all built-in validators.
func AllValidators() []Validator {
	return []Validator{N, AN, ANS, B, Z, NS, XN}
}