import { Routes, Route, Link, useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "./lib/auth.jsx";
import Login from "./pages/Login.jsx";
import Home from "./pages/Home.jsx";
import Landing from "./pages/Landing.jsx";
import BoardPage from "./pages/BoardPage.jsx";

function Header() {
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
      <span className="tagline">Real-time retrospectives</span>
      <div className="header-spacer" />
      {user ? (
        <div className="header-user">
          <span className="avatar">{(user.name || user.email)[0]}</span>
          <span>{user.email}</span>
          <button className="btn btn-light" onClick={handleLogout}>
            Logout
          </button>
        </div>
      ) : (
        onLanding && (
          <div className="header-auth">
            <Link to="/login" className="btn btn-light">
              Sign in
            </Link>
            <Link to="/login" state={{ mode: "register" }} className="btn btn-primary">
              Get started
            </Link>
          </div>
        )
      )}
    </header>
  );
}

function RootRoute() {
  const { user, loading } = useAuth();
  if (loading) {
    return (
      <div className="board-loading">
        <div className="spinner" />
        <p>Loading…</p>
      </div>
    );
  }
  return user ? <Home /> : <Landing />;
}

export default function App() {
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
