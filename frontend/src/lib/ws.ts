import type { Point } from './types';

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
