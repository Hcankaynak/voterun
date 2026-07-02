import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { api } from "../lib/api.js";

export default function Home() {
  const { t } = useTranslation();
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
      setError(err.message || t("home.createError"));
      setCreating(false);
    }
  };

  return (
    <div className="home">
      <div className="home-card">
        <h1>{t("home.heading")}</h1>
        <p className="home-sub">{t("home.sub")}</p>
        <form className="home-form" onSubmit={create}>
          <input
            type="text"
            className="input"
            placeholder={t("home.boardNamePlaceholder")}
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
          />
          <button type="submit" className="btn btn-primary" disabled={creating}>
            {creating ? t("home.creating") : t("home.createBoard")}
          </button>
        </form>
        {error && <p className="error">{error}</p>}

        <div className="my-boards">
          <h2>{t("home.yourBoards")}</h2>
          {loadingBoards ? (
            <p className="muted-text">{t("common.loading")}</p>
          ) : boards.length === 0 ? (
            <p className="muted-text">{t("home.empty")}</p>
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
