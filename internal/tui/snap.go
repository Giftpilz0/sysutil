package tui

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/giftpilz0/sysutil/internal/snapper"
)

func Snap(ctx context.Context) error {
	_, err := tea.NewProgram(newSnapModel(ctx), tea.WithContext(ctx)).Run()

	return err
}

type snapKeys struct {
	Up, Down, NextConfig, PrevConfig    key.Binding
	Create, Delete, Mark, Undo, Refresh key.Binding
	Help, Quit                          key.Binding
}

func (k snapKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Create, k.Delete, k.Mark, k.Undo, k.NextConfig, k.Help, k.Quit}
}

func (k snapKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.NextConfig, k.PrevConfig},
		{k.Create, k.Delete, k.Mark, k.Undo},
		{k.Refresh, k.Help, k.Quit},
	}
}

var defaultSnapKeys = snapKeys{
	Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	NextConfig: key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab/→", "next config")),
	PrevConfig: key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("shift+tab/←", "prev config")),
	Create:     key.NewBinding(key.WithKeys("n", "c"), key.WithHelp("n", "new")),
	Delete:     key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "delete")),
	Mark:       key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "mark")),
	Undo:       key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo changes since")),
	Refresh:    key.NewBinding(key.WithKeys("r", "ctrl+r"), key.WithHelp("r", "refresh")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:       key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
}

type snapState int

const (
	stateBrowse snapState = iota
	stateCreate
	stateConfirm
)

type (
	configsMsg struct {
		configs []snapper.Config
		err     error
	}
	snapshotsMsg struct {
		config string
		snaps  []snapper.Snapshot
		err    error
	}
	opDoneMsg struct {
		status string
		err    error
	}
)

type snapModel struct {
	ctx  context.Context
	keys snapKeys

	configs []snapper.Config
	current int
	snaps   []snapper.Snapshot
	marked  map[int]bool

	table   table.Model
	input   textinput.Model
	spinner spinner.Model
	help    help.Model

	state         snapState
	pending       *pendingOp
	busy          string
	status        string
	statusIsError bool
	fatal         error
	width, height int
}

type pendingOp struct {
	prompt     string
	busy       string
	clearMarks bool
	run        func(ctx context.Context) (string, error)
}

func newSnapModel(ctx context.Context) snapModel {
	tkm := table.DefaultKeyMap()
	tkm.PageDown.SetKeys("pgdown", "f")
	tkm.HalfPageUp.SetKeys("ctrl+u")
	tkm.HalfPageDown.SetKeys("ctrl+d")

	t := table.New(table.WithFocused(true), table.WithKeyMap(tkm))
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Foreground(muted)
	styles.Selected = styles.Selected.Foreground(accent)
	t.SetStyles(styles)

	in := textinput.New()
	in.Placeholder = "describe the snapshot"
	in.Prompt = ""
	in.CharLimit = 200

	return snapModel{
		ctx:     ctx,
		keys:    defaultSnapKeys,
		marked:  map[int]bool{},
		table:   t,
		input:   in,
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(accent))),
		help:    help.New(),
		busy:    "Loading configurations",
	}
}

func (m snapModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadConfigs())
}

func (m snapModel) config() snapper.Config {
	return m.configs[m.current]
}

func (m snapModel) loadConfigs() tea.Cmd {
	return func() tea.Msg {
		configs, err := snapper.Configs(m.ctx)
		return configsMsg{configs, err}
	}
}

func (m snapModel) loadSnapshots() tea.Cmd {
	name := m.config().Name
	return func() tea.Msg {
		snaps, err := snapper.List(m.ctx, name)
		return snapshotsMsg{name, snaps, err}
	}
}

func (m *snapModel) start(busy string, fn func(ctx context.Context) (string, error)) tea.Cmd {
	m.busy = busy
	m.status = ""
	ctx := m.ctx
	return func() tea.Msg {
		status, err := fn(ctx)
		return opDoneMsg{status, err}
	}
}

func (m snapModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case configsMsg:
		m.busy = ""
		switch {
		case msg.err != nil:
			m.fatal = msg.err
		case len(msg.configs) == 0:
			m.fatal = fmt.Errorf("no snapper configurations found, create one with: snapper -c root create-config /")
		default:
			m.configs = msg.configs
			m.busy = "Loading snapshots"
			return m, m.loadSnapshots()
		}
		return m, nil

	case snapshotsMsg:
		if len(m.configs) == 0 || msg.config != m.config().Name {
			return m, nil // stale response for a config we switched away from
		}
		m.busy = ""
		if msg.err != nil {
			m.setStatus(msg.err.Error(), true)
			return m, nil
		}
		m.setSnapshots(msg.snaps)
		return m, nil

	case opDoneMsg:
		m.busy = ""
		if msg.err != nil {
			m.setStatus(msg.err.Error(), true)
		} else {
			m.setStatus(msg.status, false)
		}
		return m, m.loadSnapshots()

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.state {
		case stateCreate:
			return m.updateCreate(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		default:
			return m.updateBrowse(msg)
		}
	}

	return m, nil
}

