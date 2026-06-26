package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// ErrEmailTaken is returned when registering an email that already exists.
var ErrEmailTaken = errors.New("email already registered")

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
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	name          TEXT NOT NULL,
	created_at    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS boards (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	owner_id   TEXT,
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
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Backfill columns for databases created before these fields existed.
	// This must happen before any index that references the new column.
	if err := s.addColumnIfMissing("boards", "owner_id", "TEXT"); err != nil {
		return fmt.Errorf("migrate boards.owner_id: %w", err)
	}

	if _, err := s.db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_boards_owner ON boards(owner_id)",
	); err != nil {
		return fmt.Errorf("migrate idx_boards_owner: %w", err)
	}
	return nil
}

// addColumnIfMissing adds a column to a table only if it isn't already present,
// making schema changes safe to run against existing databases.
func (s *Store) addColumnIfMissing(table, column, definition string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			dfltValue  sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return nil // already exists
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

var defaultColumns = []string{"What went well", "What didn't go well", "Action items"}

// CreateBoard inserts a new board (owned by ownerID) seeded with the default
// retro columns.
func (s *Store) CreateBoard(name, ownerID string) (*Board, error) {
	if name == "" {
		name = "Untitled Retro"
	}
	board := &Board{
		ID:        uuid.NewString(),
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: time.Now().UTC(),
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"INSERT INTO boards (id, name, owner_id, created_at) VALUES (?, ?, ?, ?)",
		board.ID, board.Name, board.OwnerID, board.CreatedAt,
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
	var ownerID sql.NullString
	err := s.db.QueryRow(
		"SELECT id, name, owner_id, created_at FROM boards WHERE id = ?", id,
	).Scan(&board.ID, &board.Name, &ownerID, &board.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	board.OwnerID = ownerID.String

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

// ListBoardsByOwner returns the boards owned by a user, newest first.
// The returned boards are summaries (no columns/cards loaded).
func (s *Store) ListBoardsByOwner(ownerID string) ([]Board, error) {
	rows, err := s.db.Query(
		"SELECT id, name, owner_id, created_at FROM boards WHERE owner_id = ? ORDER BY created_at DESC",
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	boards := []Board{}
	for rows.Next() {
		var b Board
		var owner sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &owner, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.OwnerID = owner.String
		boards = append(boards, b)
	}
	return boards, rows.Err()
}

// CreateUser inserts a new user. It returns ErrEmailTaken if the email is
// already registered.
func (s *Store) CreateUser(email, passwordHash, name string) (*User, error) {
	user := &User{
		ID:        uuid.NewString(),
		Email:     email,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(
		"INSERT INTO users (id, email, password_hash, name, created_at) VALUES (?, ?, ?, ?, ?)",
		user.ID, user.Email, passwordHash, user.Name, user.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return user, nil
}

// GetUserByEmail returns the user and their stored password hash for login.
// Returns (nil, "", nil) when no user matches.
func (s *Store) GetUserByEmail(email string) (*User, string, error) {
	user := &User{}
	var hash string
	err := s.db.QueryRow(
		"SELECT id, email, name, password_hash, created_at FROM users WHERE email = ?", email,
	).Scan(&user.ID, &user.Email, &user.Name, &hash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return user, hash, nil
}

// GetUserByID returns the user with the given id, or (nil, nil) if not found.
func (s *Store) GetUserByID(id string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		"SELECT id, email, name, created_at FROM users WHERE id = ?", id,
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}
