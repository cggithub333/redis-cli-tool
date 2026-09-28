package safety

import (
	"bytes"
	"strings"
	"testing"
)

func TestSafetyGuardrail(t *testing.T) {
	// 1. Force flag bypasses check
	if err := CheckDestructiveAction(false, true, "delete keys", nil, nil); err != nil {
		t.Fatalf("expected nil when force is true, got %v", err)
	}

	// 2. Non-TTY without force returns exit code 4
	err := CheckDestructiveAction(false, false, "delete keys", nil, nil)
	if err == nil {
		t.Fatal("expected error in non-TTY without force")
	}
	safetyErr, ok := err.(*SafetyError)
	if !ok || safetyErr.Code != ExitSafetyViolation {
		t.Fatalf("expected SafetyError with code 4, got %v", err)
	}

	// 3. TTY with affirmative confirmation "y"
	inY := strings.NewReader("y\n")
	outBuf := new(bytes.Buffer)
	if err := CheckDestructiveAction(true, false, "delete keys", inY, outBuf); err != nil {
		t.Fatalf("expected nil on 'y' confirmation, got %v", err)
	}

	// 4. TTY with negative confirmation "n"
	inN := strings.NewReader("n\n")
	outBuf.Reset()
	errN := CheckDestructiveAction(true, false, "delete keys", inN, outBuf)
	if errN == nil {
		t.Fatal("expected error on 'n' confirmation")
	}
	safetyErrN, ok := errN.(*SafetyError)
	if !ok || safetyErrN.Code != ExitSafetyViolation {
		t.Fatalf("expected SafetyError with code 4 on reject, got %v", errN)
	}
}
