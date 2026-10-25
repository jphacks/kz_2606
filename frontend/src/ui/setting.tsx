import { useState } from 'react';
import { Check, Copy, CheckCircle2, ArrowLeft } from 'lucide-react';

// メンバーの型定義
export interface Member {
  id: string;
  name: string;
  status: string;
  distance: string;
  elapsed?: string;
  color: string;
  isMe?: boolean; // 自分自身かどうか
}

interface SettingProps {
  teamName?: string;
  hasRoom?: boolean;
  memberCount?: number;
  members?: Member[];
  isMembersLoading?: boolean;
  inviteUrl?: string;
  initialColor?: string;
  onSaveAndReturn?: (selectedColor: string) => void;
  onBack?: () => void;
}

export default function Setting({
  teamName,
  hasRoom = false,
  memberCount,
  members = [],
  isMembersLoading = false,
  inviteUrl,
  initialColor = '#0E7861',
  onSaveAndReturn,
  onBack
}: SettingProps) {
  // 選択中のカラーを管理するステート（前回保存した色を初期値にする）
  const [selectedColor, setSelectedColor] = useState(initialColor);
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState('');

  // 10色（5色×2行）のカラーパレット
  const colors = [
    { name: 'グリーン', hex: '#0E7861' },
    { name: 'オレンジ', hex: '#E67E22' },
    { name: 'ブルー', hex: '#3498DB' },
    { name: 'ピンク', hex: '#E91E63' },
    { name: 'パープル', hex: '#9B59B6' },
    { name: 'イエロー', hex: '#F1C40F' },
    { name: 'レッド', hex: '#E74C3C' },
    { name: 'ターコイズ', hex: '#1ABC9C' },
    { name: 'インディゴ', hex: '#3F51B5' },
    { name: 'ライム', hex: '#8BC34A' },
  ];

  const handleCopyInvite = async () => {
    if (!inviteUrl) return;
    setCopyError('');
    try {
      await navigator.clipboard.writeText(inviteUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('招待リンクをコピーできませんでした:', error);
      setCopyError('招待リンクをコピーできませんでした');
    }
  };

  return (
    <div className="min-h-screen bg-[#FAFAFA] font-sans text-gray-800 relative pb-12 overflow-x-hidden flex flex-col">
      
      {/* --- ヘッダー部分 --- */}
      <header className="px-5 pt-[max(1.25rem,env(safe-area-inset-top))] pb-4 bg-white border-b border-gray-100 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button 
            onClick={onBack}
            className="text-gray-600 p-1 hover:bg-gray-100 rounded-full transition"
          >
            <ArrowLeft size={20} />
          </button>
          <div>
            <h1 className="text-lg font-bold text-gray-900">メンバーとカラー</h1>
            <p className="text-xs text-gray-500">
              {teamName ?? (isMembersLoading ? 'チーム情報を取得中' : hasRoom ? 'チーム情報を取得できません' : 'チーム未参加')}
              ・{isMembersLoading && memberCount === undefined
                ? '参加人数を取得中'
                : `${memberCount ?? members.length}人が参加中`}
            </p>
          </div>
        </div>
        <button 
          onClick={handleCopyInvite}
          disabled={!inviteUrl}
          className="p-2.5 bg-gray-50 border border-gray-200 rounded-full text-gray-600 hover:bg-gray-100 transition relative"
          title="招待リンクをコピー"
        >
          {copied ? <Check size={18} className="text-[#0E7861]" /> : <Copy size={18} />}
        </button>
      </header>
      {copyError && <p role="alert" className="px-5 pt-2 text-xs text-red-600">{copyError}</p>}

      {/* --- メインコンテンツ --- */}
      <main className="px-5 pt-4 flex-1">
        
        {/* 通知バナー */}
        <div className="bg-[#EBF5F3] text-[#0E7861] text-xs font-medium px-4 py-2.5 rounded-xl mb-5 flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-[#0E7861] animate-pulse"></span>
          記録中もカラーを変更できます
        </div>

        {/* 一緒に描いているメンバー一覧 */}
        <div className="bg-white border border-gray-200/80 rounded-2xl p-4 shadow-sm mb-6">
          <div className="flex justify-between items-center mb-3">
            <h2 className="text-sm font-bold text-gray-900">一緒に描いているメンバー</h2>
            <span className="text-xs text-gray-400">
              {isMembersLoading && memberCount === undefined
                ? '取得中'
                : `${members.length} / ${memberCount ?? members.length}人`}
            </span>
          </div>

          <div className="divide-y divide-gray-100">
            {members.map((member) => (
              <div key={member.id} className="py-3 flex items-center justify-between first:pt-0 last:pb-0">
                <div className="flex items-center gap-3">
                  <div
                    className="w-10 h-10 rounded-full text-white font-bold flex items-center justify-center text-sm shadow-sm"
                    style={{ backgroundColor: member.color }}
                  >
                    {member.name.charAt(0)}
                  </div>
                  <div>
                    <div className="flex items-center gap-1.5">
                      <span className="font-bold text-sm text-gray-900">{member.name}</span>
                      {member.isMe && (
                        <span className="text-[10px] bg-gray-100 text-gray-500 px-1.5 py-0.5 rounded-full font-medium">
                          あなた
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-gray-400 mt-0.5">
                      ● {member.status}・{member.distance}
                      {member.elapsed ? `・${member.elapsed}` : ''}
                    </p>
                  </div>
                </div>
                <div 
                  className="w-6 h-1.5 rounded-full" 
                  style={{ backgroundColor: member.color }}
                ></div>
              </div>
            ))}
          </div>
          {members.length === 0 && (
            <p className="py-4 text-center text-sm text-gray-400">
              {isMembersLoading
                ? 'メンバー情報を取得中です'
                : hasRoom
                  ? 'メンバー情報はありません'
                  : 'チームに参加していません'}
            </p>
          )}
        </div>

        {/* 自分の軌跡カラー選択 */}
        <div className="bg-white border border-gray-200/80 rounded-2xl p-4 shadow-sm mb-6">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-sm font-bold text-gray-900">自分の軌跡カラー</h2>
            <span className="text-xs font-bold px-2.5 py-1 bg-[#EBF5F3] text-[#0E7861] rounded-full">
              {colors.find(c => c.hex === selectedColor)?.name || 'カスタム'}
            </span>
          </div>

          {/* カラーパレット（5色×2行 = 10色） */}
          <div className="grid grid-cols-5 gap-3 mb-4">
            {colors.map((c) => {
              const isSelected = selectedColor === c.hex;
              return (
                <button
                  key={c.hex}
                  onClick={() => setSelectedColor(c.hex)}
                  className="aspect-square rounded-full flex items-center justify-center transition transform active:scale-95 shadow-sm relative mx-auto w-10 h-10"
                  style={{ backgroundColor: c.hex }}
                >
                  {isSelected && <Check size={16} className="text-white stroke-[3]" />}
                </button>
              );
            })}
          </div>

          {/* カスタムカラー入力 */}
          <div className="flex items-center justify-between bg-gray-50 border border-gray-200 rounded-xl px-3 py-2.5">
            <div className="flex items-center gap-2 text-gray-600 text-sm">
              <span className="text-xs font-bold text-gray-700">カスタムカラー</span>
            </div>
            <div className="flex items-center gap-2">
              <input 
                type="color" 
                value={selectedColor} 
                onChange={(e) => setSelectedColor(e.target.value)}
                className="w-6 h-6 rounded border-0 cursor-pointer bg-transparent"
              />
              <span className="text-xs font-mono text-gray-500 uppercase">{selectedColor}</span>
            </div>
          </div>

          <p className="text-[11px] text-gray-400 mt-3 leading-relaxed">
            変更は記録済みの軌跡にも反映され、全員のマップに共有されます。
          </p>
        </div>

      </main>

      {/* --- フッターボタン --- */}
      <div className="px-5 pt-2">
        <button 
          onClick={() => onSaveAndReturn && onSaveAndReturn(selectedColor)}
          className="w-full bg-[#0E7861] text-white font-bold py-4 rounded-2xl flex items-center justify-center gap-2 active:bg-[#0A5A47] transition shadow-lg shadow-[#0E7861]/20 text-sm"
        >
          <CheckCircle2 size={18} />
          カラーを保存してマップへ
        </button>
      </div>

    </div>
  );
}
