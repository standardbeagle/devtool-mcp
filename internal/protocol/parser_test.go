package protocol

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseCommand_Simple(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Command
		wantErr bool
	}{
		{
			name:  "PING",
			input: "PING\r\n",
			want:  &Command{Verb: "PING"},
		},
		{
			name:  "INFO",
			input: "INFO\r\n",
			want:  &Command{Verb: "INFO"},
		},
		{
			name:  "SHUTDOWN",
			input: "SHUTDOWN\r\n",
			want:  &Command{Verb: "SHUTDOWN"},
		},
		{
			name:  "DETECT with path",
			input: "DETECT /home/user/project\r\n",
			want:  &Command{Verb: "DETECT", Args: []string{"/home/user/project"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			got, err := parser.ParseCommand()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Verb != tt.want.Verb {
					t.Errorf("Verb = %v, want %v", got.Verb, tt.want.Verb)
				}
				if len(got.Args) != len(tt.want.Args) {
					t.Errorf("Args = %v, want %v", got.Args, tt.want.Args)
				}
			}
		})
	}
}

func TestParseCommand_WithSubVerb(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Command
		wantErr bool
	}{
		{
			name:  "PROC STATUS",
			input: "PROC STATUS my-process\r\n",
			want: &Command{
				Verb:    "PROC",
				SubVerb: "STATUS",
				Args:    []string{"my-process"},
			},
		},
		{
			name:  "PROC LIST",
			input: "PROC LIST\r\n",
			want: &Command{
				Verb:    "PROC",
				SubVerb: "LIST",
			},
		},
		{
			name:  "PROC STOP with force",
			input: "PROC STOP my-process force\r\n",
			want: &Command{
				Verb:    "PROC",
				SubVerb: "STOP",
				Args:    []string{"my-process", "force"},
			},
		},
		{
			name:  "PROXY START",
			input: "PROXY START dev http://localhost:3000 8080\r\n",
			want: &Command{
				Verb:    "PROXY",
				SubVerb: "START",
				Args:    []string{"dev", "http://localhost:3000", "8080"},
			},
		},
		{
			name:  "PROXY LIST",
			input: "PROXY LIST\r\n",
			want: &Command{
				Verb:    "PROXY",
				SubVerb: "LIST",
			},
		},
		{
			name:  "PROXYLOG STATS",
			input: "PROXYLOG STATS dev\r\n",
			want: &Command{
				Verb:    "PROXYLOG",
				SubVerb: "STATS",
				Args:    []string{"dev"},
			},
		},
		{
			name:  "CURRENTPAGE LIST",
			input: "CURRENTPAGE LIST dev\r\n",
			want: &Command{
				Verb:    "CURRENTPAGE",
				SubVerb: "LIST",
				Args:    []string{"dev"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			got, err := parser.ParseCommand()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Verb != tt.want.Verb {
					t.Errorf("Verb = %v, want %v", got.Verb, tt.want.Verb)
				}
				if got.SubVerb != tt.want.SubVerb {
					t.Errorf("SubVerb = %v, want %v", got.SubVerb, tt.want.SubVerb)
				}
				if len(got.Args) != len(tt.want.Args) {
					t.Errorf("Args = %v, want %v", got.Args, tt.want.Args)
				}
			}
		})
	}
}

