// REST 呼び出し。

import type { Room, User } from './types';

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
  const response = await fetch('/api/auth/me');
  if (response.status === 401) {
    return null;
  }

  const payload: unknown = await response.json();
  if (!response.ok) {
    throw new Error(getApiErrorMessage(payload, 'ログイン状態を確認できませんでした'));
  }

  const user = parseAuthUser(payload);
  if (!user) {
    throw new Error('サーバーから不正な認証情報が返されました');
  }

  return user;
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

/** 招待URL。QRにもリンクにもこれを使う（仕様10章 `?room=<CODE>`） */
export function buildInviteUrl(roomCode: string): string {
  const url = new URL('/', window.location.origin);
  url.searchParams.set('room', roomCode);
  return url.toString();
}
