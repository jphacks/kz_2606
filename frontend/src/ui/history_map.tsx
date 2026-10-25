import { ArrowLeft, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import MapView from './map';
import { deleteActivity, getRoomHistory } from '../lib/api';
import { TrackBuffer } from '../lib/track_buffer';
import type { ActivityRecord, RoomHistory, RoomRouteTrack } from '../lib/types';

interface HistoryMapProps {
  activity: ActivityRecord;
  onBack: () => void;
  onDeleted: () => void;
}

export default function HistoryMap({ activity, onBack, onDeleted }: HistoryMapProps) {
  const [history, setHistory] = useState<RoomHistory | null>(null);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    let active = true;
    setIsLoading(true);
    setError('');
    getRoomHistory(activity.roomCode)
      .then((value) => { if (active) setHistory(value); })
      .catch((reason: unknown) => {
        if (active) setError(reason instanceof Error ? reason.message : 'ルームの履歴を取得できませんでした');
      })
      .finally(() => { if (active) setIsLoading(false); });
    return () => { active = false; };
  }, [activity.roomCode]);

  const tracks = useMemo<RoomRouteTrack[]>(() => {
    if (!history) return [];
    return history.routes.map((route) => {
      const buffer = new TrackBuffer();
      route.points.forEach((point) => buffer.add(point));
      buffer.finalize();
      return {
        routeId: route.id,
        memberId: route.memberId,
        memberName: route.memberName,
        color: route.color,
        active: false,
        lastPoint: buffer.lastCoordinate ?? undefined,
        buffer,
      };
    });
  }, [history]);

  const firstPoint = history?.routes.flatMap((route) => route.points)[0];
  const position: [number, number] = firstPoint ? [firstPoint[0], firstPoint[1]] : [36.578, 136.648];

  const handleDelete = async () => {
    if (!window.confirm(`「${activity.roomTitle}」のこの記録を削除しますか？`)) return;
    setIsDeleting(true);
    setError('');
    try {
      await deleteActivity(activity);
      onDeleted();
    } catch (reason: unknown) {
      setError(reason instanceof Error ? reason.message : '記録を削除できませんでした');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="relative h-full min-h-screen bg-[#F5F5F5]">
      <div className="absolute inset-0">
        <MapView position={position} hasLocation={false} otherTracks={tracks} />
      </div>
      <header className="absolute left-4 right-4 top-4 z-[1000] flex items-center gap-3 rounded-2xl bg-white/95 px-3 py-2 shadow-lg backdrop-blur-md">
        <button type="button" onClick={onBack} className="rounded-full p-2 text-gray-700 hover:bg-gray-100" aria-label="活動履歴に戻る">
          <ArrowLeft size={22} />
        </button>
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-sm font-bold text-gray-900">{activity.roomTitle}</h1>
          <p className="text-xs text-gray-500">メンバー全員の軌跡・{history?.routes.length ?? 0}件</p>
        </div>
        <button
          type="button"
          onClick={handleDelete}
          disabled={isDeleting}
          className="flex items-center gap-1 rounded-xl px-2 py-2 text-xs font-bold text-red-600 hover:bg-red-50 disabled:opacity-50"
        >
          <Trash2 size={16} />
          {isDeleting ? '削除中' : '削除'}
        </button>
      </header>
      {(isLoading || error) && (
        <div className="absolute bottom-5 left-4 right-4 z-[1000] rounded-2xl bg-white/95 p-4 text-sm shadow-lg">
          {isLoading && <p className="text-gray-600">軌跡を読み込んでいます...</p>}
          {error && <p role="alert" className="text-red-600">{error}</p>}
        </div>
      )}
    </div>
  );
}
