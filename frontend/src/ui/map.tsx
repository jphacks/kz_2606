import { useEffect, useRef } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { defaultMemberColor } from '../lib/member_colors';
import type { RoomRouteTrack } from '../lib/types';
import type { TrackBuffer } from '../lib/track_buffer';

// 現在地取得時に地図の中心を自動で移動するヘルパー
const RecenterMap = ({ center }: { center: [number, number] }) => {
    const map = useMap();
    useEffect(() => {
    map.setView(center, map.getZoom());
    }, [center, map]);
    return null;
};

// 軌跡のレイヤ。点列は TrackBuffer が持っていて、ここは Leaflet のポリラインを
// 直接伸ばすだけ。点が増えても React の再レンダリングは起きない。
// 一括投入は setLatLngs / 増分は addLatLng（仕様: DATA_MODEL.md 9章）
const TrackLayer = ({ buffer, color }: { buffer: TrackBuffer; color: string }) => {
    const map = useMap();
    const colorRef = useRef(color);
    const polylinesRef = useRef<L.Polyline[]>([]);

    useEffect(() => {
        colorRef.current = color;
        for (const polyline of polylinesRef.current) {
            polyline.setStyle({ color });
        }
    }, [color]);

    useEffect(() => {
        const group = L.layerGroup().addTo(map);
        const polylines: L.Polyline[] = [];

        const appendSegment = (coordinates: [number, number][]) => {
            const polyline = L.polyline(coordinates, { color: colorRef.current, weight: 5 }).addTo(group);
            polylines.push(polyline);
            return polyline;
        };

        // 既にある点列はまとめて入れる
        for (const segment of buffer.snapshot()) {
            appendSegment([...segment]);
        }

        const unsubscribe = buffer.subscribe((update) => {
            if (update.type === 'replace') {
                polylines[update.segmentIndex]?.setLatLngs(update.coordinates);
                return;
            }
            if (update.finalizedSegment) {
                polylines[update.finalizedSegment.segmentIndex]?.setLatLngs(
                    update.finalizedSegment.coordinates,
                );
            }
            if (update.startedNewSegment || !polylines[update.segmentIndex]) {
                appendSegment(update.segmentStartCoordinate
                    ? [update.segmentStartCoordinate, update.coordinate]
                    : [update.coordinate]);
                return;
            }
            polylines[update.segmentIndex].addLatLng(update.coordinate);
        });
        polylinesRef.current = polylines;

        return () => {
            unsubscribe();
            group.remove();
            polylinesRef.current = [];
        };
    }, [buffer, map]);

    return null;
};

interface MapProps {
    position: [number, number];
    hasLocation: boolean;
    trackBuffer?: TrackBuffer;
    trackColor?: string;
    otherTracks?: RoomRouteTrack[];
}

export default function Map({
      position,
      hasLocation,
      trackBuffer,
      trackColor = defaultMemberColor,
      otherTracks = [],
}: MapProps) {
    return (
    <div style={{ width: '100%', height: '100%', position: 'relative' }}>
        <MapContainer 
        center={position} 
        zoom={16} 
        style={{ width: '100%', height: '100%' }}
        zoomControl={false}
        >
        <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        {hasLocation && (
            <Marker position={position}>
            <Popup>いまここ（あなた）</Popup>
            </Marker>
        )}
        {trackBuffer && <TrackLayer buffer={trackBuffer} color={trackColor} />}
        {otherTracks.map((track) => (
            <TrackLayer key={track.routeId} buffer={track.buffer} color={track.color} />
        ))}
        <RecenterMap center={position} />
        </MapContainer>
    </div>
    );
}
