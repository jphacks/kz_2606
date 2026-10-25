import type { Point, RoomMember, RoomRouteStats, RoomSnapshot } from './types';

/** 1通あたりの点数の上限。サーバーの maxMessageSize 64KB と連動（仕様 11章） */
export const maxPointsPerMessage = 500;
/** 点をまとめて送る間隔。stats の更新頻度でもある（仕様 11章 FLUSH_MS） */
export const flushIntervalMilliseconds = 3_000;

export interface ParsedRoomSnapshot {
  snapshot: RoomSnapshot;
  routePoints: Map<number, Point[]>;
}

export interface RouteStartedEvent {
  id: number;
  member: number;
  color: string;
  startedAt: string;
}

export function connectToRoom(roomCode: string): WebSocket {
  const url = new URL('/ws', window.location.href);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  url.searchParams.set('room', roomCode);
  return new WebSocket(url);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

export function parseRoomMember(value: unknown): RoomMember | null {
  if (
    !isRecord(value) ||
    typeof value.id !== 'number' ||
    typeof value.displayName !== 'string' ||
    typeof value.color !== 'string' ||
    typeof value.online !== 'boolean' ||
    typeof value.banned !== 'boolean' ||
    typeof value.recording !== 'boolean'
  ) {
    return null;
  }

  return {
    id: value.id,
    displayName: value.displayName,
    color: value.color,
    online: value.online,
    banned: value.banned,
    recording: value.recording,
  };
}

export function parsePoint(value: unknown): Point | null {
  if (
    !Array.isArray(value) ||
    value.length !== 4 ||
    !value.every((part) => typeof part === 'number' && Number.isFinite(part))
  ) {
    return null;
  }
  return [value[0], value[1], value[2], value[3]];
}

export function parseRoomSnapshot(value: unknown): ParsedRoomSnapshot | null {
  if (
    !isRecord(value) ||
    value.t !== 'snapshot' ||
    !isRecord(value.room) ||
    typeof value.room.code !== 'string' ||
    typeof value.room.title !== 'string' ||
    (value.room.closedAt !== null && typeof value.room.closedAt !== 'string') ||
    !isRecord(value.you) ||
    typeof value.you.id !== 'number' ||
    !Array.isArray(value.members) ||
    !Array.isArray(value.routes)
  ) {
    return null;
  }

  const members = value.members.map(parseRoomMember);
  if (members.some((member) => member === null)) {
    return null;
  }

  const routes: RoomRouteStats[] = [];
  const routePoints = new Map<number, Point[]>();
  for (const route of value.routes) {
    if (
      !isRecord(route) ||
      typeof route.id !== 'number' ||
      typeof route.member !== 'number' ||
      typeof route.color !== 'string' ||
      typeof route.startedAt !== 'string' ||
      !('endedAt' in route) ||
      (route.endedAt !== null && typeof route.endedAt !== 'string') ||
      typeof route.meters !== 'number' ||
      typeof route.elapsedMs !== 'number' ||
      !Array.isArray(route.pts)
    ) {
      return null;
    }
    const points = route.pts.map(parsePoint);
    if (points.some((point) => point === null)) return null;
    routePoints.set(route.id, points.filter((point): point is Point => point !== null));
    routes.push({
      id: route.id,
      member: route.member,
      color: route.color,
      startedAt: route.startedAt,
      endedAt: route.endedAt,
      meters: route.meters,
      elapsedMs: route.elapsedMs,
    });
  }

  return {
    snapshot: {
      room: { code: value.room.code, title: value.room.title, closedAt: value.room.closedAt },
      you: { id: value.you.id },
      members: members.filter((member): member is RoomMember => member !== null),
      routes,
    },
    routePoints,
  };
}

export function parseRouteStartedEvent(value: unknown): RouteStartedEvent | null {
  if (
    !isRecord(value) ||
    !isRecord(value.route) ||
    typeof value.route.id !== 'number' ||
    typeof value.route.member !== 'number' ||
    typeof value.route.color !== 'string' ||
    typeof value.route.startedAt !== 'string'
  ) {
    return null;
  }
  return {
    id: value.route.id,
    member: value.route.member,
    color: value.route.color,
    startedAt: value.route.startedAt,
  };
}

export function parsePointsEvent(value: unknown): { routeId: number; points: Point[] } | null {
  if (!isRecord(value) || typeof value.route !== 'number' || !Array.isArray(value.pts)) {
    return null;
  }
  const points = value.pts.map(parsePoint);
  if (points.length === 0 || points.some((point) => point === null)) return null;
  return {
    routeId: value.route,
    points: points.filter((point): point is Point => point !== null),
  };
}

export function parseRouteEndEvent(value: unknown): { routeId: number; memberId: number; endedAt: string } | null {
  if (
    !isRecord(value) ||
    typeof value.route !== 'number' ||
    typeof value.member !== 'number' ||
    typeof value.endedAt !== 'string'
  ) {
    return null;
  }
  return { routeId: value.route, memberId: value.member, endedAt: value.endedAt };
}

export function sendStart(socket: WebSocket): void {
  socket.send(JSON.stringify({ t: 'start' }));
}

/** 点をまとめて送る。上限を超える分は自動で分割する */
export function sendPoints(socket: WebSocket, points: Point[]): void {
  for (let index = 0; index < points.length; index += maxPointsPerMessage) {
    const chunk = points.slice(index, index + maxPointsPerMessage);
    socket.send(JSON.stringify({ t: 'points', pts: chunk }));
  }
}

export function sendStop(socket: WebSocket): void {
  socket.send(JSON.stringify({ t: 'stop' }));
}

/** メンバー情報を更新する。サーバー側で色を正規化・永続化し、全員へ通知する。 */
export function sendMember(socket: WebSocket, color: string): void {
  socket.send(JSON.stringify({ t: 'member', color }));
}
