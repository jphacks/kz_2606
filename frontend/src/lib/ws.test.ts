import { describe, expect, it } from 'vitest';
import { parseRoomSnapshot } from './ws';

describe('parseRoomSnapshot', () => {
  it('preserves route metadata and returns snapshot points', () => {
    const parsed = parseRoomSnapshot({
      t: 'snapshot',
      room: { code: 'ROOM01', title: 'Test room', closedAt: null },
      you: { id: 7 },
      members: [{
        id: 7,
        displayName: 'walker',
        color: '#00e5ff',
        online: true,
        banned: false,
        recording: true,
      }],
      routes: [{
        id: 12,
        member: 7,
        color: '#00e5ff',
        startedAt: '2026-10-25T01:00:00Z',
        endedAt: null,
        meters: 100,
        elapsedMs: 30_000,
        pts: [[35, 139, 1_792_900_000_000, 5]],
      }],
    });

    expect(parsed?.snapshot.routes[0]).toMatchObject({
      id: 12,
      member: 7,
      color: '#00e5ff',
      startedAt: '2026-10-25T01:00:00Z',
      endedAt: null,
    });
    expect(parsed?.routePoints.get(12)).toEqual([[35, 139, 1_792_900_000_000, 5]]);
  });
});
