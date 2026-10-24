import type { Point } from './types';

export function connectToRoom(roomCode: string): WebSocket {
  const url = new URL('/ws', window.location.href);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  url.searchParams.set('room', roomCode);
  return new WebSocket(url);
}

export function sendStart(socket: WebSocket): void {
  socket.send(JSON.stringify({ t: 'start' }));
}

export function sendPoints(socket: WebSocket, points: Point[]): void {
  if (points.length === 0 || points.length > 500) {
    throw new Error('送信する位置情報の点数が不正です');
  }
  socket.send(JSON.stringify({ t: 'points', pts: points }));
}

export function sendStop(socket: WebSocket): void {
  socket.send(JSON.stringify({ t: 'stop' }));
}
