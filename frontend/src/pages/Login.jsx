import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../lib/auth.jsx";

function GoogleIcon() {
  return (
    <svg className="google-icon" viewBox="0 0 48 48" aria-hidden="true">
      <path
        fill="#EA4335"
        d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
      />
      <path
        fill="#4285F4"
        d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
      />
      <path
        fill="#FBBC05"
        d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
      />
      <path
        fill="#34A853"
        d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
      />
    </svg>
  );
}

export default function Login() {
  const { login, register, loginWithGoogle } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const from = location.state?.from?.pathname || "/";

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const go = () => navigate(from, { replace: true });

  const handleLogin = (e) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      login(email, password);
      go();
    } catch (err) {
      setError(err.message);
      setBusy(false);
    }
  };

  const handleRegister = () => {
    setError("");
    if (!email.trim() || !password) {
      setError("Enter an email and password to register.");
      return;
    }
    try {
      register(email, password);
      go();
    } catch (err) {
      setError(err.message);
    }
  };

  const handleGoogle = () => {
    loginWithGoogle();
    go();
  };

  return (
    <div className="auth">
      <div className="auth-card">
        <div className="auth-logo">
          <div className="mark">V</div>
          <h1>Vote Run App</h1>
        </div>
        <p className="auth-sub">Sign in to run your retrospectives.</p>

        <form onSubmit={handleLogin}>
          <div className="field">
            <label htmlFor="email">Email address</label>
            <input
              id="email"
              className="input"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div className="field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              className="input"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          {error && <p className="error">{error}</p>}

          <div className="auth-actions">
            <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
              Login
            </button>
            <button
              type="button"
              className="btn btn-outline btn-block"
              onClick={handleRegister}
            >
              Register
            </button>
          </div>
        </form>

        <div className="divider">OR</div>

        <button className="btn btn-light google-btn" onClick={handleGoogle}>
          <GoogleIcon />
          Login with Google
        </button>

        <div className="auth-hint">
          Demo account — email <code>burcu@burcu.com</code> · password{" "}
          <code>123456</code>
        </div>
      </div>
    </div>
  );
}
