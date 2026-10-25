import { afterEach, describe, expect, it, vi } from 'vitest';
import { joinRoom } from './api';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('joinRoom', () => {
  it('posts the normalized join details and parses the room and member', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      room: {
        id: 7,
        code: 'KZ26AB',
        title: 'Park',
        ownerMemberId: 1,
        isPrivate: false,
        closedAt: null,
      },
      member: {
        id: 2,
        displayName: 'Guest',
        color: '#0E7861',
      },
    }), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const result = await joinRoom('KZ26AB', ' Guest ', '#0E7861');

    expect(fetchMock).toHaveBeenCalledWith('/api/rooms/KZ26AB/join', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ displayName: 'Guest', color: '#0E7861' }),
    });
    expect(result.room.code).toBe('KZ26AB');
    expect(result.member).toMatchObject({
      id: 2,
      displayName: 'Guest',
      color: '#0E7861',
    });
  });

  it('surfaces the server message for failed joins', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'private_room', message: 'この部屋には参加できません' },
    }), { status: 403 })));

    await expect(joinRoom('KZ26AB', 'Guest', '#0E7861'))
      .rejects.toThrow('この部屋には参加できません');
  });

  it('rejects malformed successful responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      room: { id: 7, code: 'KZ26AB', title: 'Park', ownerMemberId: 1 },
      member: { id: 2, displayName: 'Guest' },
    }), { status: 200 })));

    await expect(joinRoom('KZ26AB', 'Guest', '#0E7861'))
      .rejects.toThrow('サーバーから不正な参加情報が返されました');
  });
});
