package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
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
	textInput     textinput.Model
	questionInput textinput.Model
	sessions      []Session
	allMsgs       map[string][]string
	filtered      []Session
	cursor        int
	scrollOff     int
	sessionsDone  bool
	msgsDone      bool
	ready         bool
	selected      *Session
	confirmDelete bool
	pendingDelID  string
	questionMode  int // 0 closed, 1 question list, 2 action menu
	questionBusy  bool
	questionError string
	questionSess  Session
	questions     []UserQuestion
	questionHits  []UserQuestion
	questionCur   int
	questionOff   int
	questionPick  UserQuestion
	actionCur     int
	dbPath        string
	width         int
	height        int
	err           error
}

// initialSessionLimit is how many recent sessions are loaded before the first
// paint; the rest stream in afterwards (see loadMoreSessionsCmd).
const initialSessionLimit = 100

type sessionsLoadedMsg struct {
	sessions []Session
}

type moreSessionsLoadedMsg struct {
	sessions []Session
}

type msgsLoadedMsg struct {
	msgs map[string][]string
}

type userQuestionsLoadedMsg struct {
	sessionID string
	questions []UserQuestion
	err       error
}

type questionActionDoneMsg struct {
	action   string
	session  Session
	question UserQuestion
	forkedID string
	err      error
}

type dbErrMsg struct {
	err error
}

type sessionDeletedMsg struct {
	id string
}

func (m model) Init() tea.Cmd {
	return loadSessionsCmd(m.dbPath)
}

func loadSessionsCmd(dbPath string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := LoadSessions(dbPath, initialSessionLimit)
		if err != nil {
			return dbErrMsg{err}
		}
		return sessionsLoadedMsg{sessions}
	}
}

func loadMoreSessionsCmd(dbPath string, excludeIDs []string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := LoadSessionsExcluding(dbPath, excludeIDs)
		if err != nil {
			return dbErrMsg{err}
		}
		return moreSessionsLoadedMsg{sessions}
	}
}

func loadMsgsCmd(dbPath string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := LoadAllMessages(dbPath)
		if err != nil {
			return dbErrMsg{err}
		}
		return msgsLoadedMsg{msgs}
	}
}

func loadUserQuestionsCmd(dbPath, sessionID string) tea.Cmd {
	return func() tea.Msg {
		questions, err := LoadUserQuestions(dbPath, sessionID)
		return userQuestionsLoadedMsg{sessionID: sessionID, questions: questions, err: err}
	}
}

func forkQuestionCmd(session Session, question UserQuestion) tea.Cmd {
	return func() tea.Msg {
		forkedID, err := ForkSessionAtMessage(session.Directory, session.ID, question.ID)
		if err == nil {
			err = OpenForkedSessionWithPrompt(Session{ID: forkedID, Directory: session.Directory}, question.Text)
			if err != nil {
				err = fmt.Errorf("fork created (%s), but could not open it: %w", forkedID, err)
			}
		}
		return questionActionDoneMsg{action: "fork", session: session, question: question, forkedID: forkedID, err: err}
	}
}

func deleteQuestionSessionCmd(session Session) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("opencode", "session", "delete", session.ID)
		cmd.Dir = session.Directory
		out, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail != "" {
				err = fmt.Errorf("%w: %s", err, detail)
			}
		}
		return questionActionDoneMsg{action: "delete", session: session, err: err}
	}
}

func sessionIDs(sessions []Session) []string {
	ids := make([]string, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	return ids
}

// mergeSessions appends new sessions (deduped by ID) and keeps the list
// ordered by most-recently-updated, which is the display order.
func mergeSessions(old, add []Session) []Session {
	seen := make(map[string]bool, len(old))
	for _, s := range old {
		seen[s.ID] = true
	}
	out := make([]Session, 0, len(old)+len(add))
	out = append(out, old...)
	for _, s := range add {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].TimeUpdated > out[j].TimeUpdated
	})
	return out
}

func deleteSessionCmd(id string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("opencode", "session", "delete", id)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		return sessionDeletedMsg{id: id}
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
		textInput: ti,
		cursor:    0,
		scrollOff: 0,
		dbPath:    dbPath,
		width:     80,
		height:    24,
	}
}

