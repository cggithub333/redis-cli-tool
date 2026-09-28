package safety

import (
	"encoding/json"
	"fmt"
	"io"
)

// Semantic Exit Codes (0 to 5)
const (
	ExitOK              = 0 // Success
	ExitConnectionError = 1 // Socket failure / connection timeout
	ExitAuthError       = 2 // Authentication / permission failure
	ExitKeyNotFound     = 3 // Key does not exist
	ExitSafetyViolation = 4 // Destructive action attempted without --force in non-TTY
	ExitSyntaxError     = 5 // Syntax error / invalid flag / non-TTY TUI invocation
)

// SafetyError represents a typed CLI error with a semantic exit code
type SafetyError struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
}

func (e *SafetyError) Error() string {
	return e.Message
}

// NewSafetyError creates a new SafetyError with specified code and message
func NewSafetyError(code int, msg string) *SafetyError {
	return &SafetyError{
		Code:    code,
		Message: msg,
	}
}

// EmitJSON serializes the error as JSON to the provided writer
func (e *SafetyError) EmitJSON(w io.Writer) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
