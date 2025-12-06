package protocol

import (
	"fmt"
	"strconv"
)

// ResponseType indicates the type of response.
type ResponseType string

const (
	ResponseOK    ResponseType = "OK"
	ResponseErr   ResponseType = "ERR"
	ResponseData  ResponseType = "DATA"
	ResponseJSON  ResponseType = "JSON"
	ResponseChunk ResponseType = "CHUNK"
	ResponseEnd   ResponseType = "END"
	ResponsePong  ResponseType = "PONG"
)

// Response represents a response from the daemon.
type Response struct {
	Type    ResponseType
	Message string // For OK/ERR responses
	Code    string // Error code for ERR responses
	Data    []byte // Binary/JSON data for DATA/JSON/CHUNK responses
}

// ErrorCode represents daemon error codes.
type ErrorCode string

const (
	ErrNotFound      ErrorCode = "not_found"
	ErrAlreadyExists ErrorCode = "already_exists"
	ErrInvalidState  ErrorCode = "invalid_state"
	ErrShuttingDown  ErrorCode = "shutting_down"
	ErrPortInUse     ErrorCode = "port_in_use"
	ErrInvalidArgs   ErrorCode = "invalid_args"
	ErrTimeout       ErrorCode = "timeout"
	ErrInternal      ErrorCode = "internal"
)

// FormatOK formats a simple OK response.
func FormatOK(message string) []byte {
	if message == "" {
		return []byte("OK\r\n")
	}
	return []byte(fmt.Sprintf("OK %s\r\n", message))
}

// FormatErr formats an error response.
func FormatErr(code ErrorCode, message string) []byte {
	return []byte(fmt.Sprintf("ERR %s %s\r\n", code, message))
}

// FormatPong formats a PONG response.
func FormatPong() []byte {
	return []byte("PONG\r\n")
}

// FormatJSON formats a JSON response with length prefix.
func FormatJSON(data []byte) []byte {
	header := fmt.Sprintf("JSON %d\r\n", len(data))
	result := make([]byte, len(header)+len(data)+2)
	copy(result, header)
	copy(result[len(header):], data)
	copy(result[len(header)+len(data):], "\r\n")
	return result
}

// FormatData formats a binary data response with length prefix.
func FormatData(data []byte) []byte {
	header := fmt.Sprintf("DATA %d\r\n", len(data))
	result := make([]byte, len(header)+len(data)+2)
	copy(result, header)
	copy(result[len(header):], data)
	copy(result[len(header)+len(data):], "\r\n")
	return result
}

// FormatChunk formats a streaming chunk with length prefix.
func FormatChunk(data []byte) []byte {
	header := fmt.Sprintf("CHUNK %d\r\n", len(data))
	result := make([]byte, len(header)+len(data)+2)
	copy(result, header)
	copy(result[len(header):], data)
	copy(result[len(header)+len(data):], "\r\n")
	return result
}

// FormatEnd formats an END response for chunked streams.
func FormatEnd() []byte {
	return []byte("END\r\n")
}

// ParseLengthPrefixed parses a length-prefixed response line.
// Returns the length value and any remaining data on the line.
func ParseLengthPrefixed(line string, prefix string) (int, error) {
	if len(line) <= len(prefix)+1 {
		return 0, fmt.Errorf("invalid %s response: too short", prefix)
	}

	lengthStr := line[len(prefix)+1:]
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s length: %w", prefix, err)
	}

	return length, nil
}
