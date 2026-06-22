package main

import "time"

// Board is a single retrospective board that participants collaborate on.
type Board struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Columns   []Column  `json:"columns"`
}

// Column groups cards within a board (e.g. "What went well").
type Column struct {
	ID       string `json:"id"`
	BoardID  string `json:"boardId"`
	Title    string `json:"title"`
	Position int    `json:"position"`
	Cards    []Card `json:"cards"`
}

// Card is a single piece of feedback authored by a participant.
type Card struct {
	ID        string    `json:"id"`
	ColumnID  string    `json:"columnId"`
	Text      string    `json:"text"`
	Author    string    `json:"author"`
	Votes     int       `json:"votes"`
	Voters    []string  `json:"voters"`
	CreatedAt time.Time `json:"createdAt"`
}
