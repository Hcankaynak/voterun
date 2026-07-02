import { useEffect } from "react";
import { Routes, Route, Link, useNavigate, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "./lib/auth.jsx";
import Login from "./pages/Login.jsx";
import Home from "./pages/Home.jsx";
import Landing from "./pages/Landing.jsx";
import BoardPage from "./pages/BoardPage.jsx";

const LANGUAGES = [
  { code: "en", label: "EN" },
  { code: "tr", label: "TR" },
];

function LanguageSwitcher() {
  const { i18n } = useTranslation();
  const current = i18n.resolvedLanguage;

  return (
    <div className="lang-switch" role="group" aria-label="Language">
      {LANGUAGES.map((lng) => (
        <button
          key={lng.code}
          type="button"
          className={`lang-option ${current === lng.code ? "active" : ""}`}
          aria-pressed={current === lng.code}
          onClick={() => i18n.changeLanguage(lng.code)}
        >
          {lng.label}
        </button>
      ))}
    </div>
  );
}

function Header() {
  const { t } = useTranslation();
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const onLanding = location.pathname === "/";

  const handleLogout = () => {
    logout();
    navigate("/login", { replace: true });
  };

  return (
    <header className="app-header">
      <Link to="/" className="brand">
        Vote<span>Run</span>
      </Link>
      <span className="tagline">{t("nav.tagline")}</span>
      <div className="header-spacer" />
      <LanguageSwitcher />
      {user ? (
        <div className="header-user">
          <span className="avatar">{(user.name || user.email)[0]}</span>
          <span>{user.email}</span>
          <button className="btn btn-light" onClick={handleLogout}>
            {t("nav.logout")}
          </button>
        </div>
      ) : (
        onLanding && (
          <div className="header-auth">
            <Link to="/login" className="btn btn-light">
              {t("nav.signIn")}
            </Link>
            <Link to="/login" state={{ mode: "register" }} className="btn btn-primary">
              {t("nav.getStarted")}
            </Link>
          </div>
        )
      )}
    </header>
  );
}

function RootRoute() {
  const { t } = useTranslation();
  const { user, loading } = useAuth();
  if (loading) {
    return (
      <div className="board-loading">
        <div className="spinner" />
        <p>{t("common.loading")}</p>
      </div>
    );
  }
  return user ? <Home /> : <Landing />;
}

export default function App() {
  const { t, i18n } = useTranslation();

  useEffect(() => {
    document.documentElement.lang = i18n.resolvedLanguage;
    document.title = t("app.title");
  }, [t, i18n.resolvedLanguage]);

  return (
    <div className="app">
      <Header />
      <main className="app-main">
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<RootRoute />} />
          <Route path="/board/:id" element={<BoardPage />} />
        </Routes>
      </main>
    </div>
  );
}
