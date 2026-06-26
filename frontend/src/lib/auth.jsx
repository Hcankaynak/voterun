import { createContext, useContext, useEffect, useState } from "react";
import { api, getToken, setToken } from "./api.js";

// Backend-backed authentication. The JWT returned by the API is stored in
// localStorage and attached to every request by the api wrapper. On startup
// we validate any existing token by fetching the current user.

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  // Restore the session from a stored token on first load.
  useEffect(() => {
    let cancelled = false;
    async function restore() {
      if (!getToken()) {
        setLoading(false);
        return;
      }
      try {
        const me = await api.me();
        if (!cancelled) setUser(me);
      } catch {
        setToken(null); // token invalid/expired
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    restore();
    return () => {
      cancelled = true;
    };
  }, []);

  const login = async (email, password) => {
    const { token, user } = await api.login(email.trim(), password);
    setToken(token);
    setUser(user);
  };

  const register = async (email, password, name) => {
    const { token, user } = await api.register(email.trim(), password, name);
    setToken(token);
    setUser(user);
  };

  const logout = () => {
    setToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
