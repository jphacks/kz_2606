import {
  ArrowLeft,
  CheckCircle2,
  Clock,
  Footprints,
  History,
  Home,
  Image as ImageIcon,
  Lock,
  RefreshCw,
  Trash2,
  Users,
  Waypoints,
} from 'lucide-react';
import type { ActivityHistory, ActivityRecord } from '../lib/types';

interface ActivityProps {
  username?: string;
  history: ActivityHistory | null;
  isLoading?: boolean;
  error?: string;
  isRecording?: boolean;
  isConnecting?: boolean;
  onRetry?: () => void;
  onBackToHome?: () => void;
  onSelectActivity?: (activity: ActivityRecord) => void;
  onDeleteActivity?: (activity: ActivityRecord) => void;
}

function formatDistance(meters: number): string {
  return meters >= 1000 ? `${(meters / 1000).toFixed(2)} km` : `${Math.round(meters)} m`;
}

function formatElapsed(milliseconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}時間${minutes}分`;
  if (minutes > 0) return `${minutes}分${seconds}秒`;
  return `${seconds}秒`;
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('ja-JP', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}

function ActivityDetails({ activity }: { activity: ActivityRecord }) {
  return (
    <>
      <div className="flex items-center gap-2 mb-1">
        <span className="text-xs text-gray-500">{formatDate(activity.startedAt)}</span>
        <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${
          activity.endedAt ? 'text-[#0e7861] bg-[#EBF5F3]' : 'text-amber-700 bg-amber-50'
        }`}>
          {activity.endedAt ? '完了' : '記録中'}
        </span>
        {activity.isPrivate && <Lock size={12} className="text-gray-400" aria-label="個人ルーム" />}
      </div>
      <h4 className="font-bold text-gray-900 text-sm truncate mb-1">{activity.roomTitle}</h4>
      <p className="text-xs text-gray-500 truncate">
        {formatDistance(activity.meters)}・{formatElapsed(activity.elapsedMs)}・{activity.memberCount}人
      </p>
    </>
  );
}

