import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "../lib/auth.jsx";

export default function Login() {
  const { t } = useTranslation();
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const from = location.state?.from?.pathname || "/";

  const [mode, setMode] = useState(
    location.state?.mode === "register" ? "register" : "login"
  ); // "login" | "register"
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
      setError(err.message || t("login.error"));
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
          <h1>{t("login.title")}</h1>
        </div>
        <p className="auth-sub">
          {isRegister
            ? t("login.subtitleRegister")
            : t("login.subtitleLogin")}
        </p>

        <form onSubmit={submit}>
          {isRegister && (
            <div className="field">
              <label htmlFor="name">{t("login.nameLabel")}</label>
              <input
                id="name"
                className="input"
                type="text"
                autoComplete="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("login.namePlaceholder")}
                maxLength={60}
              />
            </div>
          )}

          <div className="field">
            <label htmlFor="email">{t("login.emailLabel")}</label>
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
            <label htmlFor="password">{t("login.passwordLabel")}</label>
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
            {busy
              ? t("login.busy")
              : isRegister
              ? t("login.createAccount")
              : t("login.login")}
          </button>
        </form>

        <p className="auth-switch">
          {isRegister ? (
            <>
              {t("login.haveAccount")}{" "}
              <button className="link" onClick={() => switchMode("login")}>
                {t("login.signIn")}
              </button>
            </>
          ) : (
            <>
              {t("login.newToVoteRun")}{" "}
              <button className="link" onClick={() => switchMode("register")}>
                {t("login.createOne")}
              </button>
            </>
          )}
        </p>
      </div>
    </div>
  );
}
