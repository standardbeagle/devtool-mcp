package protocol

import (
	"encoding/base64"
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
	ErrNotFound       ErrorCode = "not_found"
	ErrAlreadyExists  ErrorCode = "already_exists"
	ErrInvalidState   ErrorCode = "invalid_state"
	ErrShuttingDown   ErrorCode = "shutting_down"
	ErrPortInUse      ErrorCode = "port_in_use"
	ErrInvalidArgs    ErrorCode = "invalid_args"
	ErrInvalidAction  ErrorCode = "invalid_action"  // Unknown action/sub-verb
	ErrInvalidCommand ErrorCode = "invalid_command" // Unknown command/verb
	ErrMissingParam   ErrorCode = "missing_param"   // Required parameter missing
	ErrTimeout        ErrorCode = "timeout"
	ErrInternal       ErrorCode = "internal"
)

// StructuredError contains programmatic error details for MCP translation.
type StructuredError struct {
	Code         ErrorCode `json:"code"`
	Message      string    `json:"message"`
	Command      string    `json:"command,omitempty"`       // The command that was called
	Action       string    `json:"action,omitempty"`        // The action that was attempted
	ValidActions []string  `json:"valid_actions,omitempty"` // List of valid actions
	Param        string    `json:"param,omitempty"`         // The parameter that was invalid/missing
	ValidParams  []string  `json:"valid_params,omitempty"`  // List of valid parameter values
}

// FormatOK formats a simple OK response.
// Format: OK [message];;
func FormatOK(message string) []byte {
	if message == "" {
		return []byte("OK" + CommandTerminator)
	}
	return []byte(fmt.Sprintf("OK %s%s", message, CommandTerminator))
}

// FormatErr formats an error response.
// Format: ERR code message;;
func FormatErr(code ErrorCode, message string) []byte {
	return []byte(fmt.Sprintf("ERR %s %s%s", code, message, CommandTerminator))
}

// FormatPong formats a PONG response.
// Format: PONG;;
func FormatPong() []byte {
	return []byte("PONG" + CommandTerminator)
}

// FormatJSON formats a JSON response with base64 encoded data.
// Format: JSON -- LENGTH\nBASE64DATA;;
func FormatJSON(data []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(data)
	return []byte(fmt.Sprintf("JSON %s %d\n%s%s", DataMarker, len(encoded), encoded, CommandTerminator))
}

// FormatData formats a binary data response with base64 encoding.
// Format: DATA -- LENGTH\nBASE64DATA;;
func FormatData(data []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(data)
	return []byte(fmt.Sprintf("DATA %s %d\n%s%s", DataMarker, len(encoded), encoded, CommandTerminator))
}

// FormatChunk formats a streaming chunk with base64 encoding.
// Format: CHUNK -- LENGTH\nBASE64DATA;;
func FormatChunk(data []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(data)
	return []byte(fmt.Sprintf("CHUNK %s %d\n%s%s", DataMarker, len(encoded), encoded, CommandTerminator))
}

// FormatEnd formats an END response for chunked streams.
// Format: END;;
func FormatEnd() []byte {
	return []byte("END" + CommandTerminator)
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
