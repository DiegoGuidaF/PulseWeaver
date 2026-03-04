import { Navigate } from "react-router-dom";
import { useAuth } from "@/features/auth/AuthContext";
import { LoginForm } from "@/features/auth/components/LoginForm";

export function LoginPage() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    );
  }

  // Redirect to dashboard if already authenticated
  if (isAuthenticated) {
    return <Navigate to="/devices" replace />;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-md space-y-6 rounded-lg border p-8">
        <div className="space-y-2 text-center">
          <h1 className="text-2xl font-bold">Welcome to WallyDic</h1>
          <p className="text-muted-foreground">
            Sign in to your account to continue
          </p>
        </div>
        <LoginForm />
      </div>
    </div>
  );
}
