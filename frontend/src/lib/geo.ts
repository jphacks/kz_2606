import type { Point } from './types';

const maxAccuracyMeters = 30;
const minDistanceMeters = 5;
const maxSpeedMetersPerSecond = 14;
export const trackGapMilliseconds = 30_000;
const earthRadiusMeters = 6_371_000;

export function distanceMeters(from: Point, to: Point): number {
  const toRadians = (degrees: number) => (degrees * Math.PI) / 180;
  const lat1 = toRadians(from[0]);
  const lat2 = toRadians(to[0]);
  const deltaLat = lat2 - lat1;
  const deltaLng = toRadians(to[1] - from[1]);
  const haversine =
    Math.sin(deltaLat / 2) ** 2 +
    Math.cos(lat1) * Math.cos(lat2) * Math.sin(deltaLng / 2) ** 2;

  return 2 * earthRadiusMeters * Math.asin(Math.sqrt(haversine));
}

export function shouldAcceptPoint(point: Point, previousPoint?: Point): boolean {
  if (
    !Number.isFinite(point[0]) ||
    !Number.isFinite(point[1]) ||
    !Number.isFinite(point[2]) ||
    !Number.isFinite(point[3]) ||
    point[0] < -90 ||
    point[0] > 90 ||
    point[1] < -180 ||
    point[1] > 180 ||
    point[3] < 0 ||
    point[3] > maxAccuracyMeters
  ) {
    return false;
  }
  if (!previousPoint) {
    return true;
  }

  const elapsedMilliseconds = point[2] - previousPoint[2];
  if (elapsedMilliseconds <= 0) {
    return false;
  }
  if (elapsedMilliseconds > trackGapMilliseconds) {
    return true;
  }

  const distance = distanceMeters(previousPoint, point);
  if (distance < minDistanceMeters) {
    return false;
  }

  return distance / (elapsedMilliseconds / 1000) <= maxSpeedMetersPerSecond;
}
