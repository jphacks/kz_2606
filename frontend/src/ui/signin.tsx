import React, { useState } from 'react';
import { Mail, ArrowRight } from 'lucide-react';

interface RegisterProps {
  onNavigateToLogin?: () => void;
  onStartWithoutAuth?: () => void;
  onRegisterSuccess?: (data: any) => void; // 登録成功時に親へ通知する用（必要に応じて）
}

export default function Register({ onNavigateToLogin, onStartWithoutAuth, onRegisterSuccess }: RegisterProps) {
  const [accountName, setAccountName] = useState('');
  const [password, setPassword] = useState('');
  const [errorMessage, setErrorMessage] = useState(''); // エラーメッセージ用
  const [isLoading, setIsLoading] = useState(false);     // ローディング状態用

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage('');
    setIsLoading(true);

    try {
      // バックエンドの /api/auth/register へ送信
      // ※バックエンド側が username を求めているため accountName を当てはめています
      const response = await fetch('/api/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: accountName,
          password: password,
        }),
      });

      const data = await response.json();

      if (!response.ok) {
        // サーバーからエラーが返ってきた場合
        setErrorMessage(data.error?.message || '新規登録に失敗しました');
        return;
      }

      // 新規登録成功！
      console.log('登録成功:', data);
      alert('ユーザー登録が完了しました！');
      
      if (onRegisterSuccess) {
        onRegisterSuccess(data);
      } else if (onNavigateToLogin) {
        // 成功したら自動でログイン画面に切り替えるなどの動線も作れます
        onNavigateToLogin();
      }

    } catch (err) {
      console.error('通信エラー:', err);
      setErrorMessage('サーバーとの通信に失敗しました');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#F4F7F2] text-[#1B3B2B] flex flex-col justify-between font-sans">
      {/* ヘッダー */}
      <header className="px-6 py-4 flex items-center justify-between border-b border-[#E2EBDD] bg-white/50 backdrop-blur-sm">
        <div className="flex items-center gap-2">
          <div className="w-9 h-9 rounded-xl bg-[#0F5A40] flex items-center justify-center text-white shadow-sm">
            {/* まちえロゴ（S風のアイコン） */}
            <svg className="w-5 h-5 stroke-current fill-none stroke-[2.5]" viewBox="0 0 24 24">
              <path d="M4 6h8a4 4 0 0 1 0 8H8a4 4 0 0 0 0 8h8" />
              <circle cx="4" cy="6" r="1.5" fill="currentColor" />
              <circle cx="16" cy="22" r="1.5" fill="currentColor" />
            </svg>
          </div>
          <span className="text-xl font-bold tracking-tight text-[#0F5A40]">まちえ</span>
        </div>
        <button className="text-sm font-medium text-[#0F5A40] hover:underline cursor-pointer">
          使い方
        </button>
      </header>

      {/* メインコンテンツ */}
      <main className="flex-1 max-w-md w-full mx-auto px-6 py-6 flex flex-col justify-center">
        {/* キャッチコピー */}
        <div className="mb-6">
          <p className="text-xs font-bold text-[#0F5A40] tracking-wide mb-1.5 uppercase">
            みんなの一歩が、ひとつの絵に。
          </p>
          <h1 className="text-3xl font-extrabold tracking-tight text-[#1B3B2B] leading-tight">
            歩いた思い出を、いつでも。
          </h1>
          <p className="text-sm text-[#556B5D] mt-2 leading-relaxed">
            アカウント名とパスワードを登録して、歩いた記録を保存しよう。
          </p>
        </div>

        {/* マッププレースホルダー画像エリア */}
        <div className="mb-6 rounded-2xl overflow-hidden shadow-md border border-[#E2EBDD] bg-[#E8F0E5] h-44 relative flex items-center justify-center">
          <div className="absolute inset-0 bg-gradient-to-br from-[#E2EBDD]/50 to-[#D4E4D0]/50 flex items-center justify-center text-[#7A9A88] text-xs font-medium">
            Map Preview Area
          </div>
        </div>

        {/* 新規登録フォームカード */}
        <div className="bg-white rounded-3xl p-6 shadow-sm border border-[#E2EBDD] mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-bold text-[#1B3B2B]">新規登録</h2>
          </div>

          {/* エラーメッセージ表示エリア */}
          {errorMessage && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-600">
              {errorMessage}
            </div>
          )}

          <form onSubmit={handleRegister} className="space-y-4">
            {/* アカウント名入力 */}
            <div>
              <label className="block text-xs font-semibold text-[#556B5D] mb-1.5">
                アカウント名
              </label>
              <div className="relative flex items-center">
                <span className="absolute left-3.5 text-[#7A9A88]">
                  <Mail className="w-4 h-4" />
                </span>
                <input
                  type="text"
                  value={accountName}
                  onChange={(e) => setAccountName(e.target.value)}
                  placeholder="例 : machie_user"
                  className="w-full pl-10 pr-4 py-3 bg-[#F9FBF8] border border-[#E2EBDD] rounded-xl text-sm text-[#1B3B2B] placeholder-[#9CAEA3] focus:outline-none focus:ring-2 focus:ring-[#0F5A40]/30 focus:border-[#0F5A40] transition-all"
                  required
                />
              </div>
            </div>

            {/* パスワード入力 */}
            <div>
              <label className="block text-xs font-semibold text-[#556B5D] mb-1.5">
                パスワード
              </label>
              <div className="relative flex items-center">
                <span className="absolute left-3.5 text-[#7A9A88]">
                  <Mail className="w-4 h-4" />
                </span>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full pl-10 pr-4 py-3 bg-[#F9FBF8] border border-[#E2EBDD] rounded-xl text-sm text-[#1B3B2B] placeholder-[#9CAEA3] focus:outline-none focus:ring-2 focus:ring-[#0F5A40]/30 focus:border-[#0F5A40] transition-all"
                  required
                />
              </div>
            </div>

            {/* 新規登録ボタン */}
            <button
              type="submit"
              disabled={isLoading}
              className="w-full mt-2 bg-[#0F5A40] hover:bg-[#0C4A34] text-white font-medium py-3.5 px-4 rounded-xl shadow-sm flex items-center justify-center gap-2 transition-colors cursor-pointer disabled:opacity-50"
            >
              <ArrowRight className="w-4 h-4" />
              <span>{isLoading ? '登録中...' : '新規登録'}</span>
            </button>

            {/* ログインせずにはじめるボタン */}
            <button
              type="button"
              onClick={onStartWithoutAuth}
              className="w-full bg-white hover:bg-[#F4F7F2] text-[#0F5A40] border border-[#0F5A40]/30 font-medium py-3.5 px-4 rounded-xl flex items-center justify-center gap-2 transition-colors cursor-pointer"
            >
              <ArrowRight className="w-4 h-4" />
              <span>ログインせずにはじめる</span>
            </button>
          </form>

          <p className="text-[11px] text-[#7A9A88] text-center mt-4 leading-normal">
            アカウント名とパスワードを入力して、新規登録を完了してください。
          </p>
        </div>

        {/* すでにアカウントをお持ちの方セクション */}
        <div className="bg-[#EAEFE6] rounded-3xl p-6 border border-[#D4E4D0] text-center">
          <h3 className="text-base font-bold text-[#1B3B2B] mb-1">
            すでにアカウントをお持ちの方
          </h3>
          <p className="text-xs text-[#556B5D] mb-4">
            登録済みのアカウント名とパスワードでログインしよう。
          </p>
          <button
            onClick={onNavigateToLogin}
            className="w-full bg-white hover:bg-[#F9FBF8] text-[#0F5A40] border border-[#D4E4D0] font-medium py-3 px-4 rounded-xl shadow-sm flex items-center justify-center gap-2 transition-colors cursor-pointer"
          >
            <ArrowRight className="w-4 h-4" />
            <span>ログイン画面へ</span>
          </button>
        </div>
      </main>

      {/* フッター */}
      <footer className="py-4 text-center text-xs text-[#7A9A88] border-t border-[#E2EBDD] bg-white/30">
        <div className="flex justify-center gap-6">
          <a href="#" className="hover:underline">利用規約</a>
          <a href="#" className="hover:underline">プライバシー</a>
        </div>
      </footer>
    </div>
  );
}