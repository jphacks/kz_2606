// REST 呼び出し。

import type { ActivityHistory, ActivityRecord, AuthSession, Point, Room, RoomHistory, RoomMember, User } from './types';

export function parseAuthUser(payload: unknown): User | null {
  if (typeof payload !== 'object' || payload === null || !('user' in payload)) {
    return null;
  }

  const user = payload.user;
  if (
    typeof user !== 'object' ||
    user === null ||
    !('id' in user) ||
    !('username' in user) ||
    typeof user.id !== 'number' ||
    typeof user.username !== 'string'
  ) {
    return null;
  }

  return { id: user.id, username: user.username };
}

export function getApiErrorMessage(payload: unknown, fallback: string): string {
  if (typeof payload !== 'object' || payload === null || !('error' in payload)) {
    return fallback;
  }

  const error = payload.error;
  if (
    typeof error !== 'object' ||
    error === null ||
    !('message' in error) ||
    typeof error.message !== 'string'
  ) {
    return fallback;
  }

  return error.message;
}

export async function getCurrentUser(): Promise<User | null> {
  const session = await getCurrentSession();
  return session?.user ?? null;
}

function isNonNegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0;
}

function parseActivityRecord(value: unknown): ActivityRecord | null {
  if (
    typeof value !== 'object' ||
    value === null ||
    !('routeId' in value) ||
    !('roomCode' in value) ||
    !('roomTitle' in value) ||
    !('isPrivate' in value) ||
    !('memberCount' in value) ||
    !('startedAt' in value) ||
    !('endedAt' in value) ||
    !('meters' in value) ||
    !('elapsedMs' in value) ||
    typeof value.routeId !== 'number' ||
    !Number.isSafeInteger(value.routeId) ||
    value.routeId <= 0 ||
    typeof value.roomCode !== 'string' ||
    typeof value.roomTitle !== 'string' ||
    typeof value.isPrivate !== 'boolean' ||
    typeof value.memberCount !== 'number' ||
    !Number.isSafeInteger(value.memberCount) ||
    value.memberCount < 1 ||
    typeof value.startedAt !== 'string' ||
    Number.isNaN(Date.parse(value.startedAt)) ||
    (value.endedAt !== null && (typeof value.endedAt !== 'string' || Number.isNaN(Date.parse(value.endedAt)))) ||
    !isNonNegativeNumber(value.meters) ||
    !isNonNegativeNumber(value.elapsedMs)
  ) {
    return null;
  }

  return {
    routeId: value.routeId,
    roomCode: value.roomCode,
    roomTitle: value.roomTitle,
    isPrivate: value.isPrivate,
    memberCount: value.memberCount,
    startedAt: value.startedAt,
    endedAt: value.endedAt,
    meters: value.meters,
    elapsedMs: value.elapsedMs,
  };
}

export async function getActivityHistory(): Promise<ActivityHistory> {
  const response = await fetch('/api/activities');
  const payload: unknown = await response.json();
  if (!response.ok) {
    throw new Error(getApiErrorMessage(payload, '活動履歴を取得できませんでした'));
  }
  if (
    typeof payload !== 'object' ||
    payload === null ||
    !('summary' in payload) ||
    !('activities' in payload) ||
    typeof payload.summary !== 'object' ||
    payload.summary === null ||
    !('recordings' in payload.summary) ||
    !('completed' in payload.summary) ||
    !('meters' in payload.summary) ||
    !('elapsedMs' in payload.summary) ||
    typeof payload.summary.recordings !== 'number' ||
    !Number.isSafeInteger(payload.summary.recordings) ||
    payload.summary.recordings < 0 ||
    typeof payload.summary.completed !== 'number' ||
    !Number.isSafeInteger(payload.summary.completed) ||
    payload.summary.completed < 0 ||
    !isNonNegativeNumber(payload.summary.meters) ||
    !isNonNegativeNumber(payload.summary.elapsedMs) ||
    !Array.isArray(payload.activities)
  ) {
    throw new Error('サーバーから不正な活動履歴が返されました');
  }

  const activities = payload.activities.map(parseActivityRecord);
  if (activities.some((activity) => activity === null)) {
    throw new Error('サーバーから不正な活動履歴が返されました');
  }

  return {
    summary: {
      recordings: payload.summary.recordings,
      completed: payload.summary.completed,
      meters: payload.summary.meters,
      elapsedMs: payload.summary.elapsedMs,
    },
    activities: activities as ActivityRecord[],
  };
}

