import type { Point, RoomMember, RoomRouteStats, RoomSnapshot } from './types';

/** 1通あたりの点数の上限。サーバーの maxMessageSize 64KB と連動（仕様 11章） */
export const maxPointsPerMessage = 500;
/** 点をまとめて送る間隔。stats の更新頻度でもある（仕様 11章 FLUSH_MS） */
export const flushIntervalMilliseconds = 3_000;

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

export function parseRoomSnapshot(value: unknown): RoomSnapshot | null {
  if (
    !isRecord(value) ||
    value.t !== 'snapshot' ||
    !isRecord(value.room) ||
    typeof value.room.code !== 'string' ||
    typeof value.room.title !== 'string' ||
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
  for (const route of value.routes) {
    if (
      !isRecord(route) ||
      typeof route.id !== 'number' ||
      typeof route.member !== 'number' ||
      typeof route.meters !== 'number' ||
      typeof route.elapsedMs !== 'number'
    ) {
      return null;
    }
    routes.push({
      id: route.id,
      member: route.member,
      meters: route.meters,
      elapsedMs: route.elapsedMs,
    });
  }

  return {
    room: { code: value.room.code, title: value.room.title },
    you: { id: value.you.id },
    members: members.filter((member): member is RoomMember => member !== null),
    routes,
  };
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