func (m snapModel) updateCreate(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateBrowse
		m.input.Blur()
		return m, nil
	case "enter":
		desc := strings.TrimSpace(m.input.Value())
		if desc == "" {
			m.setStatus("Please enter a description", true)
			return m, nil
		}
		m.state = stateBrowse
		m.input.Blur()
		config := m.config().Name
		return m, m.start("Creating snapshot", func(ctx context.Context) (string, error) {
			n, err := snapper.Create(ctx, config, desc)
			return fmt.Sprintf("Created snapshot %d", n), err
		})
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m snapModel) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		op := m.pending
		m.state, m.pending = stateBrowse, nil
		if op.clearMarks {
			clear(m.marked)
			m.refreshRows()
		}
		return m, m.start(op.busy, op.run)
	case "n", "N", "esc", "q", "enter":
		m.state, m.pending = stateBrowse, nil
		m.setStatus("Cancelled", false)
	}

	return m, nil
}

func (m snapModel) updateBrowse(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.fatal != nil {
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		if msg.String() == "esc" && len(m.marked) > 0 {
			clear(m.marked)
			m.refreshRows()
			return m, nil
		}
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.layout()
		return m, nil

	case key.Matches(msg, m.keys.NextConfig, m.keys.PrevConfig):
		if len(m.configs) < 2 {
			return m, nil
		}
		step := 1
		if key.Matches(msg, m.keys.PrevConfig) {
			step = len(m.configs) - 1
		}
		m.current = (m.current + step) % len(m.configs)
		m.snaps = nil
		clear(m.marked)
		m.refreshRows()
		m.status = ""
		m.busy = "Loading snapshots"
		return m, m.loadSnapshots()

	case key.Matches(msg, m.keys.Refresh):
		if len(m.configs) == 0 {
			return m, nil
		}
		m.busy = "Refreshing"
		return m, m.loadSnapshots()
	}

	if len(m.configs) == 0 {
		return m, nil
	}

	actions := []key.Binding{m.keys.Create, m.keys.Delete, m.keys.Undo}
	if m.busy != "" && key.Matches(msg, actions...) {
		m.setStatus("Please wait until the current operation finished", true)
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Create):
		m.state = stateCreate
		m.input.Reset()
		m.status = ""
		return m, m.input.Focus()

	case key.Matches(msg, m.keys.Mark):
		if s, ok := m.selected(); ok {
			if m.marked[s.Number] {
				delete(m.marked, s.Number)
			} else {
				m.marked[s.Number] = true
			}
			m.refreshRows()
			m.table.MoveDown(1)
		}
		return m, nil

	case key.Matches(msg, m.keys.Delete):
		targets := m.deleteTargets()
		if len(targets) == 0 {
			return m, nil
		}
		config := m.config().Name
		ids := joinInts(targets)
		noun := "snapshot"
		if len(targets) > 1 {
			noun = fmt.Sprintf("%d snapshots", len(targets))
		}
		m.ask(pendingOp{
			prompt:     fmt.Sprintf("Delete %s %s from %q?", noun, ids, config),
			busy:       "Deleting " + ids,
			clearMarks: true,
			run: func(ctx context.Context) (string, error) {
				return "Deleted " + noun + " " + ids, snapper.Delete(ctx, config, targets...)
			},
		})
		return m, nil

	case key.Matches(msg, m.keys.Undo):
		s, ok := m.selected()
		if !ok {
			return m, nil
		}
		cfg := m.config()
		name := strconv.Itoa(s.Number)
		if s.Description != "" {
			name += " (" + s.Description + ")"
		}
		m.ask(pendingOp{
			prompt: fmt.Sprintf("Revert all file changes in %s made since snapshot %s?", cfg.Subvolume, name),
			busy:   fmt.Sprintf("Undoing changes since snapshot %d", s.Number),
			run: func(ctx context.Context) (string, error) {
				return fmt.Sprintf("Reverted changes since snapshot %d", s.Number), snapper.UndoChange(ctx, cfg.Name, s.Number)
			},
		})
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *snapModel) ask(op pendingOp) {
	m.state = stateConfirm
	m.pending = &op
	m.status = ""
}

func (m *snapModel) setStatus(s string, isErr bool) {
	m.status, m.statusIsError = s, isErr
}

func (m snapModel) selected() (snapper.Snapshot, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.snaps) {
		return snapper.Snapshot{}, false
	}
	return m.snaps[i], true
}

func (m snapModel) deleteTargets() []int {
	if len(m.marked) > 0 {
		var nums []int
		for n := range m.marked {
			nums = append(nums, n)
		}
		slices.Sort(nums)
		return nums
	}

	if s, ok := m.selected(); ok {
		return []int{s.Number}
	}

	return nil
}

