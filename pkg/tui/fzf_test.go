package tui

import (
	"fmt"
	"os"
	"testing"

	fzf "github.com/junegunn/fzf/src"
)

func TestFZFOptions(t *testing.T) {
	selfBin, _ := os.Executable()
	previewCmd := fmt.Sprintf("%s inspect --context test {}", selfBin)

	args := []string{
		"--reverse",
		"--border=rounded",
		"--border-label= 󰌠 REDIS EXPLORER · test (DB 0) ",
		"--prompt=🔍 Keys > ",
		"--pointer=▶",
		"--marker=✓",
		"--info=inline",
		"--header=Context: test | [Enter] inspect key | [Esc] quit",
		"--preview-window=right:60%:wrap:border-rounded",
		"--preview=" + previewCmd,
		"--color=fg:#ECEFF4,bg:-1,hl:#FF5722,fg+:#FFFFFF,bg+:#2E3440,hl+:#FF7043",
		"--color=info:#FFA726,prompt:#FF5722,pointer:#FF5722,marker:#4CAF50,spinner:#FF5722,header:#FF8A65",
		"--color=border:#E0382B,label:#FF7043,preview-border:#E0382B,preview-label:#FF7043",
	}

	opts, err := fzf.ParseOptions(true, args)
	if err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil opts")
	}

	inputChan := make(chan string, 3)
	inputChan <- "user:1001:session"
	inputChan <- "config:system"
	inputChan <- "service:status"
	close(inputChan)

	opts.Input = inputChan

	var captured []string
	opts.Printer = func(str string) {
		captured = append(captured, str)
	}

	filterPattern := "session"
	opts.Filter = &filterPattern

	code, err := fzf.Run(opts)
	if err != nil {
		t.Fatalf("fzf.Run error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if len(captured) != 1 || captured[0] != "user:1001:session" {
		t.Fatalf("expected ['user:1001:session'], got %v", captured)
	}
}
