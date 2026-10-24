import { useEffect, useState } from 'react';
import Home from './ui/home';
import MapView from './ui/map';
import BeforeRecord from './ui/before_record';
import Setting from './ui/setting';
import Result from './ui/result';
import Auth from './ui/Auth';

export default function App() {
  const path = window.location.pathname;

  if (path === '/login' || path === '/signin') {
    return <Auth />;
  }

  if (path === '/result') {
    return <Result />;
  }

  return <MainExperience />;
}

function MainExperience() {
  // 現在どの画面にいるかを管理 ("home" | "before_record" | "setting")
  const [currentScreen, setCurrentScreen] = useState<'home' | 'before_record' | 'setting'>('home');
  
  const [position, setPosition] = useState<[number, number]>([36.578, 136.648]);
  const [hasLocation, setHasLocation] = useState(false);
  const [userColor, setUserColor] = useState('#0E7861'); // 選択した軌跡カラー

  useEffect(() => {
    if ('geolocation' in navigator) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          setPosition([pos.coords.latitude, pos.coords.longitude]);
          setHasLocation(true);
        },
        (err) => console.error('位置情報取得エラー:', err),
        { enableHighAccuracy: true }
      );
    }
  }, []);

  const handleStartRecording = () => {
    console.log('記録が開始されました！');
  };

  return (
    <div style={{
      width: '100vw',
      height: '100vh',
      backgroundColor: '#f0f2f5',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      margin: 0,
      padding: 0,
      overflow: 'hidden'
    }}>
      {/* 📱 スマホアプリ風のメインフレーム */}
      <div style={{
        width: '100%',
        maxWidth: '430px',
        height: '100%',
        maxHeight: '100vh',
        backgroundColor: '#ffffff',
        position: 'relative',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        boxShadow: '0 0 20px rgba(0,0,0,0.1)'
      }}>

        {/* 画面の切り替え */}
        {currentScreen === 'home' && (
          // ホーム画面
          <div style={{ flex: 1, overflowY: 'auto' }}>
            <Home />
            {/* 🗺️ マップ画面へ進むボタン */}
            <div style={{ padding: '20px', textAlign: 'center', backgroundColor: '#FAFAFA' }}>
              <button
                onClick={() => setCurrentScreen('before_record')}
                style={{
                  width: '100%',
                  padding: '14px 20px',
                  background: '#0E7861',
                  color: 'white',
                  border: 'none',
                  borderRadius: '12px',
                  cursor: 'pointer',
                  fontWeight: 'bold',
                  fontSize: '15px',
                  boxShadow: '0 4px 6px rgba(14, 120, 97, 0.2)'
                }}
              >
                🗺️ マップ画面へ進む（テスト用）
              </button>
            </div>
          </div>
        )}

        {currentScreen === 'before_record' && (
          // 🗺️ 地図を全面にし、カードレイヤーを上に重ねる構造 (flex: 1 を追加)
          <div style={{ flex: 1, position: 'relative', width: '100%', overflow: 'hidden' }}>
            
            {/* 1. 背面のマップ（全体に敷き詰める） */}
            <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 1 }}>
              <MapView position={position} hasLocation={hasLocation} />
            </div>

            {/* 2. 前面のオーバーレイUI */}
            <div style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: '100%',
              zIndex: 10,
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'space-between',
              pointerEvents: 'none',
              padding: '16px',
              boxSizing: 'border-box'
            }}>
              <BeforeRecord 
                hasLocation={hasLocation} 
                onStartRecording={handleStartRecording} 
                onOpenSetting={() => setCurrentScreen('setting')} 
              />
            </div>

          </div>
        )}

        {currentScreen === 'setting' && (
          // ⚙️ 設定画面（メンバーとカラー）
          <div style={{ flex: 1, overflowY: 'auto', zIndex: 20, backgroundColor: '#FAFAFA' }}>
            <Setting 
              onBack={() => setCurrentScreen('before_record')}
              onSaveAndReturn={(selectedColor) => {
                setUserColor(selectedColor);
                setCurrentScreen('before_record');
              }}
            />
          </div>
        )}

      </div>
    </div>
  );
}