func (m *snapModel) setSnapshots(snaps []snapper.Snapshot) {
	slices.Reverse(snaps)
	m.snaps = snaps

	for n := range m.marked {
		if !slices.ContainsFunc(snaps, func(s snapper.Snapshot) bool { return s.Number == n }) {
			delete(m.marked, n)
		}
	}

	m.layout()
	m.table.SetCursor(min(m.table.Cursor(), max(len(snaps)-1, 0)))
}

func (m *snapModel) refreshRows() {
	rows := make([]table.Row, len(m.snaps))
	for i, s := range m.snaps {
		mark := " "
		if m.marked[s.Number] {
			mark = "●"
		}
		typ := s.Type
		if s.Type == "post" && s.PreNumber != "" {
			typ = "post←" + s.PreNumber
		}
		rows[i] = table.Row{mark, strconv.Itoa(s.Number), typ, s.Date, s.User, s.Cleanup, s.Description}
	}
	m.table.SetRows(rows)
}

func (m *snapModel) layout() {
	if m.width == 0 {
		return
	}

	dateWidth := 10
	for _, s := range m.snaps {
		dateWidth = max(dateWidth, lipgloss.Width(s.Date))
	}
	dateWidth = min(dateWidth, 28)

	cols := []table.Column{
		{Title: " ", Width: 1},
		{Title: "#", Width: 5},
		{Title: "Type", Width: 9},
		{Title: "Date", Width: dateWidth},
		{Title: "User", Width: 8},
		{Title: "Cleanup", Width: 8},
		{Title: "Description", Width: 0},
	}

	// Each cell has one column of padding on either side; the border takes two more.
	used := 2
	for _, c := range cols[:len(cols)-1] {
		used += c.Width + 2
	}
	cols[len(cols)-1].Width = max(m.width-used-2, 10)

	m.table.SetColumns(cols)
	m.table.SetWidth(m.width - 2)
	m.refreshRows()

	m.help.SetWidth(m.width)
	chrome := lipgloss.Height(m.headerView()) + 2 + 1 + lipgloss.Height(m.help.View(m.keys))
	m.table.SetHeight(max(m.height-chrome, 3))
}

func (m snapModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "sysutil · snapper"

	return v
}

func (m snapModel) render() string {
	if m.width == 0 {
		return ""
	}

	if m.fatal != nil {
		lines := []string{titleStyle.Render("Snapper"), "", errorStyle.Render(m.fatal.Error())}
		if os.Geteuid() != 0 {
			lines = append(lines, mutedStyle.Render("Snapper usually needs root privileges, try running with sudo."))
		}
		lines = append(lines, "", mutedStyle.Render("press q to quit"))

		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(muted).
		Width(m.width)

	var body string
	switch {
	case len(m.configs) == 0:
		body = ""
	case len(m.snaps) == 0 && m.busy == "":
		body = mutedStyle.Render("No snapshots yet, press n to create one.")
	default:
		body = m.table.View()
	}
	body = box.Height(m.table.Height() + 1 + 2).Render(body)

	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		body,
		m.footerView(),
		m.help.View(m.keys),
	)
}

func (m snapModel) headerView() string {
	tabs := []string{titleStyle.Render("Snapper")}
	for i, c := range m.configs {
		if i == m.current {
			tabs = append(tabs, lipgloss.NewStyle().Bold(true).Reverse(true).Padding(0, 1).Render(c.Name))
		} else {
			tabs = append(tabs, mutedStyle.Padding(0, 1).Render(c.Name))
		}
	}

	line := strings.Join(tabs, " ")
	if len(m.configs) > 0 {
		info := fmt.Sprintf("%s · %d snapshots", m.config().Subvolume, len(m.snaps))
		if len(m.marked) > 0 {
			info += fmt.Sprintf(" · %d marked", len(m.marked))
		}
		gap := max(m.width-lipgloss.Width(line)-lipgloss.Width(info), 1)
		line += strings.Repeat(" ", gap) + mutedStyle.Render(info)
	}

	return line
}

func (m snapModel) footerView() string {
	switch {
	case m.state == stateCreate:
		return titleStyle.Render("New snapshot: ") + m.input.View() + mutedStyle.Render("  (enter to create, esc to cancel)")
	case m.state == stateConfirm:
		return errorStyle.Bold(true).Render(m.pending.prompt) + mutedStyle.Render("  [y/N]")
	case m.busy != "":
		return m.spinner.View() + " " + m.busy + "…"
	case m.status != "" && m.statusIsError:
		return errorStyle.Render("✗ " + m.status)
	case m.status != "":
		return successStyle.Render("✓ " + m.status)
	}

	return ""
}

func joinInts(nums []int) string {
	s := make([]string, len(nums))
	for i, n := range nums {
		s[i] = strconv.Itoa(n)
	}

	return strings.Join(s, ", ")
}
