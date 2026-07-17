package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	colTitle = 24
	colMsg   = 56
	colTime  = 11
	colSep   = 3
	colTotal = 2 + colTitle + colSep + colMsg + colSep + colTime
)

type model struct {
	textInput   textinput.Model
	sessions    []Session
	allMsgs     map[string][]string
	filtered    []Session
	cursor      int
	scrollOff   int
	searchMsgs  bool
	loadingMsgs bool
	ready       bool
	selected    *Session
	dbPath      string
	width       int
	height      int
	err         error
}

type sessionsLoadedMsg struct {
	sessions []Session
}

type msgsLoadedMsg struct {
	msgs map[string][]string
}

type dbErrMsg struct {
	err error
}

func (m model) Init() tea.Cmd {
	return loadSessionsCmd(m.dbPath)
}

func loadSessionsCmd(dbPath string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := LoadSessions(dbPath)
		if err != nil {
			return dbErrMsg{err}
		}
		return sessionsLoadedMsg{sessions}
	}
}

func loadAllMsgsCmd(dbPath string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := LoadAllMessages(dbPath)
		if err != nil {
			return dbErrMsg{err}
		}
		return msgsLoadedMsg{msgs}
	}
}

func initialModel(dbPath string, initialQuery string) model {
	ti := textinput.New()
	ti.Placeholder = "type keywords, ! to exclude..."
	ti.Focus()
	ti.Prompt = ""
	ti.SetValue(initialQuery)
	ti.CharLimit = 200
	ti.Width = 60

	return model{
		textInput:  ti,
		cursor:     0,
		scrollOff:  0,
		searchMsgs: true,
		dbPath:     dbPath,
		width:      80,
		height:     24,
	}
}

func (m model) msgMap() map[string][]string {
	if m.searchMsgs {
		return m.allMsgs
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 20
		return m, nil

	case sessionsLoadedMsg:
		m.sessions = msg.sessions
		m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
		if len(m.filtered) > 0 {
			m.cursor = 0
			m.scrollOff = 0
		}
		m.ready = true
		if m.searchMsgs && m.allMsgs == nil {
			m.loadingMsgs = true
			return m, loadAllMsgsCmd(m.dbPath)
		}
		return m, nil

	case msgsLoadedMsg:
		m.allMsgs = msg.msgs
		m.loadingMsgs = false
		m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
		m.cursor = clampCursor(m.cursor, len(m.filtered))
		m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
		return m, nil

	case dbErrMsg:
		m.err = msg.err
		m.ready = true
		m.loadingMsgs = false
		return m, nil

	case tea.KeyMsg:
		if msg.Alt && len(msg.Runes) > 0 && (msg.Runes[0] == 's' || msg.Runes[0] == 'S') {
			if !m.loadingMsgs {
				m.searchMsgs = !m.searchMsgs
				if m.searchMsgs && m.allMsgs == nil {
					m.loadingMsgs = true
					return m, loadAllMsgsCmd(m.dbPath)
				}
				m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
				m.cursor = clampCursor(m.cursor, len(m.filtered))
				m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			}
			return m, nil
		}
		if msg.Alt && len(msg.Runes) > 0 && (msg.Runes[0] == 'q' || msg.Runes[0] == 'Q') {
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				dir := m.filtered[m.cursor].Directory
				if len(dir) >= 2 && dir[1] == ':' {
					dir = strings.ReplaceAll(dir, "/", "\\")
				}
				if err := clipboard.WriteAll(dir); err != nil {
					m.err = fmt.Errorf("clipboard: %w", err)
				}
			}
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = &m.filtered[m.cursor]
				return m, tea.Quit
			}
			return m, nil

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			return m, nil

		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			return m, nil

		case tea.KeyPgUp:
			page := m.visibleSlots()
			m.cursor = max(0, m.cursor-page)
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			return m, nil

		case tea.KeyPgDown:
			page := m.visibleSlots()
			m.cursor = min(len(m.filtered)-1, m.cursor+page)
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			return m, nil

		case tea.KeyHome:
			m.cursor = 0
			m.scrollOff = 0
			return m, nil

		case tea.KeyEnd:
			m.cursor = max(0, len(m.filtered)-1)
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			return m, nil

		case tea.KeyCtrlW:
			val := m.textInput.Value()
			idx := strings.LastIndexAny(strings.TrimRight(val, " "), " ")
			if idx >= 0 {
				m.textInput.SetValue(strings.TrimRight(val[:idx], " "))
			} else {
				m.textInput.SetValue("")
			}
			m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
			m.cursor = clampCursor(m.cursor, len(m.filtered))
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			return m, nil
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	if m.ready && !m.loadingMsgs {
		m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
		m.cursor = clampCursor(m.cursor, len(m.filtered))
		m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
	}

	return m, tea.Batch(cmds...)
}

