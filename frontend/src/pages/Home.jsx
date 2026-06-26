import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../lib/api.js";

export default function Home() {
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const [boards, setBoards] = useState([]);
  const [loadingBoards, setLoadingBoards] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    let cancelled = false;
    api
      .myBoards()
      .then((list) => {
        if (!cancelled) setBoards(list || []);
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoadingBoards(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const create = async (e) => {
    e.preventDefault();
    setCreating(true);
    setError("");
    try {
      const board = await api.createBoard(name.trim());
      navigate(`/board/${board.id}`);
    } catch (err) {
      setError(err.message || "Could not create board");
      setCreating(false);
    }
  };

  return (
    <div className="home">
      <div className="home-card">
        <h1>Run a better retro.</h1>
        <p className="home-sub">
          Spin up a board, share the link, and collect feedback in real time.
          Everyone votes on what matters most.
        </p>
        <form className="home-form" onSubmit={create}>
          <input
            type="text"
            className="input"
            placeholder="Board name (e.g. Sprint 42 Retro)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
          />
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? "Creating…" : "Create board"}
          </button>
        </form>
        {error && <p className="error">{error}</p>}

        <div className="my-boards">
          <h2>Your boards</h2>
          {loadingBoards ? (
            <p className="muted-text">Loading…</p>
          ) : boards.length === 0 ? (
            <p className="muted-text">
              You haven't created any boards yet. Create one above to get started.
            </p>
          ) : (
            <ul className="board-list">
              {boards.map((b) => (
                <li key={b.id}>
                  <Link to={`/board/${b.id}`}>
                    <span className="board-name">{b.name}</span>
                    <span className="board-date">
                      {new Date(b.createdAt).toLocaleDateString()}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
