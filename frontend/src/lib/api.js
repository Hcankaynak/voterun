// Thin wrapper around the backend REST API. All paths are relative so the
// Vite dev proxy (and a same-origin production deploy) route them correctly.

async function request(path, options = {}) {
  const res = await fetch(`/api${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const message = await res.text();
    throw new Error(message || `Request failed: ${res.status}`);
  }
  if (res.status === 204 || res.headers.get("content-length") === "0") {
    return null;
  }
  const text = await res.text();
  return text ? JSON.parse(text) : null;
}

export const api = {
  createBoard: (name) =>
    request("/boards", { method: "POST", body: JSON.stringify({ name }) }),

  getBoard: (id) => request(`/boards/${id}`),

  createCard: (boardId, columnId, text, author) =>
    request(`/boards/${boardId}/cards`, {
      method: "POST",
      body: JSON.stringify({ columnId, text, author }),
    }),

  updateCard: (cardId, text) =>
    request(`/cards/${cardId}`, {
      method: "PATCH",
      body: JSON.stringify({ text }),
    }),

  deleteCard: (cardId) => request(`/cards/${cardId}`, { method: "DELETE" }),

  vote: (cardId, voterId) =>
    request(`/cards/${cardId}/vote`, {
      method: "POST",
      body: JSON.stringify({ voterId }),
    }),
};
