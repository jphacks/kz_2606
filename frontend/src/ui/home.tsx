import { useEffect, useState } from 'react';
import { Waypoints, Footprints, Flag, Plus, X, History, QrCode, LogIn, UserPlus } from 'lucide-react';
import QRCode from 'qrcode';
import { buildInviteUrl, createRoom } from '../lib/api';

interface HomeProps {
  onNavigateToLogin?: () => void;
  onNavigateToRegister?: () => void;
  username?: string;
}

export default function Home({ onNavigateToLogin, onNavigateToRegister, username }: HomeProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [teamName, setTeamName] = useState('');
  const [roomCode, setRoomCode] = useState<string | null>(null);
  const [isCreatingTeam, setIsCreatingTeam] = useState(false);
  const [createTeamError, setCreateTeamError] = useState('');
  const [inviteQrCode, setInviteQrCode] = useState<string | null>(null);
  const [qrCodeError, setQrCodeError] = useState<string | null>(null);

  // QRには招待URL（`?room=<CODE>`）を載せる。仕様10章
  // 部屋コードが無いうちは生成しない（現在ページのURLを載せても参加できない）
  useEffect(() => {
    if (!isModalOpen || !roomCode) {
      setInviteQrCode(null);
      setQrCodeError(null);
      return;
    }

    let cancelled = false;
    QRCode.toDataURL(buildInviteUrl(roomCode), { width: 192, margin: 1 })
      .then((dataUrl) => {
        if (!cancelled) {
          setInviteQrCode(dataUrl);
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setQrCodeError(error instanceof Error ? error.message : String(error));
        }
      });

    return () => {
      cancelled = true;
    };
  }, [isModalOpen, roomCode]);

  const handleCreateTeam = async () => {
    const title = teamName.trim() || 'まちあるき部';
    setCreateTeamError('');
    setIsCreatingTeam(true);
    try {
      const room = await createRoom(title);
      setRoomCode(room.code);
    } catch (err) {
      console.error('チームを作成できませんでした:', err);
      setCreateTeamError(err instanceof Error ? err.message : 'チームを作成できませんでした');
    } finally {
      setIsCreatingTeam(false);
    }
  };

  return (
    <div
      className="min-h-screen bg-[#FAFAFA] font-sans text-gray-800 relative pb-12 overflow-x-hidden"
      style={{ colorScheme: 'light' }}
    >
      {/* --- ヘッダー（ステータスバーのセーフエリア対応） --- */}
      <header className="flex justify-between items-center px-5 pb-5 pt-[max(1.25rem,env(safe-area-inset-top))]">
        <div className="flex items-center gap-2 text-[#0E7861]">
          <Waypoints size={28} strokeWidth={2.5} />
          <span className="text-xl font-bold tracking-wider text-gray-900">まちえ</span>
        </div>

        {/* 右側のエリア：認証状態 ＆ 使い方 */}
        <div className="flex items-center gap-3">
          {username ? (
            <span className="text-xs font-bold text-[#0E7861]">{username}</span>
          ) : (
            <>
              {onNavigateToLogin && (
                <button
                  onClick={onNavigateToLogin}
                  className="bg-[#0E7861]/10 hover:bg-[#0E7861]/20 text-[#0E7861] border border-[#0E7861]/30 text-xs font-bold px-2.5 py-1.5 rounded-lg flex items-center gap-1 transition cursor-pointer"
                >
                  <LogIn size={14} />
                  <span>ログイン</span>
                </button>
              )}

              {onNavigateToRegister && (
                <button
                  onClick={onNavigateToRegister}
                  className="bg-[#0E7861] text-white text-xs font-bold px-2.5 py-1.5 rounded-lg flex items-center gap-1 shadow-sm transition cursor-pointer"
                >
                  <UserPlus size={14} />
                  <span>登録</span>
                </button>
              )}
            </>
          )}

          <button className="text-sm text-[#0E7861] font-medium ml-1">使い方</button>
        </div>
      </header>

      {/* --- メインコンテンツ --- */}
      <main className="px-5">
        {/* キャッチコピー */}
        <div className="mt-4 mb-6">
          <p className="text-[#0E7861] text-sm font-bold leading-relaxed mb-2">
            一人でも、みんなでも。<br />歩いた軌跡をアートに。
          </p>
          <h1 className="text-3xl font-extrabold text-gray-900 leading-tight">
            一人でも、チームでも。<br />GPSアートをはじめる。
          </h1>
          <p className="text-gray-500 text-sm mt-3 leading-relaxed">
            一人で自由に歩き、チームで協力して、自分の好きな場所をアートに。
          </p>
        </div>

        {/* ヒーロー画像 */}
        <div className="relative w-full h-56 rounded-2xl overflow-hidden mb-10 shadow-inner bg-gray-100 flex items-center justify-center">
          <img 
            src="/hero-map.png" 
            alt="GPSアートの軌跡" 
            className="w-full h-full object-cover"
          />
        </div>

        {/* アクションセクション */}
        <div className="mb-8">
          <div className="flex justify-between items-end mb-4">
            <h2 className="text-lg font-bold text-gray-900 leading-tight">
              アートをはじめる
            </h2>
            <span className="text-xs text-gray-500">登録・インストール不要</span>
          </div>

          <div className="grid grid-cols-2 gap-4">
            {/* 一人ではじめる ボタン */}
            <button className="bg-white border border-gray-100 p-5 rounded-2xl shadow-sm flex flex-col items-start gap-4 active:bg-gray-50 transition cursor-pointer">
              <div className="bg-[#EBF5F3] p-3 rounded-full text-[#0E7861]">
                <Footprints size={24} />
              </div>
              <div className="text-left">
                <h3 className="font-bold text-gray-900 text-sm mb-1">一人ではじめる</h3>
                <p className="text-xs text-gray-500">好きな場所を自由に歩く</p>
              </div>
            </button>

            {/* チームではじめる ボタン */}
            <button 
              onClick={() => setIsModalOpen(true)}
              className="bg-white border border-gray-100 p-5 rounded-2xl shadow-sm flex flex-col items-start gap-4 active:bg-gray-50 transition cursor-pointer"
            >
              <div className="bg-[#0E7861] p-3 rounded-full text-white">
                <Flag size={24} />
              </div>
              <div className="text-left">
                <h3 className="font-bold text-gray-900 text-sm mb-1">チームではじめる</h3>
                <p className="text-xs text-gray-500">みんなで協力してアートを作る</p>
              </div>
            </button>
          </div>

          {/* 活動履歴を見る ボタン */}
          <button
            onClick={() => window.location.assign('/result')}
            className="w-full mt-4 bg-white border border-gray-100 p-5 rounded-2xl shadow-sm flex items-center gap-4 active:bg-gray-50 transition cursor-pointer"
          >
            <div className="bg-[#EBF5F3] p-3 rounded-full text-[#0E7861]">
              <History size={24} />
            </div>
            <div className="text-left">
              <h3 className="font-bold text-gray-900 text-sm mb-1">活動履歴を見る</h3>
              <p className="text-xs text-gray-500">参加回数・距離・時間・完成アート</p>
            </div>
          </button>
        </div>
      </main>

      {/* --- チーム参加用 ボトムシート（モーダル） --- */}
      <div 
        className={`fixed inset-0 bg-black/40 z-40 transition-opacity duration-300 ${isModalOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}
        onClick={() => setIsModalOpen(false)}
        aria-hidden={!isModalOpen}
      ></div>

      <div 
        className={`fixed bottom-0 left-0 right-0 bg-[#FAFAFA] rounded-t-3xl z-50 transform transition-transform duration-300 ease-out flex flex-col max-h-[90vh] ${isModalOpen ? 'translate-y-0' : 'translate-y-full pointer-events-none'}`}
        aria-hidden={!isModalOpen}
      >
        <div className="w-full flex justify-center pt-3 pb-1">
          <div className="w-12 h-1.5 bg-gray-300 rounded-full"></div>
        </div>

        {/* ホームインジケーターのセーフエリア対応 */}
        <div className="px-6 pb-[max(2rem,env(safe-area-inset-bottom))] pt-2 overflow-y-auto">
          <div className="flex justify-between items-start mb-6">
            <div>
              <h2 className="text-xl font-bold text-gray-900 mb-1">チームではじめる</h2>
              <p className="text-sm text-gray-500">チームを作るか、招待から参加しましょう</p>
            </div>
            <button 
              onClick={() => setIsModalOpen(false)}
              className="p-2 bg-gray-100 rounded-full text-gray-600 hover:bg-gray-200 transition cursor-pointer"
              aria-label="閉じる"
            >
              <X size={20} />
            </button>
          </div>

          <div className="mb-8">
            <h3 className="font-bold text-gray-900 mb-3">チームを作る</h3>
            <div className="bg-white border border-gray-200 rounded-xl flex items-center p-3 mb-3 shadow-sm focus-within:border-[#0E7861] transition">
              <Flag size={20} className="text-[#0E7861] mr-3" />
              <input
                type="text"
                value={teamName}
                onChange={(e) => setTeamName(e.target.value)}
                placeholder="まちあるき部"
                maxLength={40}
                className="w-full outline-none text-gray-800 placeholder-gray-400 bg-transparent"
                tabIndex={isModalOpen ? 0 : -1}
              />
            </div>

            {createTeamError && (
              <div role="alert" className="mb-3 p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-600">
                {createTeamError}
              </div>
            )}

            <button
              onClick={handleCreateTeam}
              disabled={isCreatingTeam}
              className="w-full bg-[#0E7861] text-white font-bold py-3.5 rounded-xl flex items-center justify-center gap-2 active:bg-[#0A5A47] transition shadow-md cursor-pointer disabled:opacity-50"
              tabIndex={isModalOpen ? 0 : -1}
            >
              <Plus size={20} />
              {isCreatingTeam ? '作成中...' : roomCode ? 'もう一度作成' : 'チームを作成'}
            </button>

            {/* 作成できたら、招待用のQRと部屋コードを出す */}
            {roomCode && (
              <div className="mt-4 bg-white border border-gray-200 rounded-2xl p-6 shadow-sm flex flex-col items-center justify-center text-center">
                {inviteQrCode ? (
                  <img src={inviteQrCode} alt="招待用QRコード" className="w-48 h-48 mb-3" />
                ) : qrCodeError ? (
                  <p role="alert" className="text-sm text-red-600 mb-3">
                    QRコードを生成できませんでした: {qrCodeError}
                  </p>
                ) : (
                  <div className="w-48 h-48 mb-3 flex items-center justify-center text-[#0E7861]">
                    <QrCode size={40} aria-hidden="true" />
                  </div>
                )}
                <p className="text-xs text-gray-500 mb-1">このQRを見せて招待しましょう</p>
                <p className="text-lg font-bold tracking-widest text-gray-900">{roomCode}</p>
              </div>
            )}
          </div>

          <div>
            <h3 className="font-bold text-gray-900 mb-3">招待された方はこちら</h3>

            {/* 読み取りはOS標準カメラでよい（TECH_PLAN 7.1）のでアプリ内スキャナは置かない */}
            <div className="bg-white border border-gray-200 rounded-2xl p-5 shadow-sm">
              <ol className="text-sm text-gray-600 space-y-2 list-decimal list-inside">
                <li>チームを作った人にQRコードを見せてもらう</li>
                <li>スマホの<span className="font-bold text-gray-900">標準のカメラ</span>でそのまま読み取る</li>
                <li>開いたリンクからそのまま参加できます</li>
              </ol>
            </div>
            <p className="text-xs text-gray-400 mt-4 text-center">
              招待リンクを直接開いても参加できます。
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}