func (m model) visibleSlots() int {
	h := m.height
	if h < 6 {
		h = 6
	}
	slots := h - 5
	if slots < 1 {
		return 1
	}
	return slots
}

func calcScrollOff(curOff, cursor, visible int) int {
	if visible <= 0 {
		return 0
	}
	if cursor < curOff {
		return cursor
	}
	if cursor >= curOff+visible {
		return cursor - visible + 1
	}
	return curOff
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	if !m.ready {
		return "Loading sessions..."
	}

	searchBar := m.renderSearchBar()
	resultsArea := m.renderResults()
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, searchBar, resultsArea, statusBar)
}

func (m model) renderSearchBar() string {
	msgsTag := "msgs OFF"
	if m.searchMsgs {
		msgsTag = "msgs ON"
	}
	if m.loadingMsgs {
		msgsTag = "msgs ..."
	}

	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(m.width)

	rightTag := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(" " + msgsTag)

	inputWidth := m.width - 14
	if inputWidth < 20 {
		inputWidth = 20
	}
	m.textInput.Width = inputWidth

	inner := lipgloss.JoinHorizontal(lipgloss.Top, m.textInput.View(), rightTag)
	return searchStyle.Render(inner)
}

func (m model) renderResults() string {
	if m.loadingMsgs {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Padding(0, 2).
			Render("Loading messages...")
	}

	if len(m.filtered) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Padding(0, 2).
			Render("No matching sessions")
	}

	visible := m.visibleSlots()
	start := m.scrollOff
	end := start + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	filter := ParseKeys(m.textInput.Value())
	var lines []string
	for i := start; i < end; i++ {
		isSelected := i == m.cursor
		lines = append(lines, m.renderItemRow(m.filtered[i], isSelected, filter.Include))
	}

	boxH := m.height - 5
	if boxH < 1 {
		boxH = 1
	}
	box := lipgloss.NewStyle().Height(boxH)
	return box.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func formatDir(dir string, maxCols int) string {
	if dir == "" {
		return "-"
	}

	dir = strings.ReplaceAll(dir, "\\", "/")
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		return dir
	}

	parts := strings.Split(dir, "/")
	if len(parts) == 0 {
		return dir
	}

	fullPath := strings.Join(parts, "/")
	if displayWidth(fullPath) <= maxCols {
		return fullPath
	}

	avail := maxCols - 1
	leaf := parts[len(parts)-1]
	if displayWidth(leaf) > maxCols {
		return "!" + truncateCols(leaf, maxCols-1)
	}

	result := leaf
	for i := len(parts) - 2; i >= 0; i-- {
		parent := parts[i]
		full := parent + "/" + result
		if displayWidth(full) <= avail {
			result = full
			continue
		}
		abbrev := string([]rune(parent)[0]) + "/" + result
		if displayWidth(abbrev) <= avail {
			result = abbrev
			continue
		}
		break
	}

	return "!" + result
}

func (m model) renderItemRow(s Session, selected bool, keywords []string) string {
	cursor := "  "
	if selected {
		cursor = "> "
	}

	titleText := formatDir(s.Directory, colTitle)
	titleText = highlightMatches(titleText, keywords, selected)

	msgText := buildMsgColumn(s.FirstUserMsg, m.allMsgs[s.ID], keywords)

	timeText := formatTime(s.TimeUpdated)

	selBg := lipgloss.Color("12")
	selFg := lipgloss.Color("0")
	normFg := lipgloss.Color("243")

	titleStyle := lipgloss.NewStyle().Width(colTitle + 2)
	sepStyle := lipgloss.NewStyle()
	msgStyle := lipgloss.NewStyle().Width(colMsg)
	timeStyle := lipgloss.NewStyle().Width(colTime).Align(lipgloss.Right)

	if selected {
		titleStyle = titleStyle.Background(selBg).Foreground(selFg).Bold(true)
		sepStyle = sepStyle.Background(selBg).Foreground(selFg)
		msgStyle = msgStyle.Background(selBg).Foreground(selFg)
		timeStyle = timeStyle.Background(selBg).Foreground(selFg)
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("255"))
		sepStyle = sepStyle.Foreground(lipgloss.Color("240"))
		msgStyle = msgStyle.Foreground(normFg)
		timeStyle = timeStyle.Foreground(normFg)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		titleStyle.Render(cursor+titleText),
		sepStyle.Render(" │ "),
		msgStyle.Render(msgText),
		sepStyle.Render(" │ "),
		timeStyle.Render(timeText),
	)
}

