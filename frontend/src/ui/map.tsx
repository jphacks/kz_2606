import { Fragment, useEffect, useRef, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import QRCode from 'qrcode';
import { X } from 'lucide-react';
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
    inviteUrl?: string;
    showInviteQr?: boolean;
    onShowInviteQr?: () => void;
    onDismissInviteQr?: () => void;
}

export default function Map({
      position,
      hasLocation,
      trackBuffer,
      trackColor = defaultMemberColor,
      otherTracks = [],
      inviteUrl,
      showInviteQr = false,
      onShowInviteQr,
      onDismissInviteQr,
}: MapProps) {
    const [inviteQrCode, setInviteQrCode] = useState<string | null>(null);
    const [qrCodeError, setQrCodeError] = useState<string | null>(null);

    useEffect(() => {
        if (!inviteUrl) {
            setInviteQrCode(null);
            setQrCodeError(null);
            return;
        }

        let cancelled = false;
        setInviteQrCode(null);
        setQrCodeError(null);
        QRCode.toDataURL(inviteUrl, { width: 192, margin: 1 })
            .then((dataUrl) => {
                if (!cancelled) setInviteQrCode(dataUrl);
            })
            .catch((error: unknown) => {
                if (!cancelled) {
                    setQrCodeError(error instanceof Error ? error.message : String(error));
                }
            });

        return () => {
            cancelled = true;
        };
    }, [inviteUrl]);

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
            <Fragment key={track.routeId}>
              <TrackLayer buffer={track.buffer} color={track.color} />
              {track.active && track.lastPoint && (
                <Marker key={`${track.routeId}-marker`} position={track.lastPoint}>
                  <Popup>{track.memberName ?? `メンバー ${track.memberId}`}</Popup>
                </Marker>
              )}
            </Fragment>
        ))}
        <RecenterMap center={position} />
        </MapContainer>
        {inviteUrl && !showInviteQr && (
            <button
                type="button"
                onClick={onShowInviteQr}
                className="absolute left-4 top-20 z-[1000] flex items-center gap-2 rounded-xl bg-white/95 px-3 py-2 text-sm font-bold text-gray-700 shadow-lg backdrop-blur-md hover:bg-white"
            >
                <span aria-hidden="true">▦</span>
                QRコードを表示
            </button>
        )}
        {inviteUrl && showInviteQr && (
            <div
                role="dialog"
                aria-label="チーム招待QRコード"
                className="absolute left-4 right-4 top-20 z-[1000] rounded-2xl bg-white/95 p-4 text-center shadow-2xl backdrop-blur-md"
            >
                <button
                    type="button"
                    onClick={onDismissInviteQr}
                    className="absolute right-2 top-2 rounded-full p-1 text-gray-500 hover:bg-gray-100"
                    aria-label="招待QRコードを閉じる"
                >
                    <X size={18} />
                </button>
                <p className="text-sm font-bold text-gray-900">チームができました</p>
                <p className="mt-1 text-xs text-gray-500">このQRをメンバーに読み取ってもらいましょう</p>
                {inviteQrCode ? (
                    <img src={inviteQrCode} alt="チーム招待用QRコード" className="mx-auto mt-3 h-48 w-48" />
                ) : qrCodeError ? (
                    <p role="alert" className="mt-3 text-xs text-red-600">QRコードを生成できませんでした</p>
                ) : (
                    <div className="mx-auto mt-3 flex h-48 w-48 items-center justify-center text-sm text-gray-400">
                        QRコードを生成中...
                    </div>
                )}
            </div>
        )}
    </div>
    );
}
