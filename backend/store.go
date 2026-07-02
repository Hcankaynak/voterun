package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// ErrEmailTaken is returned when registering an email that already exists.
var ErrEmailTaken = errors.New("email already registered")

// ErrBoardClosed is returned when a mutation is attempted on a closed board.
var ErrBoardClosed = errors.New("board is closed")

// PostgresStore is the PostgreSQL-backed implementation of Repository.
type PostgresStore struct {
	db *sql.DB
}

// Compile-time check that PostgresStore satisfies the Repository interface.
var _ Repository = (*PostgresStore)(nil)

// NewPostgresStore connects to PostgreSQL using the given DSN and runs migrations.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// Postgres handles concurrent connections well, so allow a real pool
	// instead of the single-writer model SQLite required.
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	s := &PostgresStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	name          TEXT NOT NULL,
	created_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS boards (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	owner_id   TEXT,
	created_at TIMESTAMPTZ NOT NULL
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
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS votes (
	card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
	voter_id TEXT NOT NULL,
	PRIMARY KEY (card_id, voter_id)
);

ALTER TABLE boards ADD COLUMN IF NOT EXISTS closed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_columns_board ON columns(board_id);
CREATE INDEX IF NOT EXISTS idx_cards_column ON cards(column_id);
CREATE INDEX IF NOT EXISTS idx_votes_card ON votes(card_id);
CREATE INDEX IF NOT EXISTS idx_boards_owner ON boards(owner_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

var defaultColumns = []string{"What went well", "What didn't go well", "Action items"}

// CreateBoard inserts a new board (owned by ownerID) seeded with the default
// retro columns.
func (s *PostgresStore) CreateBoard(name, ownerID string) (*Board, error) {
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
		"INSERT INTO boards (id, name, owner_id, created_at) VALUES ($1, $2, $3, $4)",
		board.ID, board.Name, board.OwnerID, board.CreatedAt,
	); err != nil {
		return nil, err
	}

	for i, title := range defaultColumns {
		if _, err := tx.Exec(
			"INSERT INTO columns (id, board_id, title, position) VALUES ($1, $2, $3, $4)",
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
func (s *PostgresStore) GetBoard(id string) (*Board, error) {
	board := &Board{}
	var ownerID sql.NullString
	err := s.db.QueryRow(
		"SELECT id, name, owner_id, closed, created_at FROM boards WHERE id = $1", id,
	).Scan(&board.ID, &board.Name, &ownerID, &board.Closed, &board.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	board.OwnerID = ownerID.String

	colRows, err := s.db.Query(
		"SELECT id, board_id, title, position FROM columns WHERE board_id = $1 ORDER BY position",
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
WHERE col.board_id = $1
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
WHERE col.board_id = $1`, id)
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

// boardIDForColumn returns the board a column belongs to along with whether that
// board is closed.
func (s *PostgresStore) boardIDForColumn(columnID string) (string, bool, error) {
	var boardID string
	var closed bool
	err := s.db.QueryRow(`
SELECT b.id, b.closed
FROM columns col
JOIN boards b ON b.id = col.board_id
WHERE col.id = $1`, columnID).Scan(&boardID, &closed)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return boardID, closed, err
}

// boardIDForCard returns the board a card belongs to along with whether that
// board is closed.
func (s *PostgresStore) boardIDForCard(cardID string) (string, bool, error) {
	var boardID string
	var closed bool
	err := s.db.QueryRow(`
SELECT b.id, b.closed
FROM cards c
JOIN columns col ON col.id = c.column_id
JOIN boards b ON b.id = col.board_id
WHERE c.id = $1`, cardID).Scan(&boardID, &closed)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return boardID, closed, err
}

// SetBoardClosed sets a board's closed flag, but only when ownerID owns the
// board. It returns ok=false when the board does not exist or is owned by
// someone else, so the caller can respond with a 403/404.
func (s *PostgresStore) SetBoardClosed(boardID, ownerID string, closed bool) (bool, error) {
	res, err := s.db.Exec(
		"UPDATE boards SET closed = $1 WHERE id = $2 AND owner_id = $3",
		closed, boardID, ownerID,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// CreateCard adds a new card to a column and returns the owning board id.
func (s *PostgresStore) CreateCard(columnID, text, author string) (string, error) {
	boardID, closed, err := s.boardIDForColumn(columnID)
	if err != nil || boardID == "" {
		return "", err
	}
	if closed {
		return boardID, ErrBoardClosed
	}
	if author == "" {
		author = "Anonymous"
	}
	_, err = s.db.Exec(
		"INSERT INTO cards (id, column_id, text, author, created_at) VALUES ($1, $2, $3, $4, $5)",
		uuid.NewString(), columnID, text, author, time.Now().UTC(),
	)
	return boardID, err
}

// UpdateCard edits a card's text and returns the owning board id.
func (s *PostgresStore) UpdateCard(cardID, text string) (string, error) {
	boardID, closed, err := s.boardIDForCard(cardID)
	if err != nil || boardID == "" {
		return "", err
	}
	if closed {
		return boardID, ErrBoardClosed
	}
	_, err = s.db.Exec("UPDATE cards SET text = $1 WHERE id = $2", text, cardID)
	return boardID, err
}

// DeleteCard removes a card and returns the owning board id.
func (s *PostgresStore) DeleteCard(cardID string) (string, error) {
	boardID, closed, err := s.boardIDForCard(cardID)
	if err != nil || boardID == "" {
		return "", err
	}
	if closed {
		return boardID, ErrBoardClosed
	}
	_, err = s.db.Exec("DELETE FROM cards WHERE id = $1", cardID)
	return boardID, err
}

// ToggleVote adds or removes a participant's vote on a card. The toggle is
// atomic: an INSERT ... ON CONFLICT DO NOTHING either records the vote or, if it
// already existed (0 rows affected), the vote is removed with a DELETE.
func (s *PostgresStore) ToggleVote(cardID, voterID string) (string, error) {
	boardID, closed, err := s.boardIDForCard(cardID)
	if err != nil || boardID == "" {
		return "", err
	}
	if closed {
		return boardID, ErrBoardClosed
	}

	res, err := s.db.Exec(
		"INSERT INTO votes (card_id, voter_id) VALUES ($1, $2) ON CONFLICT (card_id, voter_id) DO NOTHING",
		cardID, voterID,
	)
	if err != nil {
		return "", err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if inserted == 0 {
		// The vote already existed, so this toggle removes it.
		if _, err := s.db.Exec(
			"DELETE FROM votes WHERE card_id = $1 AND voter_id = $2", cardID, voterID,
		); err != nil {
			return "", err
		}
	}
	return boardID, nil
}

// ListBoardsByOwner returns the boards owned by a user, newest first.
// The returned boards are summaries (no columns/cards loaded).
func (s *PostgresStore) ListBoardsByOwner(ownerID string) ([]Board, error) {
	rows, err := s.db.Query(
		"SELECT id, name, owner_id, closed, created_at FROM boards WHERE owner_id = $1 ORDER BY created_at DESC",
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
		if err := rows.Scan(&b.ID, &b.Name, &owner, &b.Closed, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.OwnerID = owner.String
		boards = append(boards, b)
	}
	return boards, rows.Err()
}

// CreateUser inserts a new user. It returns ErrEmailTaken if the email is
// already registered.
func (s *PostgresStore) CreateUser(email, passwordHash, name string) (*User, error) {
	user := &User{
		ID:        uuid.NewString(),
		Email:     email,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(
		"INSERT INTO users (id, email, password_hash, name, created_at) VALUES ($1, $2, $3, $4, $5)",
		user.ID, user.Email, passwordHash, user.Name, user.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return user, nil
}

// GetUserByEmail returns the user and their stored password hash for login.
// Returns (nil, "", nil) when no user matches.
func (s *PostgresStore) GetUserByEmail(email string) (*User, string, error) {
	user := &User{}
	var hash string
	err := s.db.QueryRow(
		"SELECT id, email, name, password_hash, created_at FROM users WHERE email = $1", email,
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
func (s *PostgresStore) GetUserByID(id string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		"SELECT id, email, name, created_at FROM users WHERE id = $1", id,
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
func (s *PostgresStore) Close() error {
	return s.db.Close()
}
