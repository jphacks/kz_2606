import { useState } from 'react';
import { Settings, Users, Copy, Check, MapPin, Play } from 'lucide-react';

interface BeforeRecordProps {
  hasLocation?: boolean;
  onStartRecording?: () => void;
  onOpenSetting?: () => void;
}

export default function BeforeRecord({ hasLocation, onStartRecording, onOpenSetting }: BeforeRecordProps) {
  const [copied, setCopied] = useState(false);

  const handleCopyInvite = async () => {
    try {
      await navigator.clipboard.writeText("https://machie.app/invite/sample");
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.warn("クリップボードへのコピーに失敗しました:", err);
    }
  };

  return (
    <div className="w-full h-full flex flex-col justify-between pointer-events-none">
      
      {/* 1. 上部のヘッダー ＆ 設定ボタン（ボタン部分は操作可能に） */}
      <div className="w-full flex items-center justify-between pointer-events-auto">
        <div className="bg-white/90 backdrop-blur-md px-3.5 py-1.5 rounded-full shadow-md flex items-center gap-2 border border-gray-100">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
          <span className="text-xs font-bold text-gray-800">まちあるき部</span>
          <span className="text-gray-300">|</span>
          <span className="text-xs text-gray-500">吉祥寺</span>
        </div>

        <button 
          onClick={onOpenSetting}
          className="bg-white/90 backdrop-blur-md px-3.5 py-1.5 rounded-full shadow-md text-gray-700 hover:bg-white transition flex items-center gap-1.5 border border-gray-100 font-bold text-xs"
        >
          <Settings size={14} />
          <span>設定</span>
        </button>
      </div>

      {/* 2. 下部の白いカードエリア（高さをスリムにしてコンパクトに調整） */}
      <div className="w-full bg-white rounded-t-[24px] p-4 shadow-2xl border-t border-gray-100 flex flex-col gap-3 pointer-events-auto max-h-[52vh] overflow-y-auto">
        
        {/* キャンバスバッジ ＆ タイトル */}
        <div>
          <div className="flex items-center justify-between mb-0.5">
            <span className="text-[10px] font-extrabold tracking-wider text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full uppercase">
              TODAY'S CANVAS
            </span>
            <span className="text-[10px] font-bold text-gray-400 bg-gray-50 px-2 py-0.5 rounded-md">
              記録前
            </span>
          </div>
          <h2 className="text-lg font-black text-gray-900">
            吉祥寺にハートを描こう
          </h2>
          <p className="text-[11px] text-gray-500 leading-snug">
            4つの色で、ひとつのハート。担当のルートを歩いて、軌跡をつなげよう。
          </p>
        </div>

        {/* 参加メンバー一覧 */}
        <div className="flex items-center justify-between py-1.5 border-y border-gray-100">
          <div className="flex items-center -space-x-2">
            <div className="w-7 h-7 rounded-full bg-emerald-700 text-white flex items-center justify-center text-[11px] font-bold border-2 border-white shadow-sm">さ</div>
            <div className="w-7 h-7 rounded-full bg-amber-600 text-white flex items-center justify-center text-[11px] font-bold border-2 border-white shadow-sm">ゆ</div>
            <div className="w-7 h-7 rounded-full bg-blue-600 text-white flex items-center justify-center text-[11px] font-bold border-2 border-white shadow-sm">は</div>
            <div className="w-7 h-7 rounded-full bg-pink-600 text-white flex items-center justify-center text-[11px] font-bold border-2 border-white shadow-sm">り</div>
          </div>
          <span className="text-[11px] font-bold text-gray-600 flex items-center gap-1">
            <Users size={13} className="text-gray-400" /> 4人が参加済み
          </span>
        </div>

        {/* 招待URLシェアボックス */}
        <div className="bg-gray-50 p-2.5 rounded-xl border border-gray-100 flex items-center justify-between">
          <div className="flex items-center gap-2 truncate">
            <div className="w-6 h-6 rounded-lg bg-white flex items-center justify-center shadow-xs text-emerald-700 shrink-0">
              <Copy size={12} />
            </div>
            <div className="truncate">
              <p className="text-[9px] font-bold text-gray-400 leading-none">招待URLをシェア</p>
              <p className="text-[11px] font-medium text-gray-700 truncate mt-0.5">machie.app/t/machiaruki</p>
            </div>
          </div>
          <button 
            onClick={handleCopyInvite}
            className="px-2.5 py-1 bg-white border border-gray-200 rounded-lg text-[11px] font-bold text-gray-700 hover:bg-gray-100 transition shadow-xs flex items-center gap-1 shrink-0"
          >
            {copied ? <Check size={11} className="text-emerald-600" /> : null}
            {copied ? 'コピー済' : 'コピー'}
          </button>
        </div>

        {/* 位置情報のステータス */}
        <div className="flex items-center gap-1 text-[11px] text-emerald-600 font-medium px-0.5">
          <MapPin size={13} />
          <span>{hasLocation ? '位置情報は利用可能です' : '位置情報を取得中...'}</span>
        </div>

        {/* 記録開始ボタン */}
        <button
          onClick={onStartRecording}
          className="w-full py-3 bg-[#0E7861] hover:bg-[#0b5f4d] active:scale-[0.98] transition text-white rounded-xl font-bold text-sm shadow-md shadow-emerald-900/20 flex items-center justify-center gap-2"
        >
          <Play size={16} fill="white" />
          <span>記録開始</span>
        </button>

        {/* 注意書き */}
        <p className="text-[9px] text-center text-gray-400 leading-normal px-1">
          開始すると自分の現在地と軌跡がチームに共有されます。周囲の安全を確かめて、歩きながら楽しもう。
        </p>

      </div>
    </div>
  );
}