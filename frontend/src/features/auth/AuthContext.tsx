import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { AuthContext } from "./auth-context";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const { data, isLoading, isAuthenticated } = useCurrentUser();

  return (
    <AuthContext.Provider
      value={{
        user: data,
        isLoading,
        isAuthenticated,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
