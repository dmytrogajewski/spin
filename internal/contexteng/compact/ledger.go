package compact

const (
	// TokenDivisor is the RTK R15 bytes-to-token divisor (not a tokenizer).
	TokenDivisor = 4
	percentScale = 100
)

// Ledger records R15 savings using ceil(bytes/4), not a tokenizer.
type Ledger struct {
	// BytesIn is the pre-filter stdout+stderr length.
	BytesIn int
	// BytesOut is the post-filter stdout+stderr length.
	BytesOut int
	// TokensIn is ceil(BytesIn / TokenDivisor).
	TokensIn int
	// TokensOut is ceil(BytesOut / TokenDivisor).
	TokensOut int
	// TokensSaved is TokensIn minus TokensOut.
	TokensSaved int
	// ReductionPct is the byte reduction percent (0 on identity).
	ReductionPct float64
}

// Ledger metadata keys on tool-complete events (bytes, not tokenizer counts).
const (
	MetaBytesIn  = "compact_bytes_in"
	MetaBytesOut = "compact_bytes_out"
)

// ByteReductionPct is output-bytes reduction from ledger BytesIn/BytesOut.
func ByteReductionPct(bytesIn, bytesOut int) float64 {
	return reductionPct(bytesIn, bytesOut)
}

func account(inStdout, inStderr, outStdout, outStderr []byte) Ledger {
	bytesIn := len(inStdout) + len(inStderr)
	bytesOut := len(outStdout) + len(outStderr)

	tokensIn := tokenEstimate(bytesIn)
	tokensOut := tokenEstimate(bytesOut)

	return Ledger{
		BytesIn:      bytesIn,
		BytesOut:     bytesOut,
		TokensIn:     tokensIn,
		TokensOut:    tokensOut,
		TokensSaved:  tokensIn - tokensOut,
		ReductionPct: reductionPct(bytesIn, bytesOut),
	}
}

func tokenEstimate(nBytes int) int {
	if nBytes <= 0 {
		return 0
	}

	return (nBytes + TokenDivisor - 1) / TokenDivisor
}

func reductionPct(bytesIn, bytesOut int) float64 {
	if bytesIn == 0 || bytesIn == bytesOut {
		return 0
	}

	return percentScale * (1 - float64(bytesOut)/float64(bytesIn))
}
