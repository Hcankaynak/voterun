import { createContext, useContext, useEffect, useState } from "react";

// Lightweight client-side auth so the app matches voterun.app's login flow
// without requiring a real auth backend. Credentials live in localStorage.
// A demo account is seeded so you can sign in immediately.

const SESSION_KEY = "voterun:session";
const USERS_KEY = "voterun:users";

const DEMO_USER = { email: "burcu@burcu.com", password: "123456", name: "Burcu" };

function loadUsers() {
  try {
    const stored = JSON.parse(localStorage.getItem(USERS_KEY)) || [];
    if (!stored.some((u) => u.email === DEMO_USER.email)) {
      stored.push(DEMO_USER);
    }
    return stored;
  } catch {
    return [DEMO_USER];
  }
}

function saveUsers(users) {
  localStorage.setItem(USERS_KEY, JSON.stringify(users));
}

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem(SESSION_KEY));
    } catch {
      return null;
    }
  });

  useEffect(() => {
    if (user) localStorage.setItem(SESSION_KEY, JSON.stringify(user));
    else localStorage.removeItem(SESSION_KEY);
  }, [user]);

  const login = (email, password) => {
    const found = loadUsers().find(
      (u) => u.email.toLowerCase() === email.trim().toLowerCase()
    );
    if (!found || found.password !== password) {
      throw new Error("Invalid email or password.");
    }
    setUser({ email: found.email, name: found.name || found.email.split("@")[0] });
  };

  const register = (email, password) => {
    const users = loadUsers();
    const normalized = email.trim().toLowerCase();
    if (users.some((u) => u.email.toLowerCase() === normalized)) {
      throw new Error("An account with that email already exists.");
    }
    const newUser = { email: email.trim(), password, name: email.split("@")[0] };
    users.push(newUser);
    saveUsers(users);
    setUser({ email: newUser.email, name: newUser.name });
  };

  const loginWithGoogle = () => {
    // Demo-only Google sign-in placeholder.
    setUser({ email: "guest@google.com", name: "Google User" });
  };

  const logout = () => setUser(null);

  return (
    <AuthContext.Provider
      value={{ user, login, register, loginWithGoogle, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
