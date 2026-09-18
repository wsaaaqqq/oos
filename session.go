package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Session struct {
	ID           string
	Title        string
	Slug         string
	Directory    string
	ModelID      string
	Agent        string
	TimeUpdated  int64
	FirstUserMsg string
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

// LoadSessions loads the most recent `limit` top-level sessions (limit <= 0
// means all), together with each session's first user message.
func LoadSessions(dbFile string, limit int) ([]Session, error) {
	return loadSessions(dbFile, limit, nil)
}

// LoadSessionsExcluding loads every top-level session except excludeIDs.
// Used to fetch the remainder after the initial recent-session batch.
func LoadSessionsExcluding(dbFile string, excludeIDs []string) ([]Session, error) {
	return loadSessions(dbFile, 0, excludeIDs)
}

func loadSessions(dbFile string, limit int, excludeIDs []string) ([]Session, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	var sessions []Session
	if err := loadSessionRows(db, &sessions, limit, excludeIDs); err != nil {
		return nil, err
	}
	if err := loadFirstUserTexts(db, sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func loadSessionRows(db *sql.DB, sessions *[]Session, limit int, excludeIDs []string) error {
	query := `
		SELECT s.id, s.title, s.slug, s.directory, s.model, s.agent, s.time_updated
		FROM session s
		WHERE s.time_archived IS NULL
		  AND (s.parent_id IS NULL OR s.parent_id = '')`
	var args []interface{}
	if len(excludeIDs) > 0 {
		placeholders := make([]string, len(excludeIDs))
		for i, id := range excludeIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query += " AND s.id NOT IN (" + strings.Join(placeholders, ",") + ")"
	}
	query += " ORDER BY s.time_updated DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Session
		var modelJSON sql.NullString
		var agent sql.NullString
		if err := rows.Scan(&s.ID, &s.Title, &s.Slug, &s.Directory, &modelJSON, &agent, &s.TimeUpdated); err != nil {
			return fmt.Errorf("scan session: %w", err)
		}
		s.ModelID = parseModelID(modelJSON.String)
		s.Agent = agent.String
		*sessions = append(*sessions, s)
	}
	return rows.Err()
}

func parseModelID(raw string) string {
	if raw == "" {
		return ""
	}
	var m struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return raw
	}
	if m.ProviderID != "" && !strings.EqualFold(m.ProviderID, m.ID) {
		return m.ProviderID + "/" + m.ID
	}
	return m.ID
}

func loadFirstUserTexts(db *sql.DB, sessions []Session) error {
	if len(sessions) == 0 {
		return nil
	}
	// Restrict the scan to the sessions we actually need: the planner can use
	// message_session_time_created_id_idx instead of scanning every message.
	placeholders := make([]string, len(sessions))
	args := make([]interface{}, len(sessions))
	for i, s := range sessions {
		placeholders[i] = "?"
		args[i] = s.ID
	}
	rows, err := db.Query(`
		SELECT session_id, id FROM message
		WHERE data LIKE '%"role":"user"%'
		  AND session_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY session_id, time_created
	`, args...)
	if err != nil {
		return fmt.Errorf("query user msgs: %w", err)
	}
	defer rows.Close()

	firstMsgPerSession := make(map[string]string)
	for rows.Next() {
		var sid, mid string
		if err := rows.Scan(&sid, &mid); err != nil {
			continue
		}
		if _, ok := firstMsgPerSession[sid]; !ok {
			firstMsgPerSession[sid] = mid
		}
	}
	rows.Close()

	if len(firstMsgPerSession) == 0 {
		return nil
	}

	msgIDs := make([]string, 0, len(firstMsgPerSession))
	msgsToSessions := make(map[string]string)
	for sid, mid := range firstMsgPerSession {
		msgIDs = append(msgIDs, mid)
		msgsToSessions[mid] = sid
	}

	textPerSession := make(map[string]string)
	if err := loadTextsForMessages(db, msgIDs, msgsToSessions, textPerSession); err != nil {
		return err
	}

	for i := range sessions {
		sessions[i].FirstUserMsg = textPerSession[sessions[i].ID]
	}
	return nil
}

func loadTextsForMessages(db *sql.DB, msgIDs []string, msgToSession map[string]string, out map[string]string) error {
	placeholders := make([]string, len(msgIDs))
	args := make([]interface{}, len(msgIDs))
	for i, mid := range msgIDs {
		placeholders[i] = "?"
		args[i] = mid
	}

	query := fmt.Sprintf(`
		SELECT message_id, data FROM part
		WHERE message_id IN (%s)
		  AND data LIKE '{"type":"text"%%'
		ORDER BY message_id, time_created
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("query parts: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var mid, data string
		if err := rows.Scan(&mid, &data); err != nil {
			continue
		}
		if seen[mid] {
			continue
		}
		text := extractPartText(data)
		if text != "" {
			sid := msgToSession[mid]
			if _, ok := out[sid]; !ok {
				out[sid] = text
			}
		}
		seen[mid] = true
	}
	return rows.Err()
}

func extractPartText(raw string) string {
	var p struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return ""
	}
	return p.Text
}

// LoadAllMessages loads every text part, keyed by session ID.
//
// Deliberately one unfiltered sequential scan: scoping by `session_id IN (...)`
// uses index seeks (random I/O) and measured ~34s cold vs ~13s for this
// sequential scan on a 3GB db. Do not "optimize" it into per-session batches.
func LoadAllMessages(dbFile string) (map[string][]string, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// NOTE: prefix LIKE '{"type":"text"%' relies on opencode serializing
	// "type" as the first JSON key. Verified 0 misses on 18k text rows and
	// ~30% faster than substring LIKE on a 3GB db (fail-fast on first bytes).
	// Do NOT extend to '{"type":"text","text"' — 170 rows have other key
	// order after "type" and would be silently dropped.
	rows, err := db.Query(`
		SELECT m.session_id, p.data
		FROM part p
		JOIN message m ON m.id = p.message_id
		WHERE p.data LIKE '{"type":"text"%'
		ORDER BY m.session_id, m.time_created, p.time_created
	`)
	if err != nil {
		return nil, fmt.Errorf("query texts: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var sid, data string
		if err := rows.Scan(&sid, &data); err != nil {
			continue
		}
		text := extractPartText(data)
		if text != "" {
			result[sid] = append(result[sid], text)
		}
	}
	return result, rows.Err()
}

// MonitorSession is the info + message flow for a single session monitor.
type MonitorSession struct {
	ID           string
	Title        string
	Agent        string
	ModelID      string
	TokensIn     int64
	TokensOut    int64
	Cost         float64
	Messages     []MonitorMessage
}

type MonitorMessage struct {
	TimeCreated int64
	Role        string
	Agent       string
	Title       string
	ModelID     string
	Text        string
}

func LoadMonitor(dbFile, sessionID string) (*MonitorSession, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	ms := &MonitorSession{ID: sessionID}

	if sessionID == "" {
		// global mode: all top-level sessions
		err = db.QueryRow(`
			SELECT COUNT(*) FROM session
			WHERE parent_id IS NULL OR parent_id = ''
		`).Scan(&ms.TokensIn)
		if err != nil {
			return nil, fmt.Errorf("query sessions: %w", err)
		}
	} else {
		err = db.QueryRow(`
			SELECT title, agent, model, tokens_input, tokens_output, cost
			FROM session WHERE id = ?
		`, sessionID).Scan(&ms.Title, &ms.Agent, &ms.ModelID, &ms.TokensIn, &ms.TokensOut, &ms.Cost)
		if err != nil {
			return nil, fmt.Errorf("query session: %w", err)
		}
		ms.ModelID = parseModelID(ms.ModelID)
	}

	var rows *sql.Rows
	if sessionID == "" {
		// global mode: newest messages across all top-level sessions
		rows, err = db.Query(`
			SELECT m.id, m.time_created, m.session_id, m.data FROM message m
			WHERE m.session_id IN (
				SELECT id FROM session WHERE parent_id IS NULL OR parent_id = ''
			)
			ORDER BY m.time_created DESC LIMIT 300
		`)
	} else {
		// fetch main session + its subagent sessions (recursively, so
		// nested subagents are included), ordered by time across all
		rows, err = db.Query(`
			WITH RECURSIVE subs(id) AS (
				SELECT id FROM session WHERE id = ?
				UNION ALL
				SELECT s.id FROM session s JOIN subs ON s.parent_id = subs.id
			)
			SELECT m.id, m.time_created, m.session_id, m.data FROM message m
			WHERE m.session_id IN (SELECT id FROM subs)
			ORDER BY m.time_created ASC
		`, sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}

	type msgRow struct {
		id        string
		sessionID string
		time      int64
		role      string
		agent     string
		model     string
	}
	var msgRows []msgRow
	for rows.Next() {
		var id, sid, data string
		var tc int64
		if err := rows.Scan(&id, &tc, &sid, &data); err != nil {
			continue
		}
		var msg struct {
			Role       string `json:"role"`
			Agent      string `json:"agent"`
			ModelID    string `json:"modelID"`
			ProviderID string `json:"providerID"`
			Model      struct {
				ProviderID string `json:"providerID"`
				ModelID    string `json:"modelID"`
			} `json:"model"`
		}
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			continue
		}
		// assistant messages use flat modelID/providerID,
		// user messages use nested model.providerID/model.modelID
		model := msg.ModelID
		provider := msg.ProviderID
		if model == "" && msg.Model.ModelID != "" {
			model = msg.Model.ModelID
			provider = msg.Model.ProviderID
		}
		if provider != "" && !strings.EqualFold(provider, model) {
			model = provider + "/" + model
		}
		msgRows = append(msgRows, msgRow{id: id, sessionID: sid, time: tc, role: msg.Role, agent: msg.Agent, model: model})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if sessionID == "" {
		// global mode queried DESC; reverse to ascending
		for i, j := 0, len(msgRows)-1; i < j; i, j = i+1, j-1 {
			msgRows[i], msgRows[j] = msgRows[j], msgRows[i]
		}
	}

	if len(msgRows) == 0 {
		return ms, nil
	}

	// load session titles for the involved sessions
	titleByID := make(map[string]string)
	{
		sidSet := make(map[string]bool)
		for _, r := range msgRows {
			sidSet[r.sessionID] = true
		}
		placeholders := make([]string, 0, len(sidSet))
		args := make([]interface{}, 0, len(sidSet))
		for sid := range sidSet {
			placeholders = append(placeholders, "?")
			args = append(args, sid)
		}
		trows, err := db.Query(`
			SELECT id, title FROM session WHERE id IN (`+strings.Join(placeholders, ",")+`)
		`, args...)
		if err != nil {
			return nil, fmt.Errorf("query session titles: %w", err)
		}
		for trows.Next() {
			var id, title string
			if err := trows.Scan(&id, &title); err != nil {
				continue
			}
			titleByID[id] = title
		}
		trows.Close()
		if err := trows.Err(); err != nil {
			return nil, err
		}
	}

	// batch load first text part per message
	textByMsg := make(map[string]string)
	{
		placeholders := make([]string, len(msgRows))
		args := make([]interface{}, len(msgRows))
		for i, r := range msgRows {
			placeholders[i] = "?"
			args[i] = r.id
		}
		query := fmt.Sprintf(`
			SELECT message_id, data FROM part
			WHERE message_id IN (%s)
			  AND data LIKE '{"type":"text"%%'
			ORDER BY message_id, time_created
		`, strings.Join(placeholders, ","))
		prows, err := db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("query parts: %w", err)
		}
		seen := make(map[string]bool)
		for prows.Next() {
			var mid, pdata string
			if err := prows.Scan(&mid, &pdata); err != nil {
				continue
			}
			if seen[mid] {
				continue
			}
			text := extractPartText(pdata)
			if text != "" {
				textByMsg[mid] = text
			}
			seen[mid] = true
		}
		prows.Close()
		if err := prows.Err(); err != nil {
			return nil, err
		}
	}

	for _, r := range msgRows {
		ms.Messages = append(ms.Messages, MonitorMessage{
			TimeCreated: r.time,
			Role:        r.role,
			Agent:       r.agent,
			Title:       titleByID[r.sessionID],
			ModelID:     r.model,
			Text:        textByMsg[r.id],
		})
	}
	return ms, nil
}
