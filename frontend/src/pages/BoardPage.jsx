import { useState } from "react";
import { useParams } from "react-router-dom";
import { useBoardSocket } from "../lib/useBoardSocket.js";
import { getVoterId } from "../lib/identity.js";
import Column from "../components/Column.jsx";

export default function BoardPage() {
  const { id } = useParams();
  const { board, connected } = useBoardSocket(id);
  const [copied, setCopied] = useState(false);
  const voterId = getVoterId();

  const copyLink = async () => {
    await navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (!board) {
    return (
      <div className="board-loading">
        <div className="spinner" />
        <p>Connecting to board…</p>
      </div>
    );
  }

  const totalVotes = board.columns.reduce(
    (sum, col) => sum + col.cards.reduce((s, c) => s + c.votes, 0),
    0
  );

  return (
    <div className="board">
      <div className="board-bar">
        <div>
          <h2 className="board-title">{board.name}</h2>
          <p className="board-meta">
            {totalVotes} vote{totalVotes === 1 ? "" : "s"} cast
          </p>
        </div>
        <div className="board-actions">
          <span className={`status ${connected ? "online" : "offline"}`}>
            <span className="dot" />
            {connected ? "Live" : "Reconnecting…"}
          </span>
          <button className="btn btn-light" onClick={copyLink}>
            {copied ? "Link copied!" : "Share board"}
          </button>
        </div>
      </div>

      <div className="columns">
        {board.columns.map((column) => (
          <Column
            key={column.id}
            boardId={board.id}
            column={column}
            voterId={voterId}
          />
        ))}
      </div>
    </div>
  );
}
