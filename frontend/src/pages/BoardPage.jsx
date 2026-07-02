import { useState } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useBoardSocket } from "../lib/useBoardSocket.js";
import { getVoterId } from "../lib/identity.js";
import { useAuth } from "../lib/auth.jsx";
import { api } from "../lib/api.js";
import Column from "../components/Column.jsx";

export default function BoardPage() {
  const { t } = useTranslation();
  const { id } = useParams();
  const { board, connected } = useBoardSocket(id);
  const { user } = useAuth();
  const [copied, setCopied] = useState(false);
  const [togglingStatus, setTogglingStatus] = useState(false);
  const voterId = getVoterId();

  const copyLink = async () => {
    await navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const toggleStatus = async () => {
    if (!board) return;
    setTogglingStatus(true);
    try {
      await api.setBoardStatus(board.id, !board.closed);
    } finally {
      setTogglingStatus(false);
    }
  };

  if (!board) {
    return (
      <div className="board-loading">
        <div className="spinner" />
        <p>{t("board.connecting")}</p>
      </div>
    );
  }

  const totalVotes = board.columns.reduce(
    (sum, col) => sum + col.cards.reduce((s, c) => s + c.votes, 0),
    0
  );

  const isOwner = !!user && user.id === board.ownerId;

  return (
    <div className="board">
      <div className="board-bar">
        <div>
          <h2 className="board-title">{board.name}</h2>
          <p className="board-meta">
            {t("board.votesCast", { count: totalVotes })}
          </p>
        </div>
        <div className="board-actions">
          <span className={`status ${connected ? "online" : "offline"}`}>
            <span className="dot" />
            {connected ? t("board.live") : t("board.reconnecting")}
          </span>
          {isOwner && (
            <button
              className={`btn ${board.closed ? "btn-primary" : "btn-light"}`}
              onClick={toggleStatus}
              disabled={togglingStatus}
            >
              {board.closed ? t("board.resume") : t("board.stop")}
            </button>
          )}
          <button className="btn btn-light" onClick={copyLink}>
            {copied ? t("board.copied") : t("board.share")}
          </button>
        </div>
      </div>

      {board.closed && (
        <div className="board-closed-banner" role="status">
          {t("board.closedBanner")}
        </div>
      )}

      <div className="columns">
        {board.columns.map((column) => (
          <Column
            key={column.id}
            boardId={board.id}
            column={column}
            voterId={voterId}
            closed={board.closed}
          />
        ))}
      </div>
    </div>
  );
}
