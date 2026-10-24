import { 
  ChevronLeft, 
  ChevronDown, 
  Footprints, 
  Image as ImageIcon, 
  Waypoints, 
  Clock, 
  CheckCircle2, 
  ArrowRight, 
  ChevronRight, 
  Users, 
  Home,
  Heart,
  Circle
} from 'lucide-react';

export default function Activity() {
  return (
    <div 
      className="min-h-screen bg-[#F5F5F5] font-sans text-gray-800 relative flex flex-col"
      style={{ colorScheme: 'light' }}
    >
      {/* --- ヘッダー --- */}
      <header className="flex items-center justify-between px-5 pb-4 pt-[max(1.25rem,env(safe-area-inset-top))] bg-white sticky top-0 z-10 shadow-sm">
        <button className="p-2 -ml-2 text-gray-700 hover:bg-gray-100 rounded-full transition" aria-label="戻る">
          <ChevronLeft size={24} />
        </button>
        <div className="flex-1 ml-2">
          <h1 className="text-lg font-bold text-gray-900 leading-tight">これまでの活動</h1>
          <p className="text-xs text-gray-500">さき（あなた） の記録</p>
        </div>
        <div className="w-10 h-10 bg-[#EBF5F3] text-[#0e7861] rounded-2xl flex items-center justify-center font-bold text-lg">
          さ
        </div>
      </header>

      {/* --- サマリーセクション --- */}
      <main className="px-5 pt-6 pb-8">
        <div className="flex justify-between items-end mb-4">
          <div>
            <h2 className="text-2xl font-extrabold text-gray-900 mb-1">一歩ずつ、思い出に。</h2>
            <p className="text-xs text-gray-500">みんなと歩いて、描いてきた記録です。</p>
          </div>
          <button className="flex items-center gap-1 bg-white border border-gray-200 px-3 py-1.5 rounded-full text-sm font-medium text-gray-700 shadow-sm">
            全期間
            <ChevronDown size={16} className="text-gray-500" />
          </button>
        </div>

        {/* 4つの統計パネル */}
        <div className="grid grid-cols-2 gap-3 mb-3">
          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <Footprints size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">参加した回数</span>
            </div>
            <div className="flex items-baseline gap-1">
              <span className="text-3xl font-bold text-gray-900">3</span>
              <span className="text-sm text-gray-600 font-medium">回</span>
            </div>
          </div>
          
          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <ImageIcon size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">完成したアート</span>
            </div>
            <div className="flex items-baseline gap-1">
              <span className="text-3xl font-bold text-gray-900">3</span>
              <span className="text-sm text-gray-600 font-medium">作品</span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <Waypoints size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">歩いた距離</span>
            </div>
            <div className="flex items-baseline gap-1">
              <span className="text-3xl font-bold text-gray-900">4.80</span>
              <span className="text-sm text-gray-600 font-medium">km</span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <Clock size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">活動時間</span>
            </div>
            <div className="flex items-baseline gap-1">
              <span className="text-3xl font-bold text-gray-900">2</span>
              <span className="text-xs text-gray-600 font-medium">時間 24分</span>
            </div>
          </div>
        </div>
        <p className="text-[10px] text-gray-400 mb-8">距離と時間は、あなた自身の記録の合計です。</p>

        {/* 最近完成したアート */}
        <h3 className="text-lg font-bold text-gray-900 mb-4">最近完成したアート</h3>
        <div className="bg-white rounded-3xl overflow-hidden shadow-sm border border-gray-100">
          <div className="relative h-40 bg-gray-100">
            <img 
              src="/hero-map.png" 
              alt="吉祥寺にハートを描こう" 
              className="w-full h-full object-cover"
            />
            <div className="absolute top-3 left-3 bg-white/95 text-[#0e7861] px-3 py-1.5 rounded-full text-xs font-bold flex items-center gap-1.5 shadow-sm backdrop-blur-sm">
              <CheckCircle2 size={16} />
              4人で完成！
            </div>
          </div>
          <div className="p-5">
            <p className="text-xs text-gray-500 mb-1">まちあるき部・2026.10.24</p>
            <h4 className="text-xl font-bold text-gray-900 mb-3">吉祥寺にハートを描こう</h4>
            <div className="flex justify-between items-center text-sm font-medium mb-4">
              <span className="text-gray-600">みんなの距離 <span className="text-gray-900">6.00 km</span></span>
              <span className="text-gray-600">経過時間 <span className="text-gray-900">48:32</span></span>
            </div>
            <hr className="border-gray-100 mb-4" />
            <button className="w-full flex justify-between items-center text-[#0e7861] font-bold text-sm">
              完成アートと結果を見る
              <ArrowRight size={18} />
            </button>
          </div>
        </div>
      </main>

      {/* --- 活動履歴リスト --- */}
      <section className="bg-white flex-1 px-5 pt-8 pb-[max(2rem,env(safe-area-inset-bottom))] rounded-t-3xl shadow-[0_-4px_20px_rgba(0,0,0,0.03)]">
        <div className="flex justify-between items-end mb-6">
          <div>
            <h3 className="text-lg font-bold text-gray-900 mb-2">活動履歴</h3>
            <div className="flex items-center gap-1 text-[#0e7861] text-xs font-bold bg-[#EBF5F3] px-2 py-1 rounded-md inline-flex">
              <Users size={14} />
              まちあるき部
            </div>
          </div>
          <span className="text-xs text-gray-500">全3件・新しい順</span>
        </div>

        <div className="space-y-4 mb-8">
          {/* 履歴アイテム 1 */}
          <button className="w-full flex items-center gap-4 text-left active:opacity-70 transition">
            <div className="w-12 h-12 bg-[#EBF5F3] text-[#0e7861] rounded-2xl flex items-center justify-center shrink-0">
              <Heart size={24} />
            </div>
            <div className="flex-1 overflow-hidden">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xs text-gray-500">2026.10.24</span>
                <span className="text-[10px] font-bold text-[#0e7861] bg-[#EBF5F3] px-1.5 py-0.5 rounded">完成</span>
              </div>
              <h4 className="font-bold text-gray-900 text-sm truncate mb-1">吉祥寺にハートを描こう</h4>
              <p className="text-xs text-gray-500 truncate">あなた 1.60 km・48分32秒・4人</p>
            </div>
            <ChevronRight size={20} className="text-gray-400 shrink-0" />
          </button>

          {/* 履歴アイテム 2 */}
          <button className="w-full flex items-center gap-4 text-left active:opacity-70 transition">
            <div className="w-12 h-12 bg-[#EBF5F3] text-[#0e7861] rounded-2xl flex items-center justify-center shrink-0">
              <Circle size={24} />
            </div>
            <div className="flex-1 overflow-hidden">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xs text-gray-500">2026.10.17</span>
                <span className="text-[10px] font-bold text-[#0e7861] bg-[#EBF5F3] px-1.5 py-0.5 rounded">完成</span>
              </div>
              <h4 className="font-bold text-gray-900 text-sm truncate mb-1">井の頭公園に輪を描こう</h4>
              <p className="text-xs text-gray-500 truncate">あなた 1.85 km・45分10秒・4人</p>
            </div>
            <ChevronRight size={20} className="text-gray-400 shrink-0" />
          </button>

          {/* 履歴アイテム 3 */}
          <button className="w-full flex items-center gap-4 text-left active:opacity-70 transition">
            <div className="w-12 h-12 bg-[#EBF5F3] text-[#0e7861] rounded-2xl flex items-center justify-center shrink-0">
              <Heart size={24} />
            </div>
            <div className="flex-1 overflow-hidden">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xs text-gray-500">2026.10.10</span>
                <span className="text-[10px] font-bold text-[#0e7861] bg-[#EBF5F3] px-1.5 py-0.5 rounded">完成</span>
              </div>
              <h4 className="font-bold text-gray-900 text-sm truncate mb-1">吉祥寺にハートを描こう</h4>
              <p className="text-xs text-gray-500 truncate">あなた 1.35 km・50分18秒・4人</p>
            </div>
            <ChevronRight size={20} className="text-gray-400 shrink-0" />
          </button>
        </div>

        {/* ホームに戻るボタン */}
        <button
          onClick={() => window.location.assign('/')}
          className="w-full flex items-center justify-center gap-2 text-[#0e7861] font-bold py-3 hover:bg-[#EBF5F3] rounded-xl transition"
        >
          <Home size={18} />
          ホームに戻る
        </button>
      </section>
    </div>
  );
}