package main

// Repository is the storage boundary for VoteRun. The rest of the app depends
// on this interface rather than a concrete database, so swapping the backing
// store (e.g. Postgres -> MySQL, or an in-memory fake for tests) only requires
// a new implementation and a one-line change in main.go.
type Repository interface {
	// Boards, cards, and votes (used by handlers.go).
	CreateBoard(name, ownerID string) (*Board, error)
	GetBoard(id string) (*Board, error)
	ListBoardsByOwner(ownerID string) ([]Board, error)
	SetBoardClosed(boardID, ownerID string, closed bool) (bool, error)
	CreateCard(columnID, text, author string) (string, error)
	UpdateCard(cardID, text string) (string, error)
	DeleteCard(cardID string) (string, error)
	ToggleVote(cardID, voterID string) (string, error)

	// Users (used by auth.go).
	CreateUser(email, passwordHash, name string) (*User, error)
	GetUserByEmail(email string) (*User, string, error)
	GetUserByID(id string) (*User, error)

	// Lifecycle (used by main.go).
	Close() error
}
