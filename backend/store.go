package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database and exposes retro-board operations.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database at path and runs migrations.
func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite handles concurrency best with a single writer connection.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS boards (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS columns (
	id       TEXT PRIMARY KEY,
	board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
	title    TEXT NOT NULL,
	position INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS cards (
	id         TEXT PRIMARY KEY,
	column_id  TEXT NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
	text       TEXT NOT NULL,
	author     TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS votes (
	card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
	voter_id TEXT NOT NULL,
	PRIMARY KEY (card_id, voter_id)
);

CREATE INDEX IF NOT EXISTS idx_columns_board ON columns(board_id);
CREATE INDEX IF NOT EXISTS idx_cards_column ON cards(column_id);
CREATE INDEX IF NOT EXISTS idx_votes_card ON votes(card_id);
`
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

var defaultColumns = []string{"What went well", "What didn't go well", "Action items"}

// CreateBoard inserts a new board seeded with the default retro columns.
func (s *Store) CreateBoard(name string) (*Board, error) {
	if name == "" {
		name = "Untitled Retro"
	}
	board := &Board{
		ID:        uuid.NewString(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"INSERT INTO boards (id, name, created_at) VALUES (?, ?, ?)",
		board.ID, board.Name, board.CreatedAt,
	); err != nil {
		return nil, err
	}

	for i, title := range defaultColumns {
		if _, err := tx.Exec(
			"INSERT INTO columns (id, board_id, title, position) VALUES (?, ?, ?, ?)",
			uuid.NewString(), board.ID, title, i,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetBoard(board.ID)
}

// GetBoard loads a board with its columns, cards, and vote tallies.
func (s *Store) GetBoard(id string) (*Board, error) {
	board := &Board{}
	err := s.db.QueryRow(
		"SELECT id, name, created_at FROM boards WHERE id = ?", id,
	).Scan(&board.ID, &board.Name, &board.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	colRows, err := s.db.Query(
		"SELECT id, board_id, title, position FROM columns WHERE board_id = ? ORDER BY position",
		id,
	)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	columnIndex := map[string]int{}
	for colRows.Next() {
		var c Column
		if err := colRows.Scan(&c.ID, &c.BoardID, &c.Title, &c.Position); err != nil {
			return nil, err
		}
		c.Cards = []Card{}
		columnIndex[c.ID] = len(board.Columns)
		board.Columns = append(board.Columns, c)
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	cardRows, err := s.db.Query(`
SELECT c.id, c.column_id, c.text, c.author, c.created_at
FROM cards c
JOIN columns col ON col.id = c.column_id
WHERE col.board_id = ?
ORDER BY c.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer cardRows.Close()

	cardIndex := map[string]*Card{}
	for cardRows.Next() {
		var card Card
		if err := cardRows.Scan(&card.ID, &card.ColumnID, &card.Text, &card.Author, &card.CreatedAt); err != nil {
			return nil, err
		}
		card.Voters = []string{}
		ci, ok := columnIndex[card.ColumnID]
		if !ok {
			continue
		}
		col := &board.Columns[ci]
		col.Cards = append(col.Cards, card)
		cardIndex[card.ID] = &col.Cards[len(col.Cards)-1]
	}
	if err := cardRows.Err(); err != nil {
		return nil, err
	}

	voteRows, err := s.db.Query(`
SELECT v.card_id, v.voter_id
FROM votes v
JOIN cards c ON c.id = v.card_id
JOIN columns col ON col.id = c.column_id
WHERE col.board_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer voteRows.Close()

	for voteRows.Next() {
		var cardID, voterID string
		if err := voteRows.Scan(&cardID, &voterID); err != nil {
			return nil, err
		}
		if card, ok := cardIndex[cardID]; ok {
			card.Votes++
			card.Voters = append(card.Voters, voterID)
		}
	}
	if err := voteRows.Err(); err != nil {
		return nil, err
	}

	return board, nil
}

// boardIDForColumn returns the board a column belongs to.
func (s *Store) boardIDForColumn(columnID string) (string, error) {
	var boardID string
	err := s.db.QueryRow("SELECT board_id FROM columns WHERE id = ?", columnID).Scan(&boardID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return boardID, err
}

// boardIDForCard returns the board a card belongs to.
func (s *Store) boardIDForCard(cardID string) (string, error) {
	var boardID string
	err := s.db.QueryRow(`
SELECT col.board_id
FROM cards c
JOIN columns col ON col.id = c.column_id
WHERE c.id = ?`, cardID).Scan(&boardID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return boardID, err
}

// CreateCard adds a new card to a column and returns the owning board id.
func (s *Store) CreateCard(columnID, text, author string) (string, error) {
	boardID, err := s.boardIDForColumn(columnID)
	if err != nil || boardID == "" {
		return "", err
	}
	if author == "" {
		author = "Anonymous"
	}
	_, err = s.db.Exec(
		"INSERT INTO cards (id, column_id, text, author, created_at) VALUES (?, ?, ?, ?, ?)",
		uuid.NewString(), columnID, text, author, time.Now().UTC(),
	)
	return boardID, err
}

// UpdateCard edits a card's text and returns the owning board id.
func (s *Store) UpdateCard(cardID, text string) (string, error) {
	boardID, err := s.boardIDForCard(cardID)
	if err != nil || boardID == "" {
		return "", err
	}
	_, err = s.db.Exec("UPDATE cards SET text = ? WHERE id = ?", text, cardID)
	return boardID, err
}

// DeleteCard removes a card and returns the owning board id.
func (s *Store) DeleteCard(cardID string) (string, error) {
	boardID, err := s.boardIDForCard(cardID)
	if err != nil || boardID == "" {
		return "", err
	}
	_, err = s.db.Exec("DELETE FROM cards WHERE id = ?", cardID)
	return boardID, err
}

// ToggleVote adds or removes a participant's vote on a card.
func (s *Store) ToggleVote(cardID, voterID string) (string, error) {
	boardID, err := s.boardIDForCard(cardID)
	if err != nil || boardID == "" {
		return "", err
	}

	var exists int
	err = s.db.QueryRow(
		"SELECT COUNT(1) FROM votes WHERE card_id = ? AND voter_id = ?",
		cardID, voterID,
	).Scan(&exists)
	if err != nil {
		return "", err
	}

	if exists > 0 {
		_, err = s.db.Exec("DELETE FROM votes WHERE card_id = ? AND voter_id = ?", cardID, voterID)
	} else {
		_, err = s.db.Exec("INSERT INTO votes (card_id, voter_id) VALUES (?, ?)", cardID, voterID)
	}
	return boardID, err
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}
