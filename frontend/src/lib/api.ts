// REST 呼び出し。

import type { AuthSession, Room, RoomMember, User } from './types';

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
