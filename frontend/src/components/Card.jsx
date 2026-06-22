import { useState } from "react";
import { api } from "../lib/api.js";

export default function Card({ card, voterId }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(card.text);
  const [busy, setBusy] = useState(false);

  const hasVoted = card.voters?.includes(voterId);

  const toggleVote = async () => {
    setBusy(true);
    try {
      await api.vote(card.id, voterId);
    } finally {
      setBusy(false);
    }
  };

  const saveEdit = async () => {
    const trimmed = draft.trim();
    if (!trimmed || trimmed === card.text) {
      setEditing(false);
      setDraft(card.text);
      return;
    }
    await api.updateCard(card.id, trimmed);
    setEditing(false);
  };

  const remove = async () => {
    if (!confirm("Delete this card?")) return;
    await api.deleteCard(card.id);
  };

  return (
    <article className="card">
      {editing ? (
        <textarea
          className="card-edit"
          value={draft}
          autoFocus
          rows={3}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={saveEdit}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              saveEdit();
            }
            if (e.key === "Escape") {
              setEditing(false);
              setDraft(card.text);
            }
          }}
        />
      ) : (
        <p className="card-text" onClick={() => setEditing(true)}>
          {card.text}
        </p>
      )}

      <footer className="card-foot">
        <span className="author">{card.author}</span>
        <div className="card-controls">
          <button className="link-btn" onClick={remove} title="Delete">
            ✕
          </button>
          <button
            className={`vote-btn ${hasVoted ? "voted" : ""}`}
            onClick={toggleVote}
            disabled={busy}
            title={hasVoted ? "Remove your vote" : "Vote for this card"}
          >
            ▲ {card.votes}
          </button>
        </div>
      </footer>
    </article>
  );
}
