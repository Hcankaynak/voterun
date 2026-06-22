import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api.js";

export default function Home() {
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

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
        <ul className="home-features">
          <li>Live sync across every participant</li>
          <li>Vote to surface the team's priorities</li>
          <li>No sign-up required — just share the link</li>
        </ul>
      </div>
    </div>
  );
}
