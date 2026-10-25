import { afterEach, describe, expect, it, vi } from 'vitest';
import { getActivityHistory, joinRoom } from './api';

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
        color: '#0e7861',
      },
    }), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const result = await joinRoom('KZ26AB', ' Guest ', '#0e7861');

    expect(fetchMock).toHaveBeenCalledWith('/api/rooms/KZ26AB/join', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ displayName: 'Guest', color: '#0e7861' }),
    });
    expect(result.room.code).toBe('KZ26AB');
    expect(result.member).toMatchObject({
      id: 2,
      displayName: 'Guest',
      color: '#0e7861',
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

describe('getActivityHistory', () => {
  it('parses real activity summary and route records', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      summary: { recordings: 1, completed: 1, meters: 65.98, elapsedMs: 600_000 },
      activities: [{
        routeId: 12,
        roomCode: 'ABC234',
        roomTitle: '大通公園を歩く',
        isPrivate: false,
        memberCount: 3,
        startedAt: '2026-10-24T05:00:00Z',
        endedAt: '2026-10-24T05:10:00Z',
        meters: 65.98,
        elapsedMs: 600_000,
      }],
    }), { status: 200 })));

    const history = await getActivityHistory();

    expect(fetch).toHaveBeenCalledWith('/api/activities');
    expect(history.summary).toEqual({
      recordings: 1,
      completed: 1,
      meters: 65.98,
      elapsedMs: 600_000,
    });
    expect(history.activities[0]).toMatchObject({ routeId: 12, roomTitle: '大通公園を歩く' });
  });

  it('surfaces the server error message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'internal_error', message: '活動履歴を取得できませんでした' },
    }), { status: 500 })));

    await expect(getActivityHistory()).rejects.toThrow('活動履歴を取得できませんでした');
  });

  it('rejects malformed activity values', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      summary: { recordings: 1, completed: 1, meters: 10, elapsedMs: 1000 },
      activities: [{ routeId: 1, meters: '10' }],
    }), { status: 200 })));

    await expect(getActivityHistory()).rejects.toThrow('サーバーから不正な活動履歴が返されました');
  });
});
