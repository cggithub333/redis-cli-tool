package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	fzfLeaderActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD600")) // Bright Yellow for Leader mode
	fzfResizeValStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00E5FF")) // Cyan for size values
)

// ResizeState tracks the preview window percentage and whether Leader mode is active.
type ResizeState struct {
	Size       int  // Percentage for right preview pane (default 55)
	LeaderMode bool // Whether Alt-Q leader mode is currently active
}

func getResizeStatePath(sessionID string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("redis_explore_%s.state", sessionID))
}

// InitResizeSession creates a state file for the given explorer session.
func InitResizeSession(sessionID string) (string, func()) {
	statePath := getResizeStatePath(sessionID)
	data := []byte("55:0\n")
	_ = os.WriteFile(statePath, data, 0600)

	cleanup := func() {
		_ = os.Remove(statePath)
	}
	return statePath, cleanup
}

func readResizeState(sessionID string) ResizeState {
	statePath := getResizeStatePath(sessionID)
	data, err := os.ReadFile(statePath)
	if err != nil {
		return ResizeState{Size: 55, LeaderMode: false}
	}
	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	size := 55
	mode := false
	if len(parts) >= 1 {
		if s, err := strconv.Atoi(parts[0]); err == nil && s >= 20 && s <= 85 {
			size = s
		}
	}
	if len(parts) >= 2 && parts[1] == "1" {
		mode = true
	}
	return ResizeState{Size: size, LeaderMode: mode}
}

func writeResizeState(sessionID string, state ResizeState) {
	statePath := getResizeStatePath(sessionID)
	modeStr := "0"
	if state.LeaderMode {
		modeStr = "1"
	}
	data := []byte(fmt.Sprintf("%d:%s\n", state.Size, modeStr))
	_ = os.WriteFile(statePath, data, 0600)
}

// StandardHeaderText returns the default pinned keybinds header line.
func StandardHeaderText() string {
	return fmt.Sprintf(
		"  %s %s   %s %s   %s %s   %s %s   %s %s",
		fzfKeyBadgeStyle.Render("[Enter]"), fzfDescStyle.Render("Inspect"),
		fzfKeyBadgeStyle.Render("[Ctrl-U/D]"), fzfDescStyle.Render("Scroll"),
		fzfKeyBadgeStyle.Render("[Ctrl-/]"), fzfDescStyle.Render("Toggle Preview"),
		fzfKeyBadgeStyle.Render("[Alt-←/→]"), fzfDescStyle.Render("Resize ±5%"),
		fzfKeyBadgeStyle.Render("[Esc]"), fzfDescStyle.Render("Quit"),
	)
}

// LeaderHeaderText returns the specialized resize keybinds header line when Leader mode is active.
func LeaderHeaderText(size int) string {
	menuSize := 100 - size
	return fmt.Sprintf(
		"  %s   %s %s %s   %s %s %s   %s %s",
		fzfLeaderActiveStyle.Render(fmt.Sprintf("󰌠 [RESIZE: %d%%]", size)),
		fzfKeyBadgeStyle.Render("[← / Alt-Left]"), fzfDescStyle.Render("Divider ←"), fzfResizeValStyle.Render(fmt.Sprintf("(Preview %d%%)", size)),
		fzfKeyBadgeStyle.Render("[→ / Alt-Right]"), fzfDescStyle.Render("Divider →"), fzfResizeValStyle.Render(fmt.Sprintf("(Menu %d%%)", menuSize)),
		fzfKeyBadgeStyle.Render("[Alt-Q / Alt-R / Esc]"), fzfDescStyle.Render("Done"),
	)
}

// HandleResize processes resize commands triggered by FZF's transform action.
func HandleResize(sessionID, action string) error {
	state := readResizeState(sessionID)

	switch action {
	case "toggle":
		state.LeaderMode = !state.LeaderMode
		writeResizeState(sessionID, state)
		if state.LeaderMode {
			// Rebind left/right arrows to resize and show leader header
			fmt.Printf("rebind(left,right)+change-header(%s)\n", LeaderHeaderText(state.Size))
		} else {
			// Unbind left/right arrows back to text cursor and restore standard header
			fmt.Printf("unbind(left,right)+change-header(%s)\n", StandardHeaderText())
		}

	case "esc":
		if state.LeaderMode {
			// If leader mode is active, Esc exits leader mode back to normal
			state.LeaderMode = false
			writeResizeState(sessionID, state)
			fmt.Printf("unbind(left,right)+change-header(%s)\n", StandardHeaderText())
		} else {
			// Otherwise Esc quits FZF
			fmt.Println("abort")
		}

	case "left":
		// Divider moves LEFT: Preview grows +5%, Menu shrinks -5%
		state.Size += 5
		if state.Size > 85 {
			state.Size = 85
		}
		writeResizeState(sessionID, state)
		if state.LeaderMode {
			fmt.Printf("change-preview-window(right:%d%%:wrap:border-rounded)+change-header(%s)\n",
				state.Size, LeaderHeaderText(state.Size))
		} else {
			fmt.Printf("change-preview-window(right:%d%%:wrap:border-rounded)\n", state.Size)
		}

	case "right":
		// Divider moves RIGHT: Menu grows +5%, Preview shrinks -5%
		state.Size -= 5
		if state.Size < 20 {
			state.Size = 20
		}
		writeResizeState(sessionID, state)
		if state.LeaderMode {
			fmt.Printf("change-preview-window(right:%d%%:wrap:border-rounded)+change-header(%s)\n",
				state.Size, LeaderHeaderText(state.Size))
		} else {
			fmt.Printf("change-preview-window(right:%d%%:wrap:border-rounded)\n", state.Size)
		}

	default:
		return fmt.Errorf("unknown resize action: %s", action)
	}

	return nil
}