function parseHistoryPoint(value: unknown): Point | null {
  if (!Array.isArray(value) || value.length !== 4 || !value.every((part) => typeof part === 'number' && Number.isFinite(part))) {
    return null;
  }
  return [value[0], value[1], value[2], value[3]];
}

export async function getRoomHistory(roomCode: string): Promise<RoomHistory> {
  const response = await fetch(`/api/rooms/${encodeURIComponent(roomCode)}/history`);
  const payload: unknown = await response.json();
  if (!response.ok) {
    throw new Error(getApiErrorMessage(payload, 'ルームの履歴を取得できませんでした'));
  }
  if (
    typeof payload !== 'object' || payload === null ||
    !('roomCode' in payload) || !('roomTitle' in payload) || !('closedAt' in payload) || !('routes' in payload) ||
    typeof payload.roomCode !== 'string' || typeof payload.roomTitle !== 'string' ||
    (payload.closedAt !== null && typeof payload.closedAt !== 'string') ||
    !Array.isArray(payload.routes)
  ) {
    throw new Error('サーバーから不正なルーム履歴が返されました');
  }

  const routes = payload.routes.map((value: unknown): RoomHistory['routes'][number] | null => {
    if (typeof value !== 'object' || value === null ||
        !('id' in value) || !('memberId' in value) || !('memberName' in value) || !('color' in value) ||
        !('startedAt' in value) || !('endedAt' in value) || !('points' in value) ||
        typeof value.id !== 'number' || !Number.isSafeInteger(value.id) || value.id <= 0 ||
        typeof value.memberId !== 'number' || typeof value.memberName !== 'string' ||
        typeof value.color !== 'string' || typeof value.startedAt !== 'string' ||
        (value.endedAt !== null && typeof value.endedAt !== 'string') || !Array.isArray(value.points)) {
      return null;
    }
    const points = value.points.map((point: unknown) => parseHistoryPoint(point));
    if (points.some((point) => point === null)) return null;
    return {
      id: value.id,
      memberId: value.memberId,
      memberName: value.memberName,
      color: value.color,
      startedAt: value.startedAt,
      endedAt: value.endedAt,
      points: points.filter((point): point is Point => point !== null),
    };
  });
  if (routes.some((route) => route === null)) {
    throw new Error('サーバーから不正なルーム履歴が返されました');
  }
  return {
    roomCode: payload.roomCode,
    roomTitle: payload.roomTitle,
    closedAt: payload.closedAt,
    routes: routes as RoomHistory['routes'],
  };
}

