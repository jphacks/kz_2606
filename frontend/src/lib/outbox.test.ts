import 'fake-indexeddb/auto';
import { clear, get } from 'idb-keyval';
import { afterEach, describe, expect, it } from 'vitest';
import { drainOutbox, enqueuePoints, type OutboxEntry } from './outbox';
import type { Point } from './types';

const roomCode = 'ROOM01';
const storageKey = `outbox:${roomCode}`;

afterEach(async () => {
  await clear();
});

describe('drainOutbox', () => {
  it('sends matching route points and preserves entries for older routes', async () => {
    const oldPoint: Point = [40, 140, 1, 5];
    const activePoints = Array.from(
      { length: 501 },
      (_, index): Point => [40, 140, index + 2, 5],
    );
    await enqueuePoints(roomCode, 10, [oldPoint]);
    await enqueuePoints(roomCode, 11, activePoints);

    const batches: Point[][] = [];
    await drainOutbox(roomCode, 11, (batch) => batches.push(batch));

    expect(batches.map((batch) => batch.length)).toEqual([500, 1]);
    expect(batches.flat()).toEqual(activePoints);
    expect(await get<OutboxEntry[]>(storageKey)).toEqual([
      { routeId: 10, pts: [oldPoint] },
    ]);
  });
});
