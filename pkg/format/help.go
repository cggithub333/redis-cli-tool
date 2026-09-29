package format

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// Warm color palette inspired by Redis official branding
	brandRed       = lipgloss.Color("#E0382B") // Classic Redis Red
	warmOrangeRed  = lipgloss.Color("#FF5722") // Vibrant Orange-Red
	warmCoral      = lipgloss.Color("#FF7043") // Warm Coral
	warmAmber      = lipgloss.Color("#FFA726") // Warm Amber Gold
	textWhite      = lipgloss.Color("#F5F5F5") // Crisp white text
	textMuted      = lipgloss.Color("#9E9E9E") // Muted gray
	textDim        = lipgloss.Color("#757575") // Dim gray
	defaultValCol  = lipgloss.Color("#78909C") // Subtle teal-gray for defaults

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(brandRed).
			Padding(0, 1)

	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warmCoral)

	taglineStyle = lipgloss.NewStyle().
			Foreground(textMuted)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warmOrangeRed)

	commandNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warmCoral)

	flagNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(warmAmber)

	flagTypeStyle = lipgloss.NewStyle().
			Foreground(textDim)

	descriptionStyle = lipgloss.NewStyle().
				Foreground(textWhite)

	defaultStyle = lipgloss.NewStyle().
			Foreground(defaultValCol)

	footerStyle = lipgloss.NewStyle().
			Foreground(warmCoral)
)

// SetupHelp configures colorful warm-themed help for cobra commands.
func SetupHelp(rootCmd *cobra.Command) {
	rootCmd.SetHelpFunc(HelpFunc)
}

// HelpFunc implements the Cobra HelpFunc signature with warm-colored Lipgloss styling.
func HelpFunc(cmd *cobra.Command, args []string) {
	out := cmd.OutOrStdout()
	isTTY := false
	if f, ok := out.(*os.File); ok {
		isTTY = isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}

	if !isTTY {
		// Output clean plain text when redirected or piped to preserve script predictability
		fmt.Fprint(out, cmd.UsageString())
		return
	}

	fmt.Fprint(out, RenderHelp(cmd))
}

// RenderHelp returns the stylized, warm-colored help text for a Cobra command.
func RenderHelp(cmd *cobra.Command) string {
	var b bytes.Buffer

	// 1. Header / Banner
	if cmd.Parent() == nil {
		b.WriteString(badgeStyle.Render("REDIS") + " " + headerTitleStyle.Render("CLI Tool") + " " + taglineStyle.Render("— High-Performance Redis Engine") + "\n")
		if cmd.Long != "" {
			b.WriteString(descriptionStyle.Render(cmd.Long) + "\n")
		} else if cmd.Short != "" {
			b.WriteString(descriptionStyle.Render(cmd.Short) + "\n")
		}
	} else {
		b.WriteString(badgeStyle.Render("REDIS "+strings.ToUpper(cmd.Name())) + " " + headerTitleStyle.Render("Command") + "\n")
		desc := cmd.Long
		if desc == "" {
			desc = cmd.Short
		}
		if desc != "" {
			b.WriteString(descriptionStyle.Render(desc) + "\n")
		}
	}

	// 2. Usage
	b.WriteString("\n" + sectionHeaderStyle.Render("USAGE") + "\n")
	useLine := cmd.UseLine()
	// Highlight "redis" and command path in usage
	parts := strings.SplitN(useLine, " ", 2)
	styledUse := commandNameStyle.Render(parts[0])
	if len(parts) > 1 {
		styledUse += " " + taglineStyle.Render(parts[1])
	}
	b.WriteString("  " + styledUse + "\n")

	// 3. Aliases
	if len(cmd.Aliases) > 0 {
		b.WriteString("\n" + sectionHeaderStyle.Render("ALIASES") + "\n")
		b.WriteString("  " + taglineStyle.Render(strings.Join(cmd.Aliases, ", ")) + "\n")
	}

	// 4. Available Commands
	var visibleCommands []*cobra.Command
	maxCmdLen := 0
	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() && !c.Hidden {
			visibleCommands = append(visibleCommands, c)
			if len(c.Name()) > maxCmdLen {
				maxCmdLen = len(c.Name())
			}
		}
	}

	if len(visibleCommands) > 0 {
		b.WriteString("\n" + sectionHeaderStyle.Render("COMMANDS") + "\n")
		for _, c := range visibleCommands {
			paddedName := fmt.Sprintf("%-*s", maxCmdLen+2, c.Name())
			b.WriteString("  " + commandNameStyle.Render(paddedName) + " " + descriptionStyle.Render(c.Short) + "\n")
		}
	}

	// 5. Flags (Local & Inherited)
	renderFlags := func(sectionTitle string, fs *pflag.FlagSet) {
		if fs == nil || !fs.HasAvailableFlags() {
			return
		}

		type flagItem struct {
			flagStr string
			desc    string
			defStr  string
		}

		var items []flagItem
		maxFlagLen := 0

		fs.VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			var flagStr string
			if f.Shorthand != "" {
				flagStr = fmt.Sprintf("-%s, --%s", f.Shorthand, f.Name)
			} else {
				flagStr = fmt.Sprintf("    --%s", f.Name)
			}
			if f.Value.Type() != "bool" {
				flagStr += fmt.Sprintf(" <%s>", f.Value.Type())
			}

			if len(flagStr) > maxFlagLen {
				maxFlagLen = len(flagStr)
			}

			def := ""
			if f.DefValue != "" && f.DefValue != "false" && f.DefValue != `""` && f.DefValue != "0" {
				def = fmt.Sprintf("(default %s)", f.DefValue)
			}

			items = append(items, flagItem{
				flagStr: flagStr,
				desc:    f.Usage,
				defStr:  def,
			})
		})

		if len(items) == 0 {
			return
		}

		b.WriteString("\n" + sectionHeaderStyle.Render(sectionTitle) + "\n")
		for _, it := range items {
			padded := fmt.Sprintf("%-*s", maxFlagLen+2, it.flagStr)
			line := "  " + flagNameStyle.Render(padded) + " " + descriptionStyle.Render(it.desc)
			if it.defStr != "" {
				line += " " + defaultStyle.Render(it.defStr)
			}
			b.WriteString(line + "\n")
		}
	}

	renderFlags("FLAGS", cmd.NonInheritedFlags())
	if cmd.HasInheritedFlags() {
		renderFlags("GLOBAL FLAGS", cmd.InheritedFlags())
	}

	// 6. Examples
	if cmd.Example != "" {
		b.WriteString("\n" + sectionHeaderStyle.Render("EXAMPLES") + "\n")
		for _, line := range strings.Split(strings.TrimSpace(cmd.Example), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				b.WriteString("  " + taglineStyle.Render(line) + "\n")
			} else {
				b.WriteString("  " + commandNameStyle.Render(line) + "\n")
			}
		}
	}

	// 7. Footer
	if len(visibleCommands) > 0 {
		b.WriteString("\n" + footerStyle.Render("Use \"redis [command] --help\" for more information about a command.") + "\n")
	}

	return b.String()
}
