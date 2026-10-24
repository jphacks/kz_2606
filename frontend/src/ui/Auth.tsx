// src/ui/Auth.tsx
import { useState } from 'react';
import Login from './login';
import Register from './signin';

export default function Auth() {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const startWithoutAuth = () => window.location.assign('/');

  return mode === 'login' ? (
    <Login
      onNavigateToRegister={() => setMode('register')}
      onStartWithoutAuth={startWithoutAuth}
    />
  ) : (
    <Register
      onNavigateToLogin={() => setMode('login')}
      onStartWithoutAuth={startWithoutAuth}
    />
  );
}