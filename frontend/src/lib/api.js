// Thin wrapper around the backend REST API. All paths are relative so the
// Vite dev proxy (and a same-origin production deploy) route them correctly.

const TOKEN_KEY = "voterun:token";

export function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

async function request(path, options = {}) {
  const headers = { "Content-Type": "application/json", ...(options.headers || {}) };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`/api${path}`, { ...options, headers });

  if (!res.ok) {
    // Try to surface the backend's JSON error message.
    let message = `Request failed: ${res.status}`;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      /* non-JSON error body */
    }
    const err = new Error(message);
    err.status = res.status;
    throw err;
  }

  if (res.status === 204 || res.headers.get("content-length") === "0") {
    return null;
  }
  const text = await res.text();
  return text ? JSON.parse(text) : null;
}

export const api = {
  // ---- Auth ----
  register: (email, password, name) =>
    request("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, name }),
    }),

  login: (email, password) =>
    request("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  me: () => request("/auth/me"),

  // ---- Boards ----
  myBoards: () => request("/boards"),

  createBoard: (name) =>
    request("/boards", { method: "POST", body: JSON.stringify({ name }) }),

  getBoard: (id) => request(`/boards/${id}`),

  setBoardStatus: (boardId, closed) =>
    request(`/boards/${boardId}/status`, {
      method: "PATCH",
      body: JSON.stringify({ closed }),
    }),

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
