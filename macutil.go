package iso8583

import (
	"bytes"
	"fmt"

	"github.com/Pay8583/iso8583/spec"
)

// ComputeMAC closes the Writer, then computes a MAC over the bytes before
// the placeholder. If the Writer's Protocol has a Sign config, its MACField
// and MACLength are used; pass WithMACLength() / WithMACField() to override.
//
// Returns the complete message bytes (with the MAC replacing the placeholder)
// and the MAC bytes separately.
func ComputeMAC(w *Writer, signer Signer, opts ...SignOption) (message, mac []byte, err error) {
	raw := w.Bytes()

	// Determine MAC length and placeholder position from protocol + options.
	cfg := signConfig(w.p, opts)
	placeholderLen := cfg.MACLength
	if placeholderLen <= 0 {
		placeholderLen = 8 // default: 8-byte MAC
	}

	if len(raw) < placeholderLen {
		return nil, nil, fmt.Errorf("compute MAC: message too short (%d bytes) for placeholder of %d bytes", len(raw), placeholderLen)
	}
	payload := raw[:len(raw)-placeholderLen]
	mac, err = SignMessage(payload, signer)
	if err != nil {
		return nil, nil, fmt.Errorf("compute MAC: sign: %w", err)
	}
	// Replace placeholder with MAC bytes (truncate MAC if too long).
	message = make([]byte, len(raw))
	copy(message, payload)
	if len(mac) > placeholderLen {
		copy(message[len(payload):], mac[:placeholderLen])
	} else {
		copy(message[len(payload):], mac)
	}
	return message, mac, nil
}

// CheckMAC verifies the MAC on a received message. The last macLen bytes
// of data are the expected MAC; the preceding bytes are the payload.
// If the Protocol has a Sign config, its MACLength is used; pass
// WithMACLength() to override.
func CheckMAC(data []byte, proto *spec.Protocol, signer Signer, opts ...SignOption) error {
	cfg := signConfig(proto, opts)
	macLen := cfg.MACLength
	if macLen <= 0 {
		macLen = 8
	}
	if len(data) < macLen {
		return fmt.Errorf("check MAC: data too short (%d bytes) for MAC of %d bytes", len(data), macLen)
	}
	payload := data[:len(data)-macLen]
	expected := data[len(data)-macLen:]

	computed, err := SignMessage(payload, signer)
	if err != nil {
		return fmt.Errorf("check MAC: sign: %w", err)
	}
	if len(computed) > macLen {
		computed = computed[:macLen]
	}

	if !bytes.Equal(computed, expected) {
		return fmt.Errorf("MAC mismatch: computed %x, expected %x", computed, expected)
	}
	return nil
}

// signConfig merges protocol defaults with explicit options.
func signConfig(p *spec.Protocol, opts []SignOption) *spec.SignConfig {
	var defaults *spec.SignConfig
	if p != nil {
		defaults = p.Sign
	}
	return applySignOpts(defaults, opts)
}
