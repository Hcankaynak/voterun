import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../lib/auth.jsx";

// Redirects unauthenticated visitors to the login page, remembering where
// they were headed so they land back there after signing in.
export default function RequireAuth({ children }) {
  const { user, loading } = useAuth();
  const location = useLocation();

  // Avoid redirecting to /login while we validate a stored token on refresh.
  if (loading) {
    return (
      <div className="board-loading">
        <div className="spinner" />
        <p>Loading…</p>
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return children;
}
