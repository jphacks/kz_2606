import { useEffect, useState, type FormEvent } from 'react';
import { ArrowLeft, ArrowRight, Footprints } from 'lucide-react';
import { joinRoom } from '../lib/api';
import { defaultMemberColor, memberColors } from '../lib/member_colors';
import type { Room, RoomMember, User } from '../lib/types';

interface JoinScreenProps {
  roomCode: string | null;
  user: User | null;
  isSessionLoading: boolean;
  onJoinSuccess: (room: Room, member: Pick<RoomMember, 'id' | 'displayName' | 'color'>) => void;
  onNavigateToLogin: () => void;
  onNavigateToRegister: () => void;
  onBack: () => void;
}

export default function JoinScreen({
  roomCode,
  user,
  isSessionLoading,
  onJoinSuccess,
  onNavigateToLogin,
  onNavigateToRegister,
  onBack,
}: JoinScreenProps) {
  const [displayName, setDisplayName] = useState(user?.username ?? '');
  const [color, setColor] = useState<string>(defaultMemberColor);
  const [errorMessage, setErrorMessage] = useState('');
  const [isJoining, setIsJoining] = useState(false);

  useEffect(() => {
    if (user?.username) setDisplayName(user.username);
  }, [user?.username]);

  const handleJoin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage('');

    const trimmedName = displayName.trim();
    const nameLength = Array.from(trimmedName).length;
    if (nameLength < 1 || nameLength > 16) {
      setErrorMessage('表示名は1〜16文字で入力してください。');
      return;
    }
    if (!roomCode || !/^[A-Z0-9]{6}$/.test(roomCode)) {
      setErrorMessage('招待コードが正しくありません。QRコードを確認してください。');
      return;
    }

    setIsJoining(true);
    try {
      const result = await joinRoom(roomCode, trimmedName, color);
      onJoinSuccess(result.room, result.member);
    } catch (error) {
      console.error('部屋への参加に失敗しました:', error);
      setErrorMessage(error instanceof Error ? error.message : '部屋に参加できませんでした。');
    } finally {
      setIsJoining(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#F4F7F2] text-[#1B3B2B] flex flex-col font-sans">
      <header className="px-6 py-4 flex items-center justify-between border-b border-[#E2EBDD] bg-white/70">
        <button
          type="button"
          onClick={onBack}
          className="flex items-center gap-2 text-sm font-medium text-[#0F5A40]"
        >
          <ArrowLeft className="w-4 h-4" />
          ホームへ戻る
        </button>
        <div className="flex items-center gap-2 text-[#0F5A40]">
          <Footprints className="w-5 h-5" />
          <span className="font-bold">まちえ</span>
        </div>
      </header>

      <main className="flex-1 max-w-md w-full mx-auto px-6 py-8 flex flex-col justify-center">
        <p className="text-xs font-bold text-[#0F5A40] tracking-wide mb-2">招待されたチーム</p>
        <h1 className="text-2xl font-extrabold text-[#1B3B2B]">みんなの地図に参加</h1>
        <p className="text-sm text-[#556B5D] mt-2 mb-6">
          招待コード <span className="font-bold tracking-widest">{roomCode || '不明'}</span>
        </p>

        {isSessionLoading ? (
          <section className="bg-white rounded-3xl p-6 shadow-sm border border-[#E2EBDD] text-sm text-[#556B5D]">
            ログイン状態を確認しています...
          </section>
        ) : !user ? (
          <section className="bg-white rounded-3xl p-6 shadow-sm border border-[#E2EBDD]">
            <p className="text-sm leading-relaxed text-[#556B5D] mb-5">
              チームに参加するにはログインまたはアカウント登録が必要です。認証後、この招待コードに戻ります。
            </p>
            <div className="space-y-3">
              <button
                type="button"
                onClick={onNavigateToLogin}
                className="w-full bg-[#0F5A40] hover:bg-[#0C4A34] text-white font-semibold py-3.5 rounded-xl flex items-center justify-center gap-2"
              >
                ログイン
                <ArrowRight className="w-4 h-4" />
              </button>
              <button
                type="button"
                onClick={onNavigateToRegister}
                className="w-full bg-white text-[#0F5A40] border border-[#0F5A40]/30 font-semibold py-3.5 rounded-xl"
              >
                新規アカウントを登録
              </button>
            </div>
          </section>
        ) : (
          <form onSubmit={handleJoin} className="bg-white rounded-3xl p-6 shadow-sm border border-[#E2EBDD]">
            <label htmlFor="join-display-name" className="block text-xs font-semibold text-[#556B5D] mb-1.5">
              チーム内で表示する名前
            </label>
            <input
              id="join-display-name"
              type="text"
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              maxLength={32}
              autoComplete="nickname"
              className="w-full px-4 py-3 bg-[#F9FBF8] border border-[#E2EBDD] rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-[#0F5A40]/30"
              required
            />
            <p className="text-xs text-[#7A9A88] mt-1">前後の空白を除いて1〜16文字</p>

            <fieldset className="mt-6">
              <legend className="text-xs font-semibold text-[#556B5D] mb-3">軌跡の色</legend>
              <div className="grid grid-cols-5 gap-3">
                {memberColors.map((memberColor) => (
                  <button
                    key={memberColor.hex}
                    type="button"
                    onClick={() => setColor(memberColor.hex)}
                    aria-label={`${memberColor.name}を選択`}
                    aria-pressed={color === memberColor.hex}
                    className={`w-11 h-11 rounded-full border-4 transition ${
                      color === memberColor.hex ? 'border-[#1B3B2B] scale-110' : 'border-white shadow'
                    }`}
                    style={{ backgroundColor: memberColor.hex }}
                  />
                ))}
              </div>
              <p className="text-xs text-[#7A9A88] mt-2">
                選択中: {memberColors.find((memberColor) => memberColor.hex === color)?.name}
              </p>
            </fieldset>

            {errorMessage && (
              <p role="alert" className="mt-4 p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">
                {errorMessage}
              </p>
            )}

            <button
              type="submit"
              disabled={isJoining || !roomCode}
              className="w-full mt-6 bg-[#0F5A40] hover:bg-[#0C4A34] disabled:opacity-50 text-white font-semibold py-3.5 rounded-xl flex items-center justify-center gap-2"
            >
              <span>{isJoining ? '参加しています...' : 'このチームに参加'}</span>
              {!isJoining && <ArrowRight className="w-4 h-4" />}
            </button>
            <button
              type="button"
              onClick={onNavigateToLogin}
              className="w-full mt-3 text-xs text-[#0F5A40] hover:underline"
            >
              別のアカウントでログイン
            </button>
          </form>
        )}
      </main>
    </div>
  );
}
