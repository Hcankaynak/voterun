import { Routes, Route, Link, useNavigate } from "react-router-dom";
import { useAuth } from "./lib/auth.jsx";
import RequireAuth from "./components/RequireAuth.jsx";
import Login from "./pages/Login.jsx";
import Home from "./pages/Home.jsx";
import BoardPage from "./pages/BoardPage.jsx";

function Header() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

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
      {user && (
        <div className="header-user">
          <span className="avatar">{(user.name || user.email)[0]}</span>
          <span>{user.email}</span>
          <button className="btn btn-light" onClick={handleLogout}>
            Logout
          </button>
        </div>
      )}
    </header>
  );
}

export default function App() {
  return (
    <div className="app">
      <Header />
      <main className="app-main">
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <RequireAuth>
                <Home />
              </RequireAuth>
            }
          />
          <Route
            path="/board/:id"
            element={
              <RequireAuth>
                <BoardPage />
              </RequireAuth>
            }
          />
        </Routes>
      </main>
    </div>
  );
}
