import { del, get, set } from 'idb-keyval';
import type { Point } from './types';
import { maxPointsPerMessage } from './ws';

export interface OutboxEntry {
  routeId: number;
  pts: Point[];
}

export const maxOutboxPoints = 20_000;

const locks = new Map<string, Promise<void>>();

function outboxKey(roomCode: string): string {
  return `outbox:${roomCode.toUpperCase()}`;
}

function withOutboxLock<T>(key: string, operation: () => Promise<T>): Promise<T> {
  const previous = locks.get(key) ?? Promise.resolve();
  const current = previous.then(operation);
  const settled = current.then(() => undefined, () => undefined);
  locks.set(key, settled);

  return current.finally(() => {
    if (locks.get(key) === settled) {
      locks.delete(key);
    }
  });
}

function countPoints(entries: OutboxEntry[]): number {
  return entries.reduce((total, entry) => total + entry.pts.length, 0);
}

async function readEntries(key: string): Promise<OutboxEntry[]> {
  return (await get<OutboxEntry[]>(key)) ?? [];
}

export async function enqueuePoints(
  roomCode: string,
  routeId: number,
  points: Point[],
): Promise<number> {
  if (points.length === 0) return 0;
  const key = outboxKey(roomCode);
  return withOutboxLock(key, async () => {
    const entries = await readEntries(key);
    const last = entries[entries.length - 1];
    if (last?.routeId === routeId) {
      last.pts.push(...points);
    } else {
      entries.push({ routeId, pts: [...points] });
    }

    let excess = Math.max(0, countPoints(entries) - maxOutboxPoints);
    let dropped = excess;
    while (excess > 0 && entries.length > 0) {
      const first = entries[0];
      const removed = Math.min(excess, first.pts.length);
      first.pts.splice(0, removed);
      excess -= removed;
      if (first.pts.length === 0) entries.shift();
    }

    await set(key, entries);
    return dropped;
  });
}

export async function drainOutbox(
  roomCode: string,
  routeId: number,
  sendBatch: (points: Point[]) => void,
): Promise<void> {
  const key = outboxKey(roomCode);
  await withOutboxLock(key, async () => {
    const entries = await readEntries(key);
    while (entries.length > 0) {
      const first = entries[0];
      if (first.routeId !== routeId) {
        throw new Error(`outbox route mismatch: queued ${first.routeId}, active ${routeId}`);
      }

      const batch = first.pts.slice(0, maxPointsPerMessage);
      sendBatch(batch);
      first.pts.splice(0, batch.length);
      if (first.pts.length === 0) entries.shift();
      await set(key, entries);
    }
    await del(key);
  });
}

export async function getOutboxPointCount(roomCode: string): Promise<number> {
  const key = outboxKey(roomCode);
  return withOutboxLock(key, async () => countPoints(await readEntries(key)));
}
