import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';

// 現在地取得時に地図の中心を自動で移動するヘルパー
const RecenterMap = ({ center }: { center: [number, number] }) => {
    const map = useMap();
    useEffect(() => {
    map.setView(center, map.getZoom());
    }, [center, map]);
    return null;
};

interface MapProps {
    position: [number, number];
    hasLocation: boolean;
}

export default function Map({ position, hasLocation }: MapProps) {
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
            url="https://{s}.tile.openstreetmap.org/{z}x/{y}.png"
        />
        {hasLocation && (
            <Marker position={position}>
            <Popup>いまここ（あなた）</Popup>
            </Marker>
        )}
        <RecenterMap center={position} />
        </MapContainer>
    </div>
    );
}