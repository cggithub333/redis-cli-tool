package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	fzf "github.com/junegunn/fzf/src"
	"redis-cli-tool/pkg/client"
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

func TestFZFFormattedOptions(t *testing.T) {
	args := []string{
		"--ansi",
		"--delimiter=\t",
		"--with-nth=2",
		"--layout=reverse-list",
		"--prompt=󰍉 Filter Keys > ",
		"--no-separator",
		"--header-border=bottom",
		"--preview=echo {1}",
		"--bind=ctrl-d:preview-page-down,ctrl-u:preview-page-up,ctrl-/:toggle-preview",
	}

	opts, err := fzf.ParseOptions(true, args)
	if err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}

	inputChan := make(chan string, 2)
	inputChan <- "user:profile:1001\t STRING    persist    704 B  │ user:profile:1001"
	inputChan <- "user:session:h-user\t  HASH      29m48s    296 B  │ user:session:h-user"
	close(inputChan)

	opts.Input = inputChan

	var captured []string
	opts.Printer = func(str string) {
		captured = append(captured, str)
	}

	filterPattern := "HASH"
	opts.Filter = &filterPattern

	code, err := fzf.Run(opts)
	if err != nil {
		t.Fatalf("fzf.Run error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if len(captured) != 1 {
		t.Fatalf("expected 1 captured, got %d: %v", len(captured), captured)
	}

	// Verify key extraction from tab-separated line
	key := captured[0]
	if idx := len("user:session:h-user"); len(key) >= idx && key[:idx] == "user:session:h-user" {
		// Key extracted properly
	} else {
		t.Fatalf("expected key starting with user:session:h-user, got %q", key)
	}
}

func TestFZFColumnAlignment(t *testing.T) {
	// Strip ANSI helper
	stripANSI := func(s string) string {
		var b strings.Builder
		inEsc := false
		for i := 0; i < len(s); i++ {
			if s[i] == '\x1b' {
				inEsc = true
				continue
			}
			if inEsc {
				if s[i] == 'm' {
					inEsc = false
				}
				continue
			}
			b.WriteByte(s[i])
		}
		return b.String()
	}

	runeIndex := func(s string, target rune) int {
		for i, r := range []rune(s) {
			if r == target {
				return i
			}
		}
		return -1
	}

	headerCols := stripANSI(fzfHeaderColsStyle.Render("  TYPE      TTL       MEMORY │ KEY"))
	headerRuler := stripANSI(fzfHeaderRulerStyle.Render("─────────────────────────────┼────────────────────────────────────────────"))

	idxHeaderDiv := runeIndex(headerCols, '│')
	if idxHeaderDiv != 29 {
		t.Fatalf("expected header '│' at rune index 29, got %d in %q", idxHeaderDiv, headerCols)
	}

	idxRulerCross := runeIndex(headerRuler, '┼')
	if idxRulerCross != 29 {
		t.Fatalf("expected ruler '┼' at rune index 29, got %d in %q", idxRulerCross, headerRuler)
	}

	testCases := []client.KeyMetadata{
		{Key: "user:profile:1001", Type: "string", TTL: -1, Memory: 704},
		{Key: "user:session:h-user", Type: "hash", TTL: 29 * time.Minute, Memory: 296},
		{Key: "queue:jobs:email", Type: "list", TTL: 45 * time.Second, Memory: 1024 * 1024},
		{Key: "security:blacklist", Type: "set", TTL: -1, Memory: 144},
		{Key: "leaderboard:points", Type: "zset", TTL: 3600 * time.Second, Memory: 2048},
		{Key: "events:audit", Type: "stream", TTL: -1, Memory: 976},
	}

	for _, tc := range testCases {
		item := FormatFZFItem(tc)
		parts := strings.SplitN(item, "\t", 2)
		if len(parts) != 2 {
			t.Fatalf("expected tab-separated item, got %q", item)
		}
		cleanDisplay := stripANSI(parts[1])
		idxBodyDiv := runeIndex(cleanDisplay, '│')
		if idxBodyDiv != 29 {
			t.Errorf("for key %s (type %s), expected '│' at index 29, got %d in %q", tc.Key, tc.Type, idxBodyDiv, cleanDisplay)
		}
	}
}

