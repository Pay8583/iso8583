package iso8583

import "fmt"

// MTI constants for common message classes and functions.
const (
	MTIAuthorizationRequest  = "0100"
	MTIAuthorizationResponse = "0110"
	MTIAdviceRequest         = "0120"
	MTIAdviceResponse        = "0130"
	MTIFinancialRequest      = "0200"
	MTIFinancialResponse     = "0210"
	MTIReversalRequest       = "0400"
	MTIReversalResponse      = "0410"
	MTINetworkManagementReq  = "0800"
	MTINetworkManagementResp = "0810"
)

// MTIVersion returns the third digit of the MTI, indicating the ISO 8583 version.
// 0 = 1987, 1 = 1993, 2 = 2003.
func MTIVersion(mti string) (int, error) {
	if len(mti) != 4 {
		return 0, fmt.Errorf("MTI must be 4 bytes, got %d", len(mti))
	}
	v := mti[2]
	switch v {
	case '0':
		return 1987, nil
	case '1':
		return 1993, nil
	case '2':
		return 2003, nil
	default:
		return 0, fmt.Errorf("unknown MTI version digit: %c", v)
	}
}

// MTIClass returns the message class (first two digits of MTI).
func MTIClass(mti string) (string, error) {
	if len(mti) != 4 {
		return "", fmt.Errorf("MTI must be 4 bytes, got %d", len(mti))
	}
	return mti[:2], nil
}

// IsRequest reports whether the MTI denotes a request (fourth digit is 0 or 2).
func IsRequest(mti string) bool {
	if len(mti) != 4 {
		return false
	}
	return mti[3] == '0' || mti[3] == '2'
}

// IsResponse reports whether the MTI denotes a response (fourth digit is 1 or 3).
func IsResponse(mti string) bool {
	if len(mti) != 4 {
		return false
	}
	return mti[3] == '1' || mti[3] == '3'
}
