import { createContext, useContext } from "react";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import type { components } from "@/lib/api/schema";

type User = components["schemas"]["User"];

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

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

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
