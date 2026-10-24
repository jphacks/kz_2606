import { useEffect, useState } from 'react';
import Home from './ui/home';
import MapView from './ui/map';
import BeforeRecord from './ui/before_record';
import Auth from './ui/signin';
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
  // 現在表示している画面を管理する
  const [currentScreen, setCurrentScreen] = useState<'home' | 'before_record'>('home');
  
  const [position, setPosition] = useState<[number, number]>([36.578, 136.648]);
  const [hasLocation, setHasLocation] = useState(false);

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
        {currentScreen === 'home' ? (
          // ホーム画面
          <div style={{ flex: 1, overflowY: 'auto' }}>
            <Home />
            {/* 画面遷移用ボタン */}
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
        ) : (
          // マップ ＋ 記録前画面
          <>
            <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 1 }}>
              <MapView position={position} hasLocation={hasLocation} />
            </div>
            <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 10, pointerEvents: 'none' }}>
              <BeforeRecord hasLocation={hasLocation} onStartRecording={handleStartRecording} />
            </div>
          </>
        )}

      </div>
    </div>
  );
}
