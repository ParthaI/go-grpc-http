import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';

interface AuthState {
  token: string | null;
  userId: string | null;
  email: string | null;
  isAuthenticated: boolean;
  login: (token: string, userId: string, email: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'));
  const [userId, setUserId] = useState<string | null>(() => localStorage.getItem('userId'));
  const [email, setEmail] = useState<string | null>(() => localStorage.getItem('email'));

  useEffect(() => {
    token ? localStorage.setItem('token', token) : localStorage.removeItem('token');
    userId ? localStorage.setItem('userId', userId) : localStorage.removeItem('userId');
    email ? localStorage.setItem('email', email) : localStorage.removeItem('email');
  }, [token, userId, email]);

  const login = useCallback((t: string, uid: string, e: string) => {
    setToken(t);
    setUserId(uid);
    setEmail(e);
  }, []);

  const logout = useCallback(() => {
    setToken(null);
    setUserId(null);
    setEmail(null);
  }, []);

  return (
    <AuthContext.Provider value={{ token, userId, email, isAuthenticated: !!token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