export async function deleteActivity(activity: Pick<ActivityRecord, 'roomCode' | 'routeId'>): Promise<void> {
  const response = await fetch(`/api/rooms/${encodeURIComponent(activity.roomCode)}/routes/${activity.routeId}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    const payload: unknown = await response.json();
    throw new Error(getApiErrorMessage(payload, '記録を削除できませんでした'));
  }
}

export async function logout(): Promise<void> {
  const response = await fetch('/api/auth/logout', { method: 'POST' });
  if (!response.ok) {
    const payload: unknown = await response.json();
    throw new Error(getApiErrorMessage(payload, 'ログアウトに失敗しました'));
  }
}

export async function getCurrentSession(): Promise<AuthSession | null> {
  const response = await fetch('/api/auth/me');
  if (response.status === 401) {
    return null;
  }

  const payload: unknown = await response.json();
  if (!response.ok) {
    throw new Error(getApiErrorMessage(payload, 'ログイン状態を確認できませんでした'));
  }

  if (typeof payload !== 'object' || payload === null || !('room' in payload)) {
    throw new Error('サーバーから不正なルーム情報が返されました');
  }

  const room = payload.room;
  const user = parseAuthUser(payload);
  if (
    !user ||
    typeof room !== 'object' ||
    room === null ||
    !('code' in room) ||
    typeof room.code !== 'string' ||
    !('id' in room) ||
    typeof room.id !== 'number' ||
    !('title' in room) ||
    typeof room.title !== 'string' ||
    !('ownerMemberId' in room) ||
    typeof room.ownerMemberId !== 'number' ||
    !('isPrivate' in room) ||
    typeof room.isPrivate !== 'boolean' ||
    !('closedAt' in room) ||
    (room.closedAt !== null && typeof room.closedAt !== 'string')
  ) {
    throw new Error('サーバーから不正な認証情報またはルーム情報が返されました');
  }

  return {
    user,
    room: {
      id: room.id,
      code: room.code,
      title: room.title,
      ownerMemberId: room.ownerMemberId,
      isPrivate: room.isPrivate,
      closedAt: room.closedAt,
    },
  };
}

export function parseRoom(payload: unknown): Room | null {
  if (typeof payload !== 'object' || payload === null || !('room' in payload)) {
    return null;
  }

  const room = payload.room;
  if (
    typeof room !== 'object' ||
    room === null ||
    !('id' in room) ||
    !('code' in room) ||
    !('title' in room) ||
    !('ownerMemberId' in room) ||
    typeof room.id !== 'number' ||
    typeof room.code !== 'string' ||
    typeof room.title !== 'string' ||
    typeof room.ownerMemberId !== 'number'
  ) {
    return null;
  }

  return {
    id: room.id,
    code: room.code,
    title: room.title,
    ownerMemberId: room.ownerMemberId,
    isPrivate: 'isPrivate' in room && room.isPrivate === true,
    closedAt: 'closedAt' in room && typeof room.closedAt === 'string' ? room.closedAt : null,
  };
}

/** 部屋を作る。レスポンスの code を招待QRに載せる（仕様5章） */
export async function createRoom(title: string): Promise<Room> {
  const response = await fetch('/api/rooms', {
    method: 'POST',
    // Content-Type は必須。サーバーの decodeJSON が application/json 以外を弾く
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title }),
  });

  const payload: unknown = await response.json();
  if (!response.ok) {
    throw new Error(getApiErrorMessage(payload, 'チームを作成できませんでした'));
  }

  const room = parseRoom(payload);
  if (!room) {
    throw new Error('サーバーから不正な部屋情報が返されました');
  }

  return room;
}

/** 招待された部屋へ参加する（仕様5章） */
export async function joinRoom(
  roomCode: string,
  displayName: string,
  color: string,
): Promise<{ room: Room; member: Pick<RoomMember, 'id' | 'displayName' | 'color'> }> {
  const response = await fetch(`/api/rooms/${encodeURIComponent(roomCode)}/join`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ displayName: displayName.trim(), color }),
  });

  const payload: unknown = await response.json();
  if (!response.ok) {
    throw new Error(getApiErrorMessage(payload, '部屋に参加できませんでした'));
  }

  const room = parseRoom(payload);
  if (
    !room ||
    typeof payload !== 'object' ||
    payload === null ||
    !('member' in payload) ||
    typeof payload.member !== 'object' ||
    payload.member === null ||
    !('id' in payload.member) ||
    !('displayName' in payload.member) ||
    !('color' in payload.member) ||
    typeof payload.member.id !== 'number' ||
    typeof payload.member.displayName !== 'string' ||
    typeof payload.member.color !== 'string' ||
    !/^#[0-9a-fA-F]{6}$/.test(payload.member.color) ||
    room.code.toUpperCase() !== roomCode.toUpperCase()
  ) {
    throw new Error('サーバーから不正な参加情報が返されました');
  }

  return {
    room,
    member: {
      id: payload.member.id,
      displayName: payload.member.displayName,
      color: payload.member.color,
    },
  };
}

/** 招待URL。QRにもリンクにもこれを使う（仕様10章 `?room=<CODE>`） */
export function buildInviteUrl(roomCode: string): string {
  const url = new URL('/', window.location.origin);
  url.searchParams.set('room', roomCode);
  return url.toString();
}
