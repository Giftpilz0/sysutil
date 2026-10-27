package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/giftpilz0/sysutil/internal/snapper"
)

// fakeSnapper emulates the snapper commands used by the TUI, keeping its
// snapshots in one CSV file per config inside dir.
const fakeSnapper = `#!/bin/sh
dir=$(dirname "$0")
if [ "$1" = "--csvout" ]; then shift; fi
if [ "$1" = "list-configs" ]; then printf 'config,subvolume\nroot,/\nhome,/home\n'; exit 0; fi
cfg=$2; cmd=$3; shift 3
db="$dir/$cfg.csv"; touch "$db"
case $cmd in
list)
	echo "number,type,pre-number,date,user,cleanup,description"
	echo "0,single,,,root,,current"
	cat "$db";;
create)
	n=$(( $(cut -d, -f1 "$db" | sort -n | tail -1) + 1 ))
	echo "$n,single,,2026-10-26 12:00:00,root,,$3" >> "$db"
	echo "$n";;
delete)
	for n in "$@"; do grep -v "^$n," "$db" > "$db.tmp"; mv "$db.tmp" "$db"; done;;
undochange)
	echo "$1" > "$dir/undo";;
*)
	echo "unknown command $cmd" >&2; exit 1;;
esac
`

type harness struct {
	t *testing.T
	m tea.Model
}

func newHarness(t *testing.T) (*harness, string) {
	t.Helper()

	dir := t.TempDir()
	bin := filepath.Join(dir, "snapper")
	if err := os.WriteFile(bin, []byte(fakeSnapper), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.csv"), []byte("1,pre,,2026-10-01 10:00:00,root,number,zypp\n2,post,1,2026-10-01 10:01:00,root,number,zypp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := snapper.Bin
	snapper.Bin = bin
	t.Cleanup(func() { snapper.Bin = old })

	m := newSnapModel(context.Background())
	// A blinking cursor would make every key press wait for the blink timer.
	styles := m.input.Styles()
	styles.Cursor.Blink = false
	m.input.SetStyles(styles)

	h := &harness{t: t, m: m}
	h.send(tea.WindowSizeMsg{Width: 120, Height: 30})
	h.run(h.m.Init())

	return h, dir
}

// send delivers msg and runs all resulting commands.
func (h *harness) send(msg tea.Msg) {
	var cmd tea.Cmd
	h.m, cmd = h.m.Update(msg)
	h.run(cmd)
}

func (h *harness) key(keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		default:
			r := []rune(k)[0]
			msg = tea.KeyPressMsg{Code: r, Text: k}
		}
		h.send(msg)
	}
}

func (h *harness) typeText(s string) {
	for _, r := range s {
		h.send(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

// run executes cmd and feeds its messages back, dropping spinner ticks.
func (h *harness) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	var msg tea.Msg
	select {
	case msg = <-done:
	case <-time.After(5 * time.Second):
		h.t.Fatal("command timed out")
	}

	switch msg := msg.(type) {
	case nil, spinner.TickMsg:
	case tea.BatchMsg:
		for _, c := range msg {
			h.run(c)
		}
	default:
		h.send(msg)
	}
}

func (h *harness) view() string {
	return h.m.(snapModel).render()
}

func (h *harness) contains(want ...string) {
	h.t.Helper()
	v := h.view()
	for _, w := range want {
		if !strings.Contains(v, w) {
			h.t.Fatalf("view does not contain %q:\n%s", w, v)
		}
	}
}

func TestSnapTUI(t *testing.T) {
	h, dir := newHarness(t)

	h.contains("root", "home", "2 snapshots", "post←1", "zypp")
	if lines := strings.Count(h.view(), "\n") + 1; lines != 30 {
		t.Errorf("view has %d lines, want 30", lines)
	}

	// Create a snapshot.
	h.key("n")
	h.contains("New snapshot:")
	h.typeText("before update")
	h.key("enter")
	h.contains("Created snapshot 3", "before update", "3 snapshots")

	// Undo asks first, cancelling does nothing.
	h.key("u")
	h.contains("Revert all file changes in / made since snapshot 3 (before update)")
	h.key("n")
	h.contains("Cancelled")
	if _, err := os.Stat(filepath.Join(dir, "undo")); err == nil {
		t.Fatal("undochange ran despite cancelling")
	}

	// Mark the two oldest snapshots and delete them together.
	h.key("down", "space", "space", "d")
	h.contains("Delete 2 snapshots 1, 2")
	h.key("y")
	h.contains("Deleted 2 snapshots 1, 2", "1 snapshots", "before update")
	if strings.Contains(h.view(), "zypp") {
		t.Fatal("deleted snapshots are still listed")
	}

	// Switch to the home config.
	h.key("tab")
	h.contains("/home · 0 snapshots", "No snapshots yet")
}

func TestSnapTUIError(t *testing.T) {
	old := snapper.Bin
	snapper.Bin = "/nonexistent/snapper"
	t.Cleanup(func() { snapper.Bin = old })

	h := &harness{t: t, m: newSnapModel(context.Background())}
	h.send(tea.WindowSizeMsg{Width: 80, Height: 20})
	h.run(h.m.Init())

	h.contains("snapper", "press q to quit")
}