func buildMsgColumn(firstMsg string, allMsgs []string, keywords []string) string {
	text := findBestMsg(firstMsg, allMsgs, keywords)
	if text == "" {
		return ""
	}
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	snippet := ctxSnippet(text, keywords, colMsg)
	return highlightMatches(snippet, keywords, false)
}

func findBestMsg(firstMsg string, allMsgs []string, keywords []string) string {
	if len(keywords) == 0 {
		return firstMsg
	}
	if keywordInText(firstMsg, keywords) {
		return firstMsg
	}
	for _, m := range allMsgs {
		if keywordInText(m, keywords) {
			return m
		}
	}
	return firstMsg
}

func keywordInText(text string, keywords []string) bool {
	lower := strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func ctxSnippet(text string, keywords []string, maxCols int) string {
	if len(keywords) == 0 {
		return truncateCols(text, maxCols)
	}

	lower := strings.ToLower(text)
	firstPos := -1
	for _, kw := range keywords {
		pos := strings.Index(lower, strings.ToLower(kw))
		if pos >= 0 && (firstPos < 0 || pos < firstPos) {
			firstPos = pos
		}
	}
	if firstPos < 0 {
		return truncateCols(text, maxCols)
	}

	matchCol := displayWidth(text[:firstPos])
	beforeCols := maxCols / 5

	runes := []rune(text)

	if matchCol > beforeCols {
		wantStartCol := matchCol - beforeCols
		ri, col := 0, 0
		for ri < len(runes) && col < wantStartCol {
			if runes[ri] > 127 {
				col += 2
			} else {
				col++
			}
			ri++
		}
		runes = runes[ri:]
	}

	return truncateCols(strings.TrimSpace(string(runes)), maxCols)
}

func (m model) renderStatusBar() string {
	count := fmt.Sprintf("%d matches", len(m.filtered))
	keys := "Alt+Q copy dir  esc quit"

	countWidth := displayWidth(count) + 2
	avail := m.width - countWidth
	if avail < 10 {
		avail = 10
	}

	keyText := truncateCols(keys, avail)
	pad := avail - displayWidth(keyText)

	bar := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("240"))

	return bar.Render(keyText + strings.Repeat(" ", pad) + count)
}

func highlightMatches(text string, keywords []string, selected bool) string {
	if len(keywords) == 0 || text == "" {
		return text
	}

	lower := strings.ToLower(text)
	var matches []matchRange
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if kwLower == "" {
			continue
		}
		offset := 0
		for {
			pos := strings.Index(lower[offset:], kwLower)
			if pos < 0 {
				break
			}
			absPos := offset + pos
			matches = append(matches, matchRange{absPos, absPos + len(kw)})
			offset = absPos + len(kw)
		}
	}

	if len(matches) == 0 {
		return text
	}

	matches = mergeAndSortMatches(matches)

	hlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	if selected {
		hlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("227"))
	}

	var buf strings.Builder
	lastEnd := 0
	for _, r := range matches {
		if r.start < lastEnd {
			continue
		}
		buf.WriteString(text[lastEnd:r.start])
		buf.WriteString(hlStyle.Render(text[r.start:r.end]))
		lastEnd = r.end
	}
	buf.WriteString(text[lastEnd:])
	return buf.String()
}

type matchRange struct {
	start, end int
}

func mergeAndSortMatches(matches []matchRange) []matchRange {
	if len(matches) <= 1 {
		return matches
	}
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].start < matches[i].start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
	merged := []matchRange{matches[0]}
	for i := 1; i < len(matches); i++ {
		last := &merged[len(merged)-1]
		if matches[i].start <= last.end {
			if matches[i].end > last.end {
				last.end = matches[i].end
			}
		} else {
			merged = append(merged, matches[i])
		}
	}
	return merged
}

func formatTime(ms int64) string {
	t := time.UnixMilli(ms)
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	return t.Format("01-02 15:04")
}

func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if r > 127 {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}

func truncateCols(s string, maxCols int) string {
	if maxCols <= 0 {
		return ""
	}
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		rw := 1
		if r > 127 {
			rw = 2
		}
		if w+rw > maxCols {
			return string(runes[:i])
		}
		w += rw
	}
	return s
}

func clampCursor(cursor, length int) int {
	if length == 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

func runTUI(dbPath string, initialQuery string) (*Session, error) {
	m := initialModel(dbPath, initialQuery)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	fm := final.(model)
	if fm.selected != nil {
		return fm.selected, nil
	}
	return nil, nil
}

func openSession(s Session) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}

	cmd := exec.Command(bin, "-s", s.ID)
	cmd.Dir = s.Directory
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
