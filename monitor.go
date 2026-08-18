package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	colMTime  = 9
	colMWho   = 5
	colMTitle = 20
	colMModel = 30
	colMIn    = 60
)

type monitorModel struct {
	dbPath    string
	sessionID string
	info      *MonitorSession
	err       error
	width     int
	height    int
	offset    int // scroll offset, 0 = newest at bottom
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
		// scroll: up/down/pageup/pagedown
		switch msg.Type {
		case tea.KeyUp:
			m.offset++
		case tea.KeyDown:
			if m.offset > 0 {
				m.offset--
			}
		case tea.KeyPgUp:
			m.offset += 10
		case tea.KeyPgDown:
			if m.offset >= 10 {
				m.offset -= 10
			} else {
				m.offset = 0
			}
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
	if m.info.ID == "" {
		title = "ALL SESSIONS"
	}
	meta := fmt.Sprintf("%s  %s  %s", title, m.info.Agent, m.info.ModelID)
	b.WriteString(header.Render(meta))
	b.WriteString("\n\n")

	// column header
	colHdr := padCols("TIME", colMTime) + " " +
		padCols("WHO", colMWho) + " " +
		padCols("SESSION TITLE", colMTitle) + " " +
		padCols("MODEL", colMModel) + " " +
		padCols("INPUT (first 50 chars)", colMIn)
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(colHdr))
	b.WriteString("\n")

	// messages (scroll window, bottom-anchored)
	visRows := m.height - 13
	if visRows < 5 {
		visRows = 5
	}
	all := m.info.Messages
	total := len(all)
	if m.offset > total-visRows {
		m.offset = total - visRows
	}
	if m.offset < 0 {
		m.offset = 0
	}
	start := total - m.offset - visRows
	if start < 0 {
		start = 0
	}
	end := total - m.offset
	if end > total {
		end = total
	}
	msgs := all[start:end]
	for _, msg := range msgs {
		timeStr := time.UnixMilli(msg.TimeCreated).Format("15:04:05")
		who := "assistant"
		if msg.Role == "user" {
			who = "user"
		} else if msg.Agent != "" {
			who = msg.Agent
		}
		title := truncateCols(msg.Title, colMTitle)
		if title == "" {
			title = "-"
		}
		model := truncateCols(msg.ModelID, colMModel)
		if model == "" {
			model = "-"
		}
		in := headText(msg.Text, colMIn)
		if in == "" {
			in = "-"
		}
		line := padCols(timeStr, colMTime) + " " +
			padCols(who, colMWho) + " " +
			padCols(title, colMTitle) + " " +
			padCols(model, colMModel) + " " +
			in
		b.WriteString(line)
		b.WriteString("\n")
	}

	// total (single-session mode only)
	if m.info.ID != "" {
		totalLine := fmt.Sprintf("TOTAL  in %s  out %s  cost %.4f",
			fmtToken(m.info.TokensIn), fmtToken(m.info.TokensOut), m.info.Cost)
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(totalLine))
	} else {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).
			Render(fmt.Sprintf("TOTAL  %d top-level sessions", m.info.TokensIn)))
	}

	// footer
	pos := ""
	if m.offset > 0 {
		pos = fmt.Sprintf("  ^%d older", m.offset)
	}
	who := m.sessionID
	if who == "" {
		who = "ALL SESSIONS"
	}
	footer := fmt.Sprintf("%s  (2s refresh, Ctrl+C quit, ↑/↓ scroll)%s", who, pos)
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

// padCols pads s on the right with spaces to the given display width.
func padCols(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// headText returns the first maxCols display columns of s.
func headText(s string, maxCols int) string {
	if s == "" || maxCols <= 0 {
		return ""
	}
	runes := []rune(s)
	// trim newlines for single-line display
	var clean []rune
	for _, r := range runes {
		if r == '\n' || r == '\r' || r == '\t' {
			clean = append(clean, ' ')
		} else {
			clean = append(clean, r)
		}
	}
	// walk from the start, collecting up to maxCols display columns
	w := 0
	end := 0
	for i := 0; i < len(clean); i++ {
		rw := 1
		if clean[i] > 127 {
			rw = 2
		}
		if w+rw > maxCols {
			break
		}
		w += rw
		end = i + 1
	}
	return string(clean[:end])
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
