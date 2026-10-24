import type { User } from '../lib/types';
import Login from './login';
import Register from './signin';

interface AuthProps {
  mode: 'login' | 'register';
  onNavigateToLogin: () => void;
  onNavigateToRegister: () => void;
  onStartWithoutAuth: () => void;
  onAuthSuccess: (user: User) => void;
}

export default function Auth({
  mode,
  onNavigateToLogin,
  onNavigateToRegister,
  onStartWithoutAuth,
  onAuthSuccess,
}: AuthProps) {
  return mode === 'login' ? (
    <Login
      onNavigateToRegister={onNavigateToRegister}
      onStartWithoutAuth={onStartWithoutAuth}
      onLoginSuccess={onAuthSuccess}
    />
  ) : (
    <Register
      onNavigateToLogin={onNavigateToLogin}
      onStartWithoutAuth={onStartWithoutAuth}
      onRegisterSuccess={onAuthSuccess}
    />
  );
}