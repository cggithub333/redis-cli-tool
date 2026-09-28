package safety

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// CheckDestructiveAction guards destructive operations (like DEL with patterns or flushing)
// requiring --force in non-interactive environments, and user confirmation in interactive TTYs.
func CheckDestructiveAction(isTTY bool, force bool, actionDesc string, in io.Reader, out io.Writer) error {
	if force {
		return nil
	}

	if !isTTY {
		return &SafetyError{
			Code:    ExitSafetyViolation,
			Message: fmt.Sprintf("destructive action (%s) requires --force in non-interactive mode", actionDesc),
		}
	}

	if out != nil {
		fmt.Fprintf(out, "Warning: %s. Are you sure you want to proceed? [y/N]: ", actionDesc)
	}

	reader := bufio.NewReader(in)
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return &SafetyError{
			Code:    ExitSafetyViolation,
			Message: fmt.Sprintf("failed to read confirmation: %v", err),
		}
	}

	trimmed := strings.ToLower(strings.TrimSpace(input))
	if trimmed == "y" || trimmed == "yes" {
		return nil
	}

	return &SafetyError{
		Code:    ExitSafetyViolation,
		Message: "operation aborted by user",
	}
}
