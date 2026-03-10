import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "./AuthContext";

interface ProtectedRouteProps {
  children: React.ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { user, isAuthenticated, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    // Redirect to login with returnTo query param
    const returnTo = location.pathname + location.search;
    return <Navigate to={`/login?returnTo=${encodeURIComponent(returnTo)}`} replace />;
  }

  if (isAuthenticated && user?.must_change_password && location.pathname !== "/settings") {
    return <Navigate to="/settings" replace />;
  }

  return <>{children}</>;
}