func TestParseCommand_WithData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantVerb string
		wantData string
		wantErr  bool
	}{
		{
			name:     "RUN-JSON",
			input:    "RUN-JSON 44\r\n{\"id\":\"test\",\"path\":\".\",\"mode\":\"background\"}\r\n",
			wantVerb: "RUN-JSON",
			wantData: `{"id":"test","path":".","mode":"background"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			got, err := parser.ParseCommand()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Verb != tt.wantVerb {
					t.Errorf("Verb = %v, want %v", got.Verb, tt.wantVerb)
				}
				if string(got.Data) != tt.wantData {
					t.Errorf("Data = %v, want %v", string(got.Data), tt.wantData)
				}
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType ResponseType
		wantMsg  string
		wantCode string
		wantData string
		wantErr  bool
	}{
		{
			name:     "OK simple",
			input:    "OK\r\n",
			wantType: ResponseOK,
		},
		{
			name:     "OK with message",
			input:    "OK 12345\r\n",
			wantType: ResponseOK,
			wantMsg:  "12345",
		},
		{
			name:     "OK with multi-word message",
			input:    "OK Process started successfully\r\n",
			wantType: ResponseOK,
			wantMsg:  "Process started successfully",
		},
		{
			name:     "PONG",
			input:    "PONG\r\n",
			wantType: ResponsePong,
		},
		{
			name:     "ERR",
			input:    "ERR not_found process-123\r\n",
			wantType: ResponseErr,
			wantCode: "not_found",
			wantMsg:  "process-123",
		},
		{
			name:     "END",
			input:    "END\r\n",
			wantType: ResponseEnd,
		},
		{
			name:     "JSON",
			input:    "JSON 20\r\n{\"status\":\"running\"}\r\n",
			wantType: ResponseJSON,
			wantData: `{"status":"running"}`,
		},
		{
			name:     "DATA",
			input:    "DATA 5\r\nhello\r\n",
			wantType: ResponseData,
			wantData: "hello",
		},
		{
			name:     "CHUNK",
			input:    "CHUNK 10\r\nsome bytes\r\n",
			wantType: ResponseChunk,
			wantData: "some bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			got, err := parser.ParseResponse()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Type != tt.wantType {
					t.Errorf("Type = %v, want %v", got.Type, tt.wantType)
				}
				if got.Message != tt.wantMsg {
					t.Errorf("Message = %v, want %v", got.Message, tt.wantMsg)
				}
				if got.Code != tt.wantCode {
					t.Errorf("Code = %v, want %v", got.Code, tt.wantCode)
				}
				if string(got.Data) != tt.wantData {
					t.Errorf("Data = %v, want %v", string(got.Data), tt.wantData)
				}
			}
		})
	}
}

func TestFormatOK(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{"empty", "", "OK\r\n"},
		{"with message", "12345", "OK 12345\r\n"},
		{"with text", "started", "OK started\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatOK(tt.message)
			if string(got) != tt.want {
				t.Errorf("FormatOK() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatErr(t *testing.T) {
	got := FormatErr(ErrNotFound, "process-123")
	want := "ERR not_found process-123\r\n"
	if string(got) != want {
		t.Errorf("FormatErr() = %q, want %q", got, want)
	}
}

func TestFormatJSON(t *testing.T) {
	data := []byte(`{"status":"running"}`)
	got := FormatJSON(data)

	// Parse it back
	parser := NewParser(bytes.NewReader(got))
	resp, err := parser.ParseResponse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if resp.Type != ResponseJSON {
		t.Errorf("Type = %v, want JSON", resp.Type)
	}
	if string(resp.Data) != string(data) {
		t.Errorf("Data = %v, want %v", string(resp.Data), string(data))
	}
}

func TestFormatChunk(t *testing.T) {
	data := []byte("chunk data")
	got := FormatChunk(data)

	parser := NewParser(bytes.NewReader(got))
	resp, err := parser.ParseResponse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if resp.Type != ResponseChunk {
		t.Errorf("Type = %v, want CHUNK", resp.Type)
	}
	if string(resp.Data) != string(data) {
		t.Errorf("Data = %v, want %v", string(resp.Data), string(data))
	}
}

func TestFormatCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "simple",
			cmd:  &Command{Verb: "PING"},
			want: "PING\r\n",
		},
		{
			name: "with subverb",
			cmd:  &Command{Verb: "PROC", SubVerb: "STATUS", Args: []string{"test"}},
			want: "PROC STATUS test\r\n",
		},
		{
			name: "with multiple args",
			cmd:  &Command{Verb: "PROXY", SubVerb: "START", Args: []string{"dev", "http://localhost:3000", "8080"}},
			want: "PROXY START dev http://localhost:3000 8080\r\n",
		},
		{
			name: "with data",
			cmd:  &Command{Verb: "RUN-JSON", Data: []byte(`{"id":"test"}`)},
			want: "RUN-JSON 13\r\n{\"id\":\"test\"}\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCommand(tt.cmd)
			if string(got) != tt.want {
				t.Errorf("FormatCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Test WriteOK
	if err := w.WriteOK("12345"); err != nil {
		t.Errorf("WriteOK failed: %v", err)
	}
	if got := buf.String(); got != "OK 12345\r\n" {
		t.Errorf("WriteOK = %q, want %q", got, "OK 12345\r\n")
	}
	buf.Reset()

	// Test WriteErr
	if err := w.WriteErr(ErrNotFound, "proc-1"); err != nil {
		t.Errorf("WriteErr failed: %v", err)
	}
	if got := buf.String(); got != "ERR not_found proc-1\r\n" {
		t.Errorf("WriteErr = %q, want %q", got, "ERR not_found proc-1\r\n")
	}
	buf.Reset()

	// Test WritePong
	if err := w.WritePong(); err != nil {
		t.Errorf("WritePong failed: %v", err)
	}
	if got := buf.String(); got != "PONG\r\n" {
		t.Errorf("WritePong = %q, want %q", got, "PONG\r\n")
	}
	buf.Reset()

	// Test WriteEnd
	if err := w.WriteEnd(); err != nil {
		t.Errorf("WriteEnd failed: %v", err)
	}
	if got := buf.String(); got != "END\r\n" {
		t.Errorf("WriteEnd = %q, want %q", got, "END\r\n")
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that formatting and parsing are inverse operations

	// Command round-trip
	originalCmd := &Command{
		Verb:    "PROC",
		SubVerb: "STATUS",
		Args:    []string{"my-process"},
	}

	formatted := FormatCommand(originalCmd)
	parser := NewParser(bytes.NewReader(formatted))
	parsedCmd, err := parser.ParseCommand()
	if err != nil {
		t.Fatalf("Failed to parse command: %v", err)
	}

	if parsedCmd.Verb != originalCmd.Verb {
		t.Errorf("Verb = %v, want %v", parsedCmd.Verb, originalCmd.Verb)
	}
	if parsedCmd.SubVerb != originalCmd.SubVerb {
		t.Errorf("SubVerb = %v, want %v", parsedCmd.SubVerb, originalCmd.SubVerb)
	}

	// JSON response round-trip
	jsonData := []byte(`{"id":"test","state":"running","pid":12345}`)
	jsonResp := FormatJSON(jsonData)
	parser = NewParser(bytes.NewReader(jsonResp))
	parsedResp, err := parser.ParseResponse()
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if parsedResp.Type != ResponseJSON {
		t.Errorf("Type = %v, want JSON", parsedResp.Type)
	}
	if string(parsedResp.Data) != string(jsonData) {
		t.Errorf("Data = %v, want %v", string(parsedResp.Data), string(jsonData))
	}
}

func TestParseChunkedResponse(t *testing.T) {
	// Simulate a chunked streaming response
	input := "CHUNK 5\r\nhello\r\nCHUNK 6\r\n world\r\nEND\r\n"
	parser := NewParser(strings.NewReader(input))

	var collected bytes.Buffer

	for {
		resp, err := parser.ParseResponse()
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if resp.Type == ResponseEnd {
			break
		}

		if resp.Type == ResponseChunk {
			collected.Write(resp.Data)
		} else {
			t.Fatalf("Unexpected response type: %v", resp.Type)
		}
	}

	if collected.String() != "hello world" {
		t.Errorf("Collected = %q, want %q", collected.String(), "hello world")
	}
}
