import { useEffect, useRef, useState } from 'react';
import Home from './ui/home';
import MapView from './ui/map';
import BeforeRecord from './ui/before_record';
import Setting from './ui/setting';
import Result from './ui/result';
import Auth from './ui/Auth';
import { buildInviteUrl, getCurrentSession, logout } from './lib/api';
import { shouldAcceptPoint } from './lib/geo';
import { connectToRoom, flushIntervalMilliseconds, parseRoomMember, parseRoomSnapshot, sendPoints, sendStart, sendStop } from './lib/ws';
import { TrackBuffer } from './lib/track_buffer';
import type { Point, Room, RoomSnapshot, User } from './lib/types';

type Screen = 'home' | 'before_record' | 'solo_record' | 'setting' | 'login' | 'signin' | 'result';

function formatRouteElapsedTime(milliseconds: number): string {
  const totalMinutes = Math.floor(milliseconds / 60_000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return hours > 0 ? `${hours}時間${minutes}分` : `${minutes}分`;
}

function getScreenFromPath(path: string): Screen {
  if (path === '/login') return 'login';
  if (path === '/signin') return 'signin';
  if (path === '/solo') return 'solo_record';
  if (path === '/result') return 'result';
  return 'home';
}

export default function App() {
  return <MainExperience initialScreen={getScreenFromPath(window.location.pathname)} />;
}

function MainExperience({ initialScreen }: { initialScreen: Screen }) {
  const [currentScreen, setCurrentScreen] = useState<Screen>(initialScreen);
  const [position, setPosition] = useState<[number, number]>([36.578, 136.648]);
  const [hasLocation, setHasLocation] = useState(false);
  const [userColor, setUserColor] = useState('#0E7861'); // 選択した軌跡カラー
  const [user, setUser] = useState<User | null>(null);
  const [roomCode, setRoomCode] = useState<string | null>(null);
  const [roomTitle, setRoomTitle] = useState<string | null>(null);
  const [teamRoom, setTeamRoom] = useState<Room | null>(null);
  const [roomSnapshot, setRoomSnapshot] = useState<RoomSnapshot | null>(null);
  const [roomDataError, setRoomDataError] = useState('');
  const [sessionLoaded, setSessionLoaded] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [authError, setAuthError] = useState('');
  const [recordingError, setRecordingError] = useState('');
  const [recordingMessage, setRecordingMessage] = useState('');
  const [isRecording, setIsRecording] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  // 点列は TrackBuffer が持つ（Reactの状態に入れない。仕様9章）。
  // 記録を始めるたびに作り直して、地図側のレイヤも一緒に張り替える。
  const [trackBuffer, setTrackBuffer] = useState(() => new TrackBuffer());
  const [distanceTravelled, setDistanceTravelled] = useState(0);
  const [elapsedSeconds, setElapsedSeconds] = useState(0);
  const [hasRecorded, setHasRecorded] = useState(false);
  const authRequestVersion = useRef(0);
  const socketRef = useRef<WebSocket | null>(null);
  const watchIdRef = useRef<number | null>(null);
  const pendingPointsRef = useRef<Point[]>([]);
  const routeStartedRef = useRef(false);
  const recordingRef = useRef(false);
  const connectingRef = useRef(false);
  const recordingStartedAtRef = useRef<number | null>(null);
  const stopRecordingRef = useRef<() => void>(() => {});

  useEffect(() => {
    let isMounted = true;
    const requestVersion = authRequestVersion.current;

    getCurrentSession()
      .then((session) => {
        if (isMounted && authRequestVersion.current === requestVersion) {
          setUser(session?.user ?? null);
          setRoomCode(session?.room.code ?? null);
          setRoomTitle(session?.room.title ?? null);
          setSessionLoaded(true);
        }
      })
      .catch((err: unknown) => {
        console.error('ログイン状態の確認に失敗しました:', err);
        if (isMounted && authRequestVersion.current === requestVersion) {
          setAuthError('ログイン状態を確認できませんでした。時間をおいて再読み込みしてください。');
          setSessionLoaded(true);
        }
      });

    if ('geolocation' in navigator) {
      navigator.geolocation.getCurrentPosition(
        (location) => {
          setPosition([location.coords.latitude, location.coords.longitude]);
          setHasLocation(true);
        },
        (error) => console.error('位置情報取得エラー:', error),
        { enableHighAccuracy: true }
      );
    }

    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    const activeRoomCode = currentScreen === 'solo_record' ||
      (currentScreen === 'setting' && window.location.pathname === '/solo')
      ? roomCode
      : teamRoom?.code ?? null;
    const isRoomScreen = currentScreen === 'before_record' || currentScreen === 'solo_record' ||
      (currentScreen === 'setting' && window.location.pathname !== '/result');
    if (!sessionLoaded || !activeRoomCode || !isRoomScreen) {
      setRoomSnapshot(null);
      setRoomDataError('');
      return;
    }

    let isActive = true;
    setRoomSnapshot(null);
    setRoomDataError('');
    let socket: WebSocket;
    try {
      socket = connectToRoom(activeRoomCode);
    } catch (error) {
      console.error('チーム情報の接続を開始できませんでした:', error);
      setRoomDataError('チーム情報を取得できませんでした。接続を確認してください。');
      return;
    }

    socket.onmessage = (event: MessageEvent<string>) => {
      if (!isActive) return;
      let message: unknown;
      try {
        message = JSON.parse(event.data);
      } catch (error) {
        console.error('チーム情報の応答を解析できませんでした:', error);
        setRoomDataError('チーム情報を読み取れませんでした。');
        return;
      }

      if (typeof message !== 'object' || message === null || !('t' in message)) return;
      if (message.t === 'snapshot') {
        const snapshot = parseRoomSnapshot(message);
        if (!snapshot) {
          console.error('サーバーから不正なチーム情報が返されました:', message);
          setRoomDataError('サーバーから不正なチーム情報が返されました。');
          return;
        }
        if (snapshot.room.code.toUpperCase() !== activeRoomCode.toUpperCase()) {
          console.error('別のルームの情報を受信しました:', snapshot.room.code);
          setRoomDataError('別のチーム情報を受信しました。再読み込みしてください。');
          return;
        }
        setRoomSnapshot(snapshot);
        setRoomDataError('');
      } else if (message.t === 'member' && 'member' in message) {
        const member = parseRoomMember(message.member);
        if (!member) {
          console.error('サーバーから不正なメンバー情報が返されました:', message);
          return;
        }
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          members: [
            ...snapshot.members.filter((current) => current.id !== member.id),
            member,
          ],
        }) : snapshot);
      } else if (message.t === 'leave' && 'member' in message && typeof message.member === 'number') {
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          members: snapshot.members.map((member) => member.id === message.member
            ? { ...member, online: false }
            : member),
        }) : snapshot);
      } else if (
        (message.t === 'banned' || message.t === 'unbanned') &&
        'member' in message &&
        typeof message.member === 'number'
      ) {
        const banned = message.t === 'banned';
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          members: snapshot.members.map((member) => member.id === message.member
            ? { ...member, banned }
            : member),
        }) : snapshot);
      } else if (message.t === 'route_deleted' && 'route' in message && typeof message.route === 'number') {
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          routes: snapshot.routes.filter((route) => route.id !== message.route),
        }) : snapshot);
      } else if (message.t === 'error' && 'message' in message && typeof message.message === 'string') {
        setRoomDataError(message.message);
      }
    };
    socket.onerror = () => {
      if (!isActive) return;
      console.error('チーム情報のWebSocket通信に失敗しました');
      setRoomDataError('チーム情報を取得できませんでした。接続を確認してください。');
    };
    socket.onclose = () => {
      if (isActive) {
        setRoomDataError((error) => error || 'チーム情報との接続が終了しました。再読み込みしてください。');
      }
    };

    return () => {
      isActive = false;
      socket.close();
    };
  }, [currentScreen, roomCode, sessionLoaded, teamRoom?.code]);

  useEffect(() => {
    const handlePopState = () => setCurrentScreen(getScreenFromPath(window.location.pathname));
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const navigateToScreen = (screen: Screen, replaceHistory = false) => {
    const pathByScreen: Partial<Record<Screen, string>> = {
      home: '/',
      login: '/login',
      signin: '/signin',
      solo_record: '/solo',
      result: '/result',
    };
    const path = pathByScreen[screen];
    if (path && window.location.pathname !== path) {
      if (replaceHistory) {
        window.history.replaceState({}, '', path);
      } else {
        window.history.pushState({}, '', path);
      }
    }
    setCurrentScreen(screen);
  };

  const handleAuthSuccess = (authenticatedUser: User) => {
    const requestVersion = ++authRequestVersion.current;
    setUser(authenticatedUser);
    setAuthError('');
    setSessionLoaded(false);
    getCurrentSession()
      .then((session) => {
        if (authRequestVersion.current === requestVersion) {
          setRoomCode(session?.room.code ?? null);
          setRoomTitle(session?.room.title ?? null);
          setSessionLoaded(true);
        }
      })
      .catch((err: unknown) => {
        console.error('個人ルームの取得に失敗しました:', err);
        if (authRequestVersion.current === requestVersion) {
          setAuthError('個人ルームを取得できませんでした。時間をおいて再度お試しください。');
          setSessionLoaded(true);
        }
      });
    navigateToScreen('home');
  };

  const handleTeamCreated = (room: Room) => {
    setTeamRoom(room);
  };

  const handleLogout = async () => {
    const requestVersion = ++authRequestVersion.current;
    setIsLoggingOut(true);
    setAuthError('');
    try {
      await logout();
      if (authRequestVersion.current !== requestVersion) return;
      stopRecording();
      setUser(null);
      setRoomCode(null);
      setRoomTitle(null);
      setTeamRoom(null);
      setRoomSnapshot(null);
      setRoomDataError('');
      setSessionLoaded(true);
      setRecordingError('');
      setRecordingMessage('');
      setTrackBuffer(new TrackBuffer());
      navigateToScreen('home', true);
    } catch (err) {
      console.error('ログアウトに失敗しました:', err);
      setAuthError(err instanceof Error ? err.message : 'ログアウトに失敗しました');
    } finally {
      if (authRequestVersion.current === requestVersion) {
        setIsLoggingOut(false);
      }
    }
  };

  const stopRecording = () => {
    if (watchIdRef.current !== null) {
      navigator.geolocation.clearWatch(watchIdRef.current);
      watchIdRef.current = null;
    }

    const socket = socketRef.current;
    if (socket && socket.readyState === WebSocket.OPEN && routeStartedRef.current) {
      // まとめ送信にしているので、最後の数秒分が残っている
      if (pendingPointsRef.current.length > 0) {
        try {
          sendPoints(socket, pendingPointsRef.current);
          pendingPointsRef.current = [];
        } catch (err) {
          console.error('終了前の位置情報を送信できませんでした:', err);
        }
      }
      sendStop(socket);
    }
    if (recordingStartedAtRef.current !== null) {
      setElapsedSeconds(Math.floor((Date.now() - recordingStartedAtRef.current) / 1000));
      recordingStartedAtRef.current = null;
    }
    socketRef.current = null;
    routeStartedRef.current = false;
    recordingRef.current = false;
    connectingRef.current = false;
    pendingPointsRef.current = [];
    setIsRecording(false);
    setIsConnecting(false);
    setRecordingMessage(hasRecorded ? '記録を終了しました' : '');
    if (socket && socket.readyState < WebSocket.CLOSING) {
      socket.close();
    }
  };
  stopRecordingRef.current = stopRecording;

  useEffect(() => {
    if (currentScreen !== 'solo_record') {
      stopRecordingRef.current();
      setTrackBuffer(new TrackBuffer());
    }
  }, [currentScreen]);

  useEffect(() => () => stopRecordingRef.current(), []);

  const handleGeolocationError = (error: GeolocationPositionError) => {
    const message = error.code === error.PERMISSION_DENIED
      ? '位置情報の利用が許可されていません。ブラウザーの設定を確認してください。'
      : error.code === error.POSITION_UNAVAILABLE
        ? '現在地を取得できませんでした。GPSの状態を確認してください。'
        : '位置情報の取得がタイムアウトしました。電波のよい場所で再度お試しください。';
    setRecordingError(message);
    stopRecordingRef.current();
  };

  const startLocationWatch = (buffer: TrackBuffer) => {
    if (watchIdRef.current !== null) return;
    watchIdRef.current = navigator.geolocation.watchPosition(
      (location) => {
        const point: Point = [
          location.coords.latitude,
          location.coords.longitude,
          Math.trunc(location.timestamp),
          location.coords.accuracy,
        ];
        setPosition([point[0], point[1]]);
        setHasLocation(true);

        if (!shouldAcceptPoint(point, buffer.lastRecordedPoint ?? undefined)) {
          return;
        }

        // 点列と累積距離は TrackBuffer が持つ。描画は購読側が
        // Leaflet のレイヤを直接伸ばすので、ここで状態は更新しない。
        buffer.add(point);
        // 送信は flushIntervalMilliseconds ごとにまとめて行う（仕様11章 FLUSH_MS）
        pendingPointsRef.current.push(point);
      },
      handleGeolocationError,
      { enableHighAccuracy: true, maximumAge: 0, timeout: 20_000 }
    );
  };

  useEffect(() => {
    if (!isRecording || currentScreen !== 'solo_record') return;

    // 表示の更新はここだけ。点が増えるたびには再レンダリングしない（仕様9章）
    const updateStats = () => {
      if (recordingStartedAtRef.current !== null) {
        setElapsedSeconds(Math.floor((Date.now() - recordingStartedAtRef.current) / 1000));
      }
      setDistanceTravelled(trackBuffer.totalMeters);
    };
    updateStats();
    const statsTimerId = window.setInterval(updateStats, 1000);

    // 溜めた点をまとめて送る（仕様11章 FLUSH_MS）
    const flushPoints = () => {
      const socket = socketRef.current;
      if (!socket || !routeStartedRef.current || socket.readyState !== WebSocket.OPEN) return;
      if (pendingPointsRef.current.length === 0) return;
      const queuedPoints = pendingPointsRef.current;
      pendingPointsRef.current = [];
      try {
        sendPoints(socket, queuedPoints);
      } catch (err) {
        console.error('位置情報の送信に失敗しました:', err);
        pendingPointsRef.current = queuedPoints.concat(pendingPointsRef.current);
        setRecordingError('位置情報を送信できませんでした。接続を確認して再度お試しください。');
        stopRecordingRef.current();
      }
    };
    const flushTimerId = window.setInterval(flushPoints, flushIntervalMilliseconds);

    return () => {
      window.clearInterval(statsTimerId);
      window.clearInterval(flushTimerId);
    };
  }, [currentScreen, isRecording, trackBuffer]);

  const handleStartRecording = () => {
    if (isRecording || isConnecting) {
      stopRecording();
      return;
    }
    if (!sessionLoaded) {
      setRecordingError('ログイン状態を確認中です。少し待ってから再度お試しください。');
      return;
    }
    const recordingRoomCode = currentScreen === 'solo_record' ? roomCode : teamRoom?.code;
    if (!recordingRoomCode) {
      setRecordingError('位置情報の記録にはログインが必要です。ログイン後にもう一度お試しください。');
      return;
    }
    if (!navigator.geolocation) {
      setRecordingError('このブラウザーでは位置情報を利用できません。');
      return;
    }

    setRecordingError('');
    setRecordingMessage('サーバーに接続しています...');
    const buffer = new TrackBuffer();
    setTrackBuffer(buffer);
    setDistanceTravelled(0);
    setElapsedSeconds(0);
    setHasRecorded(false);
    recordingStartedAtRef.current = null;
    pendingPointsRef.current = [];
    routeStartedRef.current = false;
    connectingRef.current = true;
    setIsConnecting(true);

    let socket: WebSocket;
    try {
      socket = connectToRoom(recordingRoomCode);
      socketRef.current = socket;
    } catch (err) {
      console.error('WebSocketの接続を開始できませんでした:', err);
      setRecordingError('サーバーに接続できませんでした。');
      stopRecording();
      return;
    }

    socket.onopen = () => {
      try {
        sendStart(socket);
        startLocationWatch(buffer);
      } catch (err) {
        console.error('記録開始メッセージを送信できませんでした:', err);
        setRecordingError('サーバーに記録開始を送信できませんでした。');
        stopRecording();
      }
    };

    socket.onmessage = (event: MessageEvent<string>) => {
      let message: unknown;
      try {
        message = JSON.parse(event.data);
      } catch (err) {
        console.error('WebSocketメッセージを解析できませんでした:', err);
        setRecordingError('サーバーから不正な応答が届きました。');
        stopRecording();
        return;
      }

      if (typeof message !== 'object' || message === null || !('t' in message)) {
        return;
      }
      if (message.t === 'error') {
        const messageText = 'message' in message && typeof message.message === 'string'
          ? message.message
          : 'サーバーが記録を受け付けませんでした。';
        setRecordingError(messageText);
        stopRecording();
        return;
      }
      if (message.t === 'route' && !routeStartedRef.current) {
        routeStartedRef.current = true;
        recordingRef.current = true;
        connectingRef.current = false;
        recordingStartedAtRef.current = Date.now();
        setIsConnecting(false);
        setIsRecording(true);
        setHasRecorded(true);
        setRecordingMessage('位置情報を記録してサーバーへ送信中です');
        try {
          sendPoints(socket, pendingPointsRef.current);
          pendingPointsRef.current = [];
        } catch (err) {
          console.error('取得済み位置情報の送信に失敗しました:', err);
          setRecordingError('取得済みの位置情報を送信できませんでした。');
          stopRecording();
        }
      }
    };

    socket.onerror = () => {
      setRecordingError('サーバーとのWebSocket通信に失敗しました。');
    };
    socket.onclose = () => {
      if (socketRef.current !== socket) return;
      if (watchIdRef.current !== null) {
        navigator.geolocation.clearWatch(watchIdRef.current);
        watchIdRef.current = null;
      }
      socketRef.current = null;
      routeStartedRef.current = false;
      const wasActive = recordingRef.current || connectingRef.current;
      recordingRef.current = false;
      connectingRef.current = false;
      if (recordingStartedAtRef.current !== null) {
        setElapsedSeconds(Math.floor((Date.now() - recordingStartedAtRef.current) / 1000));
        recordingStartedAtRef.current = null;
      }
      pendingPointsRef.current = [];
      setRecordingMessage('');
      setIsRecording(false);
      setIsConnecting(false);
      if (wasActive) {
        setRecordingMessage('');
        setRecordingError((currentError) => currentError || 'サーバーとの接続が終了しました。');
      }
    };
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
        {authError && (
          <div role="alert" className="px-4 py-2 text-sm text-red-700 bg-red-50">
            {authError}
          </div>
        )}
        {recordingError && (
          <div role="alert" className="px-4 py-2 text-sm text-red-700 bg-red-50">
            {recordingError}
          </div>
        )}
        {roomDataError && (
          <div role="alert" className="px-4 py-2 text-sm text-red-700 bg-red-50">
            {roomDataError}
          </div>
        )}

        {/* 1. ホーム画面 */}
        {currentScreen === 'home' && (
          <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column' }}>
            <div style={{ flex: 1 }}>
              <Home
                key={user?.username ?? 'anonymous'}
                username={user?.username}
                onLogout={handleLogout}
                isLoggingOut={isLoggingOut}
                onNavigateToLogin={() => navigateToScreen('login')}
                onNavigateToRegister={() => navigateToScreen('signin')}
                onStartSolo={() => navigateToScreen('solo_record')}
                onTeamCreated={handleTeamCreated}
              />
            </div>
            
            {/* 🗺️ マップ画面へ進むボタン（テスト用） */}
            <div style={{ padding: '20px', textAlign: 'center', backgroundColor: '#FAFAFA' }}>
              <button
                onClick={() => navigateToScreen('before_record')}
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

        {/* 2. 認証画面 */}
        {(currentScreen === 'login' || currentScreen === 'signin') && (
          <div style={{ flex: 1, overflowY: 'auto' }}>
            <Auth
              mode={currentScreen === 'login' ? 'login' : 'register'}
              onNavigateToLogin={() => navigateToScreen('login')}
              onNavigateToRegister={() => navigateToScreen('signin')}
              onStartWithoutAuth={() => navigateToScreen('home')}
              onAuthSuccess={handleAuthSuccess}
            />
          </div>
        )}

        {currentScreen === 'result' && (
          <div style={{ flex: 1, overflowY: 'auto' }}>
            <Result username={user?.username} roomName={roomTitle ?? undefined} />
          </div>
        )}

        {/* 4. 記録前画面 ＆ マップ */}
        {(currentScreen === 'before_record' || currentScreen === 'solo_record') && (
          <div style={{ flex: 1, position: 'relative', width: '100%', overflow: 'hidden' }}>
            <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 1 }}>
              <MapView position={position} hasLocation={hasLocation} trackBuffer={trackBuffer} trackColor={userColor} />
            </div>
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
                mode={currentScreen === 'solo_record' ? 'solo' : 'team'}
                teamName={roomSnapshot?.room.title}
                hasRoom={Boolean(currentScreen === 'solo_record' ? roomCode : teamRoom)}
                members={roomSnapshot?.members.filter((member) => !member.banned) ?? []}
                isMembersLoading={currentScreen === 'before_record' && (
                  !sessionLoaded || (Boolean(teamRoom?.code) && !roomSnapshot && !roomDataError)
                )}
                inviteUrl={teamRoom ? buildInviteUrl(teamRoom.code) : undefined}
                isRecording={isRecording}
                isConnecting={isConnecting}
                recordingMessage={recordingMessage}
                distanceTravelled={distanceTravelled}
                elapsedSeconds={elapsedSeconds}
                hasRecorded={hasRecorded}
                onBack={() => navigateToScreen('home', currentScreen === 'solo_record')}
                onStartRecording={handleStartRecording} 
                onOpenSetting={() => navigateToScreen('setting')}
              />
            </div>
          </div>
        )}

        {/* 5. 設定画面 */}
        {currentScreen === 'setting' && (
          <div style={{ flex: 1, overflowY: 'auto', zIndex: 20, backgroundColor: '#FAFAFA' }}>
            <Setting 
              teamName={roomSnapshot?.room.title}
              hasRoom={Boolean(window.location.pathname === '/solo' ? roomCode : teamRoom)}
              members={roomSnapshot?.members
                .filter((member) => !member.banned)
                .map((member) => {
                  const memberRoutes = roomSnapshot.routes.filter((route) => route.member === member.id);
                  const meters = memberRoutes.reduce((total, route) => total + route.meters, 0);
                  const elapsedMs = memberRoutes.reduce((total, route) => total + route.elapsedMs, 0);
                  return {
                    id: String(member.id),
                    name: member.displayName,
                    status: member.recording ? '記録中' : member.online ? 'オンライン' : 'オフライン',
                    distance: meters >= 1000 ? `${(meters / 1000).toFixed(2)} km` : `${Math.round(meters)} m`,
                    elapsed: elapsedMs > 0 ? formatRouteElapsedTime(elapsedMs) : undefined,
                    color: member.color,
                    isMe: member.id === roomSnapshot.you.id,
                  };
                })}
              memberCount={roomSnapshot?.members.filter((member) => !member.banned).length}
              isMembersLoading={!sessionLoaded || Boolean(teamRoom && !roomSnapshot && !roomDataError)}
              inviteUrl={teamRoom ? buildInviteUrl(teamRoom.code) : undefined}
              initialColor={userColor}
              onBack={() => navigateToScreen(window.location.pathname === '/solo' ? 'solo_record' : 'before_record')}
              onSaveAndReturn={(selectedColor) => {
                setUserColor(selectedColor);
                navigateToScreen(window.location.pathname === '/solo' ? 'solo_record' : 'before_record');
              }}
            />
          </div>
        )}

      </div>
    </div>
  );
}