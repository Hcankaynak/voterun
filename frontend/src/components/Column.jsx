import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../lib/api.js";
import { getName, setName as persistName } from "../lib/identity.js";
import Card from "./Card.jsx";

export default function Column({ boardId, column, voterId }) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [author, setAuthor] = useState(getName());
  const [adding, setAdding] = useState(false);

  const addCard = async (e) => {
    e.preventDefault();
    const trimmed = text.trim();
    if (!trimmed) return;
    setAdding(true);
    if (author.trim()) persistName(author.trim());
    try {
      await api.createCard(boardId, column.id, trimmed, author.trim());
      setText("");
    } finally {
      setAdding(false);
    }
  };

  const sorted = [...column.cards].sort((a, b) => b.votes - a.votes);
  const title = t(`column.defaults.${column.title}`, {
    defaultValue: column.title,
  });

  return (
    <section className="column">
      <header className="column-head">
        <h3>{title}</h3>
        <span className="count">{column.cards.length}</span>
      </header>

      <div className="card-list">
        {sorted.map((card) => (
          <Card key={card.id} card={card} voterId={voterId} />
        ))}
      </div>

      <form className="add-card" onSubmit={addCard}>
        <textarea
          placeholder={t("column.addPlaceholder")}
          value={text}
          onChange={(e) => setText(e.target.value)}
          rows={2}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) addCard(e);
          }}
        />
        <div className="add-card-row">
          <input
            className="author-input"
            placeholder={t("column.namePlaceholder")}
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            maxLength={40}
          />
          <button
            type="submit"
            className="btn btn-primary"
            disabled={adding || !text.trim()}
          >
            {t("column.add")}
          </button>
        </div>
      </form>
    </section>
  );
}
