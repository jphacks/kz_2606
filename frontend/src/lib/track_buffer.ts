import {
  distanceMeters,
  simplifyTrackSegment,
  trackGapMilliseconds,
} from './geo';
import type { Point } from './types';

export const maxTrackSegmentPoints = 1_000;

type Coordinate = [number, number];

export interface TrackAppend {
  type: 'append';
  segmentIndex: number;
  coordinate: Coordinate;
  startedNewSegment: boolean;
  segmentStartCoordinate?: Coordinate;
  finalizedSegment?: { segmentIndex: number; coordinates: Coordinate[] };
}

export interface TrackSegmentReplacement {
  type: 'replace';
  segmentIndex: number;
  coordinates: Coordinate[];
}

type TrackUpdate = TrackAppend | TrackSegmentReplacement;
type TrackListener = (update: TrackUpdate) => void;

/**
 * 軌跡の点列を保持する。**Reactの状態には入れない。**
 * 毎秒頂点が増えるので、状態に載せると毎秒アプリ全体が再レンダリングされる。
 * 描画側は subscribe して Leaflet のレイヤを直接伸ばす。
 * Leafletの各線は最大1000点で分け、確定した線は描画用に間引く。
 */
export class TrackBuffer {
  private segments: Coordinate[][] = [];
  private meters = 0;
  private lastPoint: Point | null = null;
  private currentSegmentPointCount = 0;
  private listeners = new Set<TrackListener>();

  /** 採否判定を通った点を足す。ギャップを超えていたらセグメントを分ける */
  add(point: Point): TrackAppend {
    const coordinate: Coordinate = [point[0], point[1]];
    const previousPoint = this.lastPoint;
    let startedNewSegment: boolean;
    let segmentStartCoordinate: Coordinate | undefined;
    let finalizedSegment: TrackAppend['finalizedSegment'];
    const gapSincePrevious =
      previousPoint === null || point[2] - previousPoint[2] > trackGapMilliseconds;
    const previousSegmentFinalized = previousPoint !== null && this.currentSegmentPointCount === 0;

    if (gapSincePrevious || previousSegmentFinalized) {
      startedNewSegment = true;
      if (previousPoint !== null && !previousSegmentFinalized) {
        const segmentIndex = this.segments.length - 1;
        const coordinates = simplifyTrackSegment(this.segments[segmentIndex]);
        this.segments[segmentIndex] = coordinates;
        finalizedSegment = { segmentIndex, coordinates };
      }
      if (previousSegmentFinalized && !gapSincePrevious && previousPoint) {
        segmentStartCoordinate = [previousPoint[0], previousPoint[1]];
        this.segments.push([segmentStartCoordinate, coordinate]);
        this.currentSegmentPointCount = 2;
      } else {
        this.segments.push([coordinate]);
        this.currentSegmentPointCount = 1;
      }
    } else {
      startedNewSegment = false;
      const segmentIndex = this.segments.length - 1;
      if (this.currentSegmentPointCount >= maxTrackSegmentPoints) {
        const coordinates = simplifyTrackSegment(this.segments[segmentIndex]);
        this.segments[segmentIndex] = coordinates;
        finalizedSegment = { segmentIndex, coordinates };
        segmentStartCoordinate = [previousPoint[0], previousPoint[1]];
        this.segments.push([segmentStartCoordinate, coordinate]);
        this.currentSegmentPointCount = 2;
        startedNewSegment = true;
      } else {
        this.segments[segmentIndex].push(coordinate);
        this.currentSegmentPointCount += 1;
      }
      // ギャップを超えた区間は距離に数えない（仕様 3章・11章）
      this.meters += distanceMeters(previousPoint, point);
    }
    this.lastPoint = point;

    const append: TrackAppend = {
      type: 'append',
      segmentIndex: this.segments.length - 1,
      coordinate,
      startedNewSegment,
      segmentStartCoordinate,
      finalizedSegment,
    };
    for (const listener of this.listeners) {
      listener(append);
    }
    return append;
  }

  finalize(): void {
    const segmentIndex = this.segments.length - 1;
    if (segmentIndex < 0 || this.currentSegmentPointCount === 0) return;

    const coordinates = simplifyTrackSegment(this.segments[segmentIndex]);
    this.segments[segmentIndex] = coordinates;
    this.currentSegmentPointCount = 0;
    const replacement: TrackSegmentReplacement = {
      type: 'replace',
      segmentIndex,
      coordinates,
    };
    for (const listener of this.listeners) {
      listener(replacement);
    }
  }

  /** 累積距離（メートル）。再計算しない */
  get totalMeters(): number {
    return this.meters;
  }

  get lastRecordedPoint(): Point | null {
    return this.lastPoint;
  }

  /** 既にある点列。描画側の初回投入に使う */
  snapshot(): readonly Coordinate[][] {
    return this.segments;
  }

  reset(): void {
    this.segments = [];
    this.meters = 0;
    this.lastPoint = null;
    this.currentSegmentPointCount = 0;
  }

  subscribe(listener: TrackListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }
}
