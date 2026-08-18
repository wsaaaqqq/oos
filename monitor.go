package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	colMTime = 9
	colMRole = 11
	colMIn   = 10
	colMOut  = 10
	colMCost = 10
)

type monitorModel struct {
	dbPath    string
	sessionID string
	info      *MonitorSession
	err       error
	width     int
	height    int
}

type monitorLoadedMsg struct {
	info *MonitorSession
	err  error
}

type monitorTickMsg time.Time

func (m monitorModel) Init() tea.Cmd {
	return tea.Batch(
		loadMonitorCmd(m.dbPath, m.sessionID),
		monitorTick(),
	)
}

func monitorTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return monitorTickMsg(t)
	})
}

func loadMonitorCmd(dbPath, sessionID string) tea.Cmd {
	return func() tea.Msg {
		info, err := LoadMonitor(dbPath, sessionID)
		return monitorLoadedMsg{info, err}
	}
}

func (m monitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case monitorLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.info = msg.info
		}
		return m, nil

	case monitorTickMsg:
		return m, tea.Batch(loadMonitorCmd(m.dbPath, m.sessionID), monitorTick())

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
			return m, tea.Quit
		}
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'q' || msg.Runes[0] == 'Q') {
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

func (m monitorModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	if m.info == nil {
		return "Loading monitor..."
	}

	w := m.width
	if w < 60 {
		w = 60
	}

	var b strings.Builder

	// header: session title + agent + model
	header := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(w - 2)

	title := truncateCols(m.info.Title, 40)
	meta := fmt.Sprintf("%s  %s  %s", title, m.info.Agent, m.info.ModelID)
	b.WriteString(header.Render(meta))
	b.WriteString("\n\n")

	// column header
	colHdr := fmt.Sprintf("%-*s %-*s %*s %*s %*s",
		colMTime, "TIME",
		colMRole, "ROLE",
		colMIn, "IN",
		colMOut, "OUT",
		colMCost, "COST")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(colHdr))
	b.WriteString("\n")

	// messages
	visRows := m.height - 12
	if visRows < 5 {
		visRows = 5
	}
	msgs := m.info.Messages
	if len(msgs) > visRows {
		msgs = msgs[len(msgs)-visRows:]
	}
	for _, msg := range msgs {
		timeStr := time.UnixMilli(msg.TimeCreated).Format("15:04:05")
		role := msg.Role
		if role == "" {
			role = "-"
		}
		line := fmt.Sprintf("%-*s %-*s %*s %*s %*s",
			colMTime, timeStr,
			colMRole, role,
			colMIn, fmtToken(msg.TokensIn),
			colMOut, fmtToken(msg.TokensOut),
			colMCost, fmt.Sprintf("%.4f", msg.Cost))
		b.WriteString(line)
		b.WriteString("\n")
	}

	// total
	total := fmt.Sprintf("TOTAL  in %s  out %s  cost %.4f",
		fmtToken(m.info.TokensIn), fmtToken(m.info.TokensOut), m.info.Cost)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(total))

	// footer
	footer := fmt.Sprintf("%s  (2s refresh, Ctrl+C quit)", m.sessionID)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(footer))

	return b.String()
}

func fmtToken(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func runMonitor(dbPath, sessionID string) error {
	m := monitorModel{
		dbPath:    dbPath,
		sessionID: sessionID,
		width:     80,
		height:    24,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
