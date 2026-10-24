import { useState } from 'react';
import { Waypoints, Footprints, Flag, Link as LinkIcon, Plus, X, History, LogIn, UserPlus } from 'lucide-react';

interface HomeProps {
  onNavigateToLogin?: () => void;
  onNavigateToRegister?: () => void;
  onStartSolo?: () => void;
  username?: string;
}

export default function Home({ onNavigateToLogin, onNavigateToRegister, onStartSolo, username }: HomeProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);

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
            <button
              onClick={onStartSolo}
              className="bg-white border border-gray-100 p-5 rounded-2xl shadow-sm flex flex-col items-start gap-4 active:bg-gray-50 transition cursor-pointer"
            >
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
                placeholder="まちあるき部" 
                className="w-full outline-none text-gray-800 placeholder-gray-400 bg-transparent"
                tabIndex={isModalOpen ? 0 : -1}
              />
            </div>
            <button 
              className="w-full bg-[#0E7861] text-white font-bold py-3.5 rounded-xl flex items-center justify-center gap-2 active:bg-[#0A5A47] transition shadow-md cursor-pointer"
              tabIndex={isModalOpen ? 0 : -1}
            >
              <Plus size={20} />
              チームを作成
            </button>
          </div>

          <div>
            <h3 className="font-bold text-gray-900 mb-3">招待された方はこちら</h3>
            <div className="bg-white border border-gray-200 rounded-xl flex items-center p-3 mb-3 shadow-sm focus-within:border-[#0E7861] transition">
              <LinkIcon size={20} className="text-gray-400 mr-3" />
              <input 
                type="text" 
                placeholder="招待URLを貼り付け" 
                className="w-full outline-none text-gray-800 placeholder-gray-400 bg-transparent"
                tabIndex={isModalOpen ? 0 : -1}
              />
              <button 
                className="text-[#0E7861] text-sm font-bold whitespace-nowrap ml-2 cursor-pointer"
                tabIndex={isModalOpen ? 0 : -1}
              >
                貼り付け
              </button>
            </div>
            <button 
              className="w-full bg-white border border-[#0E7861] text-[#0E7861] font-bold py-3.5 rounded-xl flex items-center justify-center gap-2 active:bg-[#EBF5F3] transition cursor-pointer"
              tabIndex={isModalOpen ? 0 : -1}
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M5 12h14"></path>
                <path d="m12 5 7 7-7 7"></path>
              </svg>
              招待URLから参加
            </button>
            <p className="text-xs text-gray-400 mt-4 text-center">届いた招待リンクを直接開いても参加できます</p>
          </div>
        </div>
      </div>
    </div>
  );
}