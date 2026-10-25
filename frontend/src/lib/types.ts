// サーバーと共有する型。Point は [lat, lng, ts, acc] のタプル。
// 仕様: DATA_MODEL.md 3章・5章
//
// TODO: Point / Member / ServerMsg / ClientMsg / PALETTE

export interface User {
  id: number;
  username: string;
}

export interface Room {
  id: number;
  code: string;
  title: string;
  ownerMemberId: number;
  isPrivate: boolean;
  closedAt: string | null;
}

export interface AuthSession {
  user: User;
  room: Room;
}

export interface ActivityRecord {
  routeId: number;
  roomCode: string;
  roomTitle: string;
  isPrivate: boolean;
  memberCount: number;
  startedAt: string;
  endedAt: string | null;
  meters: number;
  elapsedMs: number;
}

export interface ActivityHistory {
  summary: {
    recordings: number;
    completed: number;
    meters: number;
    elapsedMs: number;
  };
  activities: ActivityRecord[];
}

export interface HistoryRoute {
  id: number;
  memberId: number;
  memberName: string;
  color: string;
  startedAt: string;
  endedAt: string | null;
  points: Point[];
}

export interface RoomHistory {
  roomCode: string;
  roomTitle: string;
  closedAt: string | null;
  routes: HistoryRoute[];
}

export interface RoomMember {
  id: number;
  displayName: string;
  color: string;
  online: boolean;
  banned: boolean;
  recording: boolean;
}

export interface RoomRouteStats {
  id: number;
  member: number;
  color: string;
  startedAt: string;
  endedAt: string | null;
  meters: number;
  elapsedMs: number;
}

export interface RoomSnapshot {
  room: Pick<Room, 'code' | 'title' | 'closedAt' | 'ownerMemberId'> & {
    recordingEnabled: boolean;
  };
  you: Pick<RoomMember, 'id'>;
  members: RoomMember[];
  routes: RoomRouteStats[];
}

export type Point = [lat: number, lng: number, ts: number, acc: number];

export interface RoomRouteTrack {
  routeId: number;
  memberId: number;
  memberName?: string;
  color: string;
  active: boolean;
  lastPoint?: [number, number];
  buffer: import('./track_buffer').TrackBuffer;
}
