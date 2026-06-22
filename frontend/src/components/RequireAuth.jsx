import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../lib/auth.jsx";

// Redirects unauthenticated visitors to the login page, remembering where
// they were headed so they land back there after signing in.
export default function RequireAuth({ children }) {
  const { user } = useAuth();
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return children;
}
