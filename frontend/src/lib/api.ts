// REST 呼び出し。
//
// TODO: createRoom / roomExists

import type { AuthSession, User } from './types';

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
