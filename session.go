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

func LoadSessions(dbFile string) ([]Session, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	var sessions []Session
	if err := loadSessionRows(db, &sessions); err != nil {
		return nil, err
	}
	if err := loadFirstUserTexts(db, sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func loadSessionRows(db *sql.DB, sessions *[]Session) error {
	rows, err := db.Query(`
		SELECT s.id, s.title, s.slug, s.directory, s.model, s.agent, s.time_updated
		FROM session s
		WHERE s.time_archived IS NULL
		  AND (s.parent_id IS NULL OR s.parent_id = '')
		ORDER BY s.time_updated DESC
	`)
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
	rows, err := db.Query(`
		SELECT session_id, id FROM message
		WHERE data LIKE '%"role":"user"%'
		ORDER BY session_id, time_created
	`)
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
		  AND data LIKE '%%"type":"text"%%'
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

func LoadAllMessages(dbFile string) (map[string][]string, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT m.session_id, p.data
		FROM part p
		JOIN message m ON m.id = p.message_id
		WHERE p.data LIKE '%"type":"text"%'
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
