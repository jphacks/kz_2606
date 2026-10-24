// サーバーと共有する型。Point は [lat, lng, ts, acc] のタプル。
// 仕様: DATA_MODEL.md 3章・5章
//
// TODO: Point / Member / ServerMsg / ClientMsg / PALETTE

export interface User {
  id: number;
  username: string;
}

export interface Room {
  id: number;
  code: string;
  title: string;
  ownerMemberId: number;
  isPrivate: boolean;
  closedAt: string | null;
}

export interface AuthSession {
  user: User;
  room: Room;
}

export type Point = [lat: number, lng: number, ts: number, acc: number];
