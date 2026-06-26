import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../lib/auth.jsx";

export default function Login() {
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const from = location.state?.from?.pathname || "/";

  const [mode, setMode] = useState("login"); // "login" | "register"
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const isRegister = mode === "register";

  const submit = async (e) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      if (isRegister) {
        await register(email, password, name);
      } else {
        await login(email, password);
      }
      navigate(from, { replace: true });
    } catch (err) {
      setError(err.message || "Something went wrong.");
      setBusy(false);
    }
  };

  const switchMode = (next) => {
    setMode(next);
    setError("");
  };

  return (
    <div className="auth">
      <div className="auth-card">
        <div className="auth-logo">
          <div className="mark">V</div>
          <h1>Vote Run App</h1>
        </div>
        <p className="auth-sub">
          {isRegister
            ? "Create an account to run your retrospectives."
            : "Sign in to run your retrospectives."}
        </p>

        <form onSubmit={submit}>
          {isRegister && (
            <div className="field">
              <label htmlFor="name">Name</label>
              <input
                id="name"
                className="input"
                type="text"
                autoComplete="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Optional"
                maxLength={60}
              />
            </div>
          )}

          <div className="field">
            <label htmlFor="email">Email address</label>
            <input
              id="email"
              className="input"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>

          <div className="field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              className="input"
              type="password"
              autoComplete={isRegister ? "new-password" : "current-password"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={isRegister ? 6 : undefined}
            />
          </div>

          {error && <p className="error">{error}</p>}

          <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
            {busy ? "Please wait…" : isRegister ? "Create account" : "Login"}
          </button>
        </form>

        <p className="auth-switch">
          {isRegister ? (
            <>
              Already have an account?{" "}
              <button className="link" onClick={() => switchMode("login")}>
                Sign in
              </button>
            </>
          ) : (
            <>
              New to VoteRun?{" "}
              <button className="link" onClick={() => switchMode("register")}>
                Create an account
              </button>
            </>
          )}
        </p>
      </div>
    </div>
  );
}