func (m model) msgMap() map[string][]string {
	return m.allMsgs
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 20
		if m.questionMode != 0 {
			m.questionInput.Width = max(12, int(float64(msg.Width)*0.9)-14)
		}
		return m, nil

	case sessionsLoadedMsg:
		// stage 1: recent sessions are shown immediately, the rest stream in
		m.sessions = msg.sessions
		m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
		m.cursor = clampCursor(0, len(m.filtered))
		m.scrollOff = 0
		m.ready = true
		m.sessionsDone = false
		cmds := []tea.Cmd{loadMoreSessionsCmd(m.dbPath, sessionIDs(msg.sessions))}
		if !m.msgsDone {
			cmds = append(cmds, loadMsgsCmd(m.dbPath))
		}
		return m, tea.Batch(cmds...)

	case moreSessionsLoadedMsg:
		m.sessions = mergeSessions(m.sessions, msg.sessions)
		m.sessionsDone = true
		m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
		m.cursor = clampCursor(m.cursor, len(m.filtered))
		m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
		return m, nil

	case msgsLoadedMsg:
		m.allMsgs = msg.msgs
		m.msgsDone = true
		m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
		m.cursor = clampCursor(m.cursor, len(m.filtered))
		m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
		return m, nil

	case userQuestionsLoadedMsg:
		if m.questionMode == 0 || m.questionSess.ID != msg.sessionID {
			return m, nil
		}
		m.questionBusy = false
		if msg.err != nil {
			m.questionError = msg.err.Error()
			return m, nil
		}
		m.questions = msg.questions
		m.questionHits = filterUserQuestions(m.questions, m.questionInput.Value())
		m.questionCur = clampCursor(0, len(m.questionHits))
		m.questionOff = 0
		return m, nil

	case questionActionDoneMsg:
		m.questionBusy = false
		if msg.err != nil {
			m.questionError = msg.err.Error()
			return m, nil
		}
		if msg.action == "delete" {
			for i, s := range m.sessions {
				if s.ID == msg.session.ID {
					m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
					break
				}
			}
			delete(m.allMsgs, msg.session.ID)
			m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
			m.cursor = clampCursor(m.cursor, len(m.filtered))
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			m.questionMode = 0
			return m, nil
		}
		// Fork action already opened the new session in a separate tab.
		if msg.action == "fork" {
			m.questionMode = 0
		}
		return m, nil

	case dbErrMsg:
		m.err = msg.err
		m.ready = true
		m.sessionsDone = true
		m.msgsDone = true
		return m, nil

	case sessionDeletedMsg:
		for i, s := range m.sessions {
			if s.ID == msg.id {
				m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
				break
			}
		}
		delete(m.allMsgs, msg.id)
		m.filtered = FilterSessions(m.sessions, ParseKeys(m.textInput.Value()), m.msgMap())
		m.cursor = clampCursor(m.cursor, len(m.filtered))
		m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
		return m, nil

	case tea.KeyMsg:
		if m.questionMode != 0 {
			return m.updateQuestionPicker(msg)
		}
		if msg.Alt && len(msg.Runes) > 0 && (msg.Runes[0] == 's' || msg.Runes[0] == 'S') {
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
				s := m.filtered[m.cursor]
				m.questionSess = s
				m.questionMode = 1
				m.questionBusy = true
				m.questionError = ""
				m.questions = nil
				m.questionHits = nil
				m.questionCur = -1
				m.questionOff = 0
				m.questionPick = UserQuestion{}
				m.questionInput = textinput.New()
				m.questionInput.Placeholder = "filter user questions..."
				m.questionInput.Prompt = ""
				m.questionInput.CharLimit = 200
				m.questionInput.Focus()
				m.questionInput.Width = max(12, int(float64(m.width)*0.9)-14)
				return m, loadUserQuestionsCmd(m.dbPath, s.ID)
			}
			return m, nil
		}
		if msg.Alt && len(msg.Runes) > 0 && (msg.Runes[0] == 'q' || msg.Runes[0] == 'Q') {
			m.confirmDelete = false
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
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
		if msg.Alt && len(msg.Runes) > 0 && (msg.Runes[0] == 'm' || msg.Runes[0] == 'M') {
			m.confirmDelete = false
			id := ""
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
				id = m.filtered[m.cursor].ID
			}
			if err := spawnMonitor(id); err != nil {
				m.err = fmt.Errorf("spawn monitor: %w", err)
			}
			return m, nil
		}
		if msg.Type == tea.KeyCtrlD {
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
				id := m.filtered[m.cursor].ID
				if m.confirmDelete && m.pendingDelID == id {
					m.confirmDelete = false
					m.pendingDelID = ""
					return m, deleteSessionCmd(id)
				}
				m.confirmDelete = true
				m.pendingDelID = id
			}
			return m, nil
		}
		m.confirmDelete = false
		m.pendingDelID = ""
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
				s := m.filtered[m.cursor]
				err := openSessionBg(s)
				if err != nil {
					fmt.Fprintf(os.Stderr, "oos: openSessionBg error: %v\n", err)
				}
			}
			return m, nil

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			} else if m.cursor == 0 {
				m.cursor = -1 // cancel selection
			}
			m.scrollOff = calcScrollOff(m.scrollOff, m.cursor, m.visibleSlots())
			return m, nil

		case tea.KeyDown:
			if m.cursor < 0 {
				m.cursor = 0
			} else if m.cursor < len(m.filtered)-1 {
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

	if m.ready {
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
	if cursor < 0 {
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

func filterUserQuestions(questions []UserQuestion, query string) []UserQuestion {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return questions
	}
	filtered := make([]UserQuestion, 0, len(questions))
	for _, question := range questions {
		if strings.Contains(strings.ToLower(question.Text), query) {
			filtered = append(filtered, question)
		}
	}
	return filtered
}

func (m model) questionVisibleRows() int {
	rows := int(float64(m.height)*0.9) - 8
	if rows < 1 {
		return 1
	}
	return rows
}

func (m model) updateQuestionPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		if m.questionMode == 2 {
			m.questionMode = 1
			m.questionError = ""
			return m, nil
		}
		m.questionMode = 0
		m.questionError = ""
		m.questionBusy = false
		return m, nil
	}
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	if m.questionBusy {
		return m, nil
	}
	if msg.Type == tea.KeyCtrlD {
		m.questionBusy = true
		m.questionError = "Deleting this opencode session..."
		return m, deleteQuestionSessionCmd(m.questionSess)
	}

	if m.questionMode == 2 {
		switch msg.Type {
		case tea.KeyUp:
			if m.actionCur > 0 {
				m.actionCur--
			}
		case tea.KeyDown:
			if m.actionCur < 1 {
				m.actionCur++
			}
		case tea.KeyHome:
			m.actionCur = 0
		case tea.KeyEnd:
			m.actionCur = 1
		case tea.KeyEnter:
			switch m.actionCur {
			case 0: // Fork
				m.questionBusy = true
				m.questionError = "Starting OpenCode to fork..."
				return m, forkQuestionCmd(m.questionSess, m.questionPick)
			case 1: // Copy selected question text.
				if err := clipboard.WriteAll(m.questionPick.Text); err != nil {
					m.questionError = fmt.Sprintf("Clipboard: %v", err)
				} else {
					m.questionMode = 1
					m.questionError = "Question copied to clipboard"
				}
			}
		}
		return m, nil
	}

	visible := m.questionVisibleRows()
	switch msg.Type {
	case tea.KeyUp:
		if m.questionCur > 0 {
			m.questionCur--
		}
		m.questionOff = calcScrollOff(m.questionOff, m.questionCur, visible)
		return m, nil
	case tea.KeyDown:
		if m.questionCur < len(m.questionHits)-1 {
			m.questionCur++
		}
		m.questionOff = calcScrollOff(m.questionOff, m.questionCur, visible)
		return m, nil
	case tea.KeyPgUp:
		m.questionCur = max(0, m.questionCur-visible)
		m.questionOff = calcScrollOff(m.questionOff, m.questionCur, visible)
		return m, nil
	case tea.KeyPgDown:
		m.questionCur = min(len(m.questionHits)-1, m.questionCur+visible)
		m.questionOff = calcScrollOff(m.questionOff, m.questionCur, visible)
		return m, nil
	case tea.KeyHome:
		m.questionCur = clampCursor(0, len(m.questionHits))
		m.questionOff = 0
		return m, nil
	case tea.KeyEnd:
		m.questionCur = clampCursor(len(m.questionHits)-1, len(m.questionHits))
		m.questionOff = calcScrollOff(m.questionOff, m.questionCur, visible)
		return m, nil
	case tea.KeyEnter:
		if m.questionCur >= 0 && m.questionCur < len(m.questionHits) {
			m.questionPick = m.questionHits[m.questionCur]
			m.actionCur = 0
			m.questionMode = 2
			m.questionError = ""
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.questionInput, cmd = m.questionInput.Update(msg)
	m.questionHits = filterUserQuestions(m.questions, m.questionInput.Value())
	m.questionCur = clampCursor(0, len(m.questionHits))
	m.questionOff = 0
	return m, cmd
}

func (m model) renderQuestionPicker() string {
	panelW := int(float64(m.width) * 0.9)
	panelH := int(float64(m.height) * 0.9)
	if panelW > m.width {
		panelW = m.width
	}
	if panelH > m.height {
		panelH = m.height
	}
	if panelW < 30 {
		panelW = min(30, m.width)
	}
	if panelH < 8 {
		panelH = min(8, m.height)
	}
	innerW := max(10, panelW-4)

	var lines []string
	if m.questionMode == 2 {
		lines = append(lines, truncateCols("Actions — "+m.questionSess.Title, innerW))
		lines = append(lines, truncateCols("Question: "+oneLine(m.questionPick.Text), innerW))
		lines = append(lines, "")
		actions := []string{"Fork from this question", "Copy this question"}
		for i, action := range actions {
			prefix := "  "
			if i == m.actionCur {
				prefix = "> "
			}
			line := lipgloss.NewStyle().Width(innerW).Render(truncateCols(prefix+action, innerW))
			if i == m.actionCur {
				line = lipgloss.NewStyle().Width(innerW).Background(lipgloss.Color("12")).Foreground(lipgloss.Color("0")).Bold(true).Render(truncateCols(prefix+action, innerW))
			}
			lines = append(lines, line)
		}
	} else {
		title := fmt.Sprintf("User questions — %s (%d)", m.questionSess.Title, len(m.questions))
		lines = append(lines, truncateCols(title, innerW))
		if m.questionBusy {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("Loading user questions..."))
		} else {
			m.questionInput.Width = max(12, innerW-2)
			lines = append(lines, m.questionInput.View())
			seqW, timeW := 6, 16
			questionW := max(1, innerW-seqW-timeW-6)
			head := padCols("#", seqW) + " │ " + padCols("USER QUESTION", questionW) + " │ " + padCols("TIME", timeW)
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(head))
			lines = append(lines, strings.Repeat("-", innerW))
			start := max(0, m.questionOff)
			end := min(len(m.questionHits), start+m.questionVisibleRows())
			for i := start; i < end; i++ {
				q := m.questionHits[i]
				seq := questionSequence(m.questions, q.ID)
				text := headQuestion(q.Text)
				timeText := time.UnixMilli(q.TimeCreated).Format("2006-01-02 15:04")
				line := padCols(fmt.Sprintf("%d", seq), seqW) + " │ " + padCols(truncateCols(text, questionW), questionW) + " │ " + padCols(timeText, timeW)
				if i == m.questionCur {
					line = lipgloss.NewStyle().Width(innerW).Background(lipgloss.Color("12")).Foreground(lipgloss.Color("0")).Bold(true).Render(line)
				}
				lines = append(lines, line)
			}
			if len(m.questionHits) == 0 && !m.questionBusy {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("No matching questions"))
			}
		}
	}

	if m.questionError != "" {
		color := "9"
		if strings.HasPrefix(m.questionError, "Copied") {
			color = "10"
		} else if m.questionBusy {
			color = "11"
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(truncateCols(m.questionError, innerW)))
	}
	if m.questionMode == 2 {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("↑/↓ select  Enter choose  Ctrl+D delete session  Esc back"))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Type to filter  ↑/↓ select  Enter actions  Ctrl+D delete session  Esc back"))
	}

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(panelW - 2).
		Height(panelH - 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func questionSequence(questions []UserQuestion, id string) int {
	for i, question := range questions {
		if question.ID == id {
			return i + 1
		}
	}
	return 0
}

func oneLine(text string) string {
	return strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(text)
}

func headQuestion(text string) string {
	return oneLine(text)
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	if m.questionMode != 0 {
		return m.renderQuestionPicker()
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
	tag := "all session loaded"
	if !m.sessionsDone || !m.msgsDone {
		tag = "last 100 session loaded"
	}

	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(m.width)

	rightTag := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(" " + tag)

	// reserve room for the longest tag ("last 100 session loaded", 23 cols)
	inputWidth := m.width - 26
	if inputWidth < 20 {
		inputWidth = 20
	}
	m.textInput.Width = inputWidth

	inner := lipgloss.JoinHorizontal(lipgloss.Top, m.textInput.View(), rightTag)
	return searchStyle.Render(inner)
}

func (m model) renderResults() string {
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
	if m.confirmDelete {
		title := ""
		for _, s := range m.filtered {
			if s.ID == m.pendingDelID {
				title = s.Title
				break
			}
		}
		if len(title) > 30 {
			title = title[:30]
		}
		msg := fmt.Sprintf("Delete \"%s\"? Press Ctrl+D again to confirm, any other key to cancel", title)
		return lipgloss.NewStyle().
			Width(m.width).
			Foreground(lipgloss.Color("9")).
			Render(truncateCols(msg, m.width))
	}

	count := fmt.Sprintf("%d matches", len(m.filtered))
	keys := "Alt+Q copy  Alt+M monitor  Ctrl+D delete  esc quit"

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
	if length <= 0 {
		return -1
	}
	if cursor < 0 {
		return -1 // keep "no selection" state
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
	_, err := p.Run()
	return nil, err
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
