import type { Point } from './types';

const maxAccuracyMeters = 30;
const minDistanceMeters = 5;
const maxSpeedMetersPerSecond = 14;
export const trackGapMilliseconds = 30_000;
export const trackSimplificationToleranceMeters = 5;
const earthRadiusMeters = 6_371_000;
const metersPerLatitudeDegree = 111_320;

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

export function simplifyTrackSegment(
  coordinates: readonly [number, number][],
  toleranceMeters = trackSimplificationToleranceMeters,
): [number, number][] {
  if (coordinates.length <= 2 || toleranceMeters <= 0) {
    return coordinates.map(([latitude, longitude]) => [latitude, longitude]);
  }

  const referenceLatitude =
    ((coordinates[0][0] + coordinates[coordinates.length - 1][0]) / 2) * (Math.PI / 180);
  const metersPerLongitudeDegree = metersPerLatitudeDegree * Math.cos(referenceLatitude);
  const projected = coordinates.map(([latitude, longitude]) => [
    longitude * metersPerLongitudeDegree,
    latitude * metersPerLatitudeDegree,
  ]);
  const keep = new Uint8Array(coordinates.length);
  const pending: [number, number][] = [[0, coordinates.length - 1]];
  const toleranceSquared = toleranceMeters ** 2;
  keep[0] = 1;
  keep[coordinates.length - 1] = 1;

  while (pending.length > 0) {
    const range = pending.pop();
    if (!range) continue;
    const [startIndex, endIndex] = range;
    const [startX, startY] = projected[startIndex];
    const [endX, endY] = projected[endIndex];
    const dx = endX - startX;
    const dy = endY - startY;
    const lengthSquared = dx * dx + dy * dy;
    let farthestIndex = -1;
    let farthestDistanceSquared = toleranceSquared;

    for (let index = startIndex + 1; index < endIndex; index += 1) {
      const [x, y] = projected[index];
      const projection = lengthSquared === 0
        ? 0
        : Math.max(0, Math.min(1, ((x - startX) * dx + (y - startY) * dy) / lengthSquared));
      const distanceX = x - (startX + projection * dx);
      const distanceY = y - (startY + projection * dy);
      const distanceSquared = distanceX * distanceX + distanceY * distanceY;
      if (distanceSquared > farthestDistanceSquared) {
        farthestDistanceSquared = distanceSquared;
        farthestIndex = index;
      }
    }

    if (farthestIndex !== -1) {
      keep[farthestIndex] = 1;
      pending.push([startIndex, farthestIndex], [farthestIndex, endIndex]);
    }
  }

  return coordinates.filter((_, index) => keep[index] === 1)
    .map(([latitude, longitude]) => [latitude, longitude]);
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
