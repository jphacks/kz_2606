import { distanceMeters, trackGapMilliseconds } from './geo';
import type { Point } from './types';

export interface TrackAppend {
  /** 何本目のセグメントに足したか */
  segmentIndex: number;
  coordinate: [number, number];
  /** この点で新しいセグメントが始まったか（ギャップ明け） */
  startedNewSegment: boolean;
}

type TrackListener = (append: TrackAppend) => void;

/**
 * 軌跡の点列を保持する。**Reactの状態には入れない。**
 * 毎秒頂点が増えるので、状態に載せると毎秒アプリ全体が再レンダリングされる。
 * 描画側は subscribe して Leaflet のレイヤを直接伸ばす。
 * 仕様: DATA_MODEL.md 9章
 */
export class TrackBuffer {
  private segments: [number, number][][] = [];
  private meters = 0;
  private lastPoint: Point | null = null;
  private listeners = new Set<TrackListener>();

  /** 採否判定を通った点を足す。ギャップを超えていたらセグメントを分ける */
  add(point: Point): TrackAppend {
    const coordinate: [number, number] = [point[0], point[1]];
    const previousPoint = this.lastPoint;
    const startedNewSegment =
      previousPoint === null || point[2] - previousPoint[2] > trackGapMilliseconds;

    if (startedNewSegment) {
      this.segments.push([coordinate]);
    } else {
      this.segments[this.segments.length - 1].push(coordinate);
      // ギャップを超えた区間は距離に数えない（仕様 3章・11章）
      this.meters += distanceMeters(previousPoint, point);
    }
    this.lastPoint = point;

    const append: TrackAppend = {
      segmentIndex: this.segments.length - 1,
      coordinate,
      startedNewSegment,
    };
    for (const listener of this.listeners) {
      listener(append);
    }
    return append;
  }

  /** 累積距離（メートル）。再計算しない */
  get totalMeters(): number {
    return this.meters;
  }

  get lastRecordedPoint(): Point | null {
    return this.lastPoint;
  }

  /** 既にある点列。描画側の初回投入に使う */
  snapshot(): readonly [number, number][][] {
    return this.segments;
  }

  reset(): void {
    this.segments = [];
    this.meters = 0;
    this.lastPoint = null;
  }

  subscribe(listener: TrackListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }
}