export default function Activity({
  username,
  history,
  isLoading = false,
  error = '',
  isRecording = false,
  isConnecting = false,
  onRetry,
  onBackToHome,
  onSelectActivity,
  onDeleteActivity,
}: ActivityProps) {
  const displayUsername = username?.trim() || null;
  const summary = history?.summary;
  const latest = history?.activities[0];

  return (
    <div
      className="min-h-screen bg-[#F5F5F5] font-sans text-gray-800 relative flex flex-col"
      style={{ colorScheme: 'light' }}
    >
      <header className="flex items-center justify-between px-5 pb-4 pt-[max(1.25rem,env(safe-area-inset-top))] bg-white sticky top-0 z-10 shadow-sm">
        <button
          onClick={onBackToHome}
          className="p-2 -ml-2 text-gray-700 hover:bg-gray-100 rounded-full transition"
          aria-label="ホームに戻る"
        >
          <ArrowLeft size={24} />
        </button>
        <div className="flex-1 ml-2">
          <h1 className="text-lg font-bold text-gray-900 leading-tight">これまでの活動</h1>
          <p className="text-xs text-gray-500">
            {displayUsername ? `${displayUsername}（あなた）の記録` : 'あなたの記録'}
          </p>
        </div>
        <div className="w-10 h-10 bg-[#EBF5F3] text-[#0e7861] rounded-2xl flex items-center justify-center font-bold text-lg">
          {displayUsername?.charAt(0) ?? <Footprints size={20} />}
        </div>
      </header>

      {(isRecording || isConnecting) && (
        <div role="status" className="flex items-center gap-2 bg-emerald-50 px-5 py-2 text-xs font-bold text-emerald-800">
          <span className="h-2 w-2 rounded-full bg-emerald-600 animate-pulse" />
          {isConnecting ? '記録を接続中です' : '記録中です。最新値は再読み込みすると反映されます'}
        </div>
      )}

      <main className="px-5 pt-6 pb-8" aria-busy={isLoading}>
        <div className="mb-4">
          <h2 className="text-2xl font-extrabold text-gray-900 mb-1">一歩ずつ、思い出に。</h2>
          <p className="text-xs text-gray-500">保存されたGPS記録を集計しています。</p>
        </div>

        {error && (
          <div role="alert" className="mb-4 rounded-2xl border border-red-100 bg-red-50 p-4 text-sm text-red-700">
            <p>{error}</p>
            {onRetry && username && (
              <button onClick={onRetry} className="mt-3 flex items-center gap-1 font-bold text-red-800">
                <RefreshCw size={14} />
                再読み込み
              </button>
            )}
          </div>
        )}

        <div className="grid grid-cols-2 gap-3 mb-3" aria-live="polite">
          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <History size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">記録した回数</span>
            </div>
            <div className="text-2xl font-bold text-gray-900">{isLoading ? '—' : `${summary?.recordings ?? 0}回`}</div>
          </div>

          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <ImageIcon size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">完了した記録</span>
            </div>
            <div className="text-2xl font-bold text-gray-900">{isLoading ? '—' : `${summary?.completed ?? 0}件`}</div>
          </div>

          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <Waypoints size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">歩いた距離</span>
            </div>
            <div className="text-2xl font-bold text-gray-900">{isLoading ? '—' : formatDistance(summary?.meters ?? 0)}</div>
          </div>

          <div className="bg-white p-4 rounded-2xl border border-gray-100 shadow-sm">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <Clock size={16} className="text-[#0e7861]" />
              <span className="text-xs font-bold">活動時間</span>
            </div>
            <div className="text-2xl font-bold text-gray-900">{isLoading ? '—' : formatElapsed(summary?.elapsedMs ?? 0)}</div>
          </div>
        </div>
        <p className="text-[10px] text-gray-400 mb-8">距離と時間は、あなた自身のrouteの合計です。</p>

        {latest && (
          <>
            <h3 className="text-lg font-bold text-gray-900 mb-4">最新の記録</h3>
            <div className="bg-white rounded-3xl p-5 shadow-sm border border-gray-100 mb-8">
              <div className="flex items-start gap-4">
                <div className="w-12 h-12 bg-[#EBF5F3] text-[#0e7861] rounded-2xl flex items-center justify-center shrink-0">
                  <CheckCircle2 size={24} />
                </div>
                <div className="min-w-0 flex-1">
                  <ActivityDetails activity={latest} />
                </div>
              </div>
            </div>
          </>
        )}
      </main>

      <section className="bg-white flex-1 px-5 pt-8 pb-[max(2rem,env(safe-area-inset-bottom))] rounded-t-3xl shadow-[0_-4px_20px_rgba(0,0,0,0.03)]">
        <div className="flex justify-between items-end mb-6">
          <h3 className="text-lg font-bold text-gray-900">活動履歴</h3>
          <span className="text-xs text-gray-500">
            {isLoading ? '読み込み中' : `${history?.activities.length ?? 0}件・新しい順`}
          </span>
        </div>

        {!isLoading && !error && history?.activities.length === 0 && (
          <div className="rounded-2xl bg-gray-50 px-4 py-8 text-center text-sm text-gray-500 mb-8">
            <Footprints size={28} className="mx-auto mb-2 text-gray-400" />
            まだ記録がありません。最初のGPSアートを始めましょう。
          </div>
        )}

        <div className="space-y-4 mb-8">
          {history?.activities.map((activity) => (
            <div key={activity.routeId} className="w-full flex items-center gap-3 text-left">
              <button
                type="button"
                onClick={() => onSelectActivity?.(activity)}
                className="flex min-w-0 flex-1 items-center gap-4 rounded-2xl text-left hover:bg-gray-50"
              >
              <div className="w-12 h-12 bg-[#EBF5F3] text-[#0e7861] rounded-2xl flex items-center justify-center shrink-0">
                {activity.memberCount > 1 ? <Users size={23} /> : <Footprints size={23} />}
              </div>
              <div className="flex-1 overflow-hidden">
                <ActivityDetails activity={activity} />
              </div>
              </button>
              {onDeleteActivity && (
                <button
                  type="button"
                  onClick={() => onDeleteActivity(activity)}
                  className="rounded-xl p-2 text-gray-400 hover:bg-red-50 hover:text-red-600"
                  aria-label={`${activity.roomTitle}の記録を削除`}
                >
                  <Trash2 size={18} />
                </button>
              )}
            </div>
          ))}
        </div>

        <button
          onClick={onBackToHome}
          className="w-full flex items-center justify-center gap-2 text-[#0e7861] font-bold py-3 hover:bg-[#EBF5F3] rounded-xl transition"
        >
          <Home size={18} />
          ホームに戻る
        </button>
      </section>
    </div>
  );
}
