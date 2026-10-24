import React from 'react';

interface BeforeRecordProps {
  hasLocation: boolean;
  onStartRecording?: () => void;
}

export default function BeforeRecord({ hasLocation, onStartRecording }: BeforeRecordProps) {
  return (
    <div style={{ 
      position: 'absolute', 
      top: 0, 
      left: 0, 
      width: '100%', 
      height: '100%', 
      display: 'flex', 
      flexDirection: 'column', 
      justifyContent: 'space-between', 
      pointerEvents: 'none', // UIの隙間から地図を触れるようにする
      padding: '12px',
      boxSizing: 'border-box',
      zIndex: 10
    }}>
      {/* ================= 上エリア：ヘッダー ================= */}
      <div style={{ 
        background: 'white', 
        padding: '12px 16px', 
        borderRadius: '16px',
        boxShadow: '0 4px 16px rgba(0,0,0,0.1)',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        pointerEvents: 'auto' // ボタンなどは押せるように戻す
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <button style={{ border: 'none', background: 'none', fontSize: '18px', cursor: 'pointer' }}>‹</button>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 'bold', color: '#1a1a1a' }}>まちあるき部</div>
            <div style={{ fontSize: '11px', color: '#717171' }}>チーム共有マップ : 吉祥寺</div>
          </div>
        </div>
        <button style={{ border: 'none', background: '#f5f5f5', borderRadius: '50%', width: '36px', height: '36px', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          👥
        </button>
      </div>

      {/* ================= 下エリア：記録前操作パネル ================= */}
      <div style={{ 
        background: 'white', 
        padding: '20px', 
        borderRadius: '20px', 
        boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
        pointerEvents: 'auto' // 操作できるように戻す
      }}>
        {/* 今日のキャンバス・ステータス */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
          <span style={{ fontSize: '11px', fontWeight: 'bold', color: '#888', letterSpacing: '0.5px' }}>TODAY'S CANVAS</span>
          <span style={{ fontSize: '11px', background: '#e6f4ea', color: '#137333', padding: '3px 8px', borderRadius: '12px', fontWeight: 'bold' }}>記録前</span>
        </div>

        {/* タイトル */}
        <div style={{ fontSize: '18px', fontWeight: 'bold', color: '#1a1a1a', marginBottom: '6px' }}>
          吉祥寺にハートを描こう
        </div>

        {/* 参加メンバーアイコン */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
          <div style={{ display: 'flex', gap: '6px' }}>
            <span style={{ width: '28px', height: '28px', borderRadius: '50%', background: '#0d6efd', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 'bold' }}>さ</span>
            <span style={{ width: '28px', height: '28px', borderRadius: '50%', background: '#ffc107', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 'bold' }}>ゆ</span>
            <span style={{ width: '28px', height: '28px', borderRadius: '50%', background: '#198754', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 'bold' }}>は</span>
            <span style={{ width: '28px', height: '28px', borderRadius: '50%', background: '#dc3545', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 'bold' }}>り</span>
          </div>
          <span style={{ fontSize: '12px', color: '#666' }}>4人が参加済み</span>
        </div>

        {/* 位置情報ステータス */}
        <div style={{ fontSize: '11px', color: hasLocation ? '#198754' : '#dc3545', marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '4px' }}>
          <span>{hasLocation ? '✓' : '✕'}</span>
          <span>{hasLocation ? '位置情報は利用可能です' : '位置情報を取得中/未許可です'}</span>
        </div>

        {/* 記録開始ボタン */}
        <button 
          onClick={onStartRecording}
          style={{ 
            width: '100%', 
            background: '#0d5c3a', 
            color: 'white', 
            border: 'none', 
            padding: '14px', 
            borderRadius: '14px', 
            fontSize: '15px', 
            fontWeight: 'bold',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px'
          }}
        >
          ▶ 記録開始
        </button>
      </div>
    </div>
  );
}