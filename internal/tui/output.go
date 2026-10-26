package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

var (
	accent  = lipgloss.Color("212")
	muted   = lipgloss.Color("245")
	danger  = lipgloss.Color("203")
	success = lipgloss.Color("78")

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	labelStyle   = lipgloss.NewStyle().Foreground(muted)
	errorStyle   = lipgloss.NewStyle().Foreground(danger)
	successStyle = lipgloss.NewStyle().Foreground(success)
	mutedStyle   = lipgloss.NewStyle().Foreground(muted)
)

type KV struct {
	Label string
	Value string
}

func PrintKV(w io.Writer, title string, rows []KV) {
	width := 0
	for _, r := range rows {
		width = max(width, len(r.Label))
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	for _, r := range rows {
		if r.Value == "" {
			continue
		}
		b.WriteString("\n  " + labelStyle.Render(fmt.Sprintf("%-*s", width, r.Label)) + "  " + r.Value)
	}

	lipgloss.Fprintln(w, b.String())
}

func PrintSuccess(w io.Writer, format string, a ...any) {
	lipgloss.Fprintln(w, successStyle.Render("✓ ")+fmt.Sprintf(format, a...))
}

func PrintError(w io.Writer, err error) {
	lipgloss.Fprintln(w, errorStyle.Render("✗ "+err.Error()))
}

func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
