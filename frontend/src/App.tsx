import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import Home from './ui/home';
import MapView from './ui/map';
import BeforeRecord from './ui/before_record';
import Setting from './ui/setting';
import Result from './ui/result';
import Auth from './ui/Auth';
import { buildInviteUrl, getActivityHistory, getCurrentSession, logout } from './lib/api';
import { defaultMemberColor, normalizeMemberColor } from './lib/member_colors';
import { shouldAcceptPoint } from './lib/geo';
import {
  connectToRoom,
  flushIntervalMilliseconds,
  parsePointsEvent,
  parseRecordingGateEvent,
  parseRoomMember,
  parseRoomSnapshot,
  parseRouteEndEvent,
  parseRouteStartedEvent,
  sendMember,
  sendPoints,
  sendStart,
  sendStop,
} from './lib/ws';
import { drainOutbox, enqueuePoints } from './lib/outbox';
import { TrackBuffer } from './lib/track_buffer';
import type { ActivityHistory, Point, Room, RoomMember, RoomRouteTrack, RoomSnapshot, User } from './lib/types';

type Screen = 'home' | 'before_record' | 'solo_record' | 'setting' | 'login' | 'signin' | 'result' | 'join';
const JoinScreen = lazy(() => import('./components/JoinScreen'));

function formatRouteElapsedTime(milliseconds: number): string {
  const totalMinutes = Math.floor(milliseconds / 60_000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return hours > 0 ? `${hours}時間${minutes}分` : `${minutes}分`;
}

function getScreenFromPath(path: string, hasInviteRoom: boolean): Screen {
  if (path === '/login') return 'login';
  if (path === '/signin') return 'signin';
  if (path === '/solo') return 'solo_record';
  if (path === '/result') return 'result';
  if (hasInviteRoom) return 'join';
  return 'home';
}

export default function App() {
  return (
    <MainExperience
      initialScreen={getScreenFromPath(
        window.location.pathname,
        new URLSearchParams(window.location.search).has('room'),
      )}
    />
  );
}

function MainExperience({ initialScreen }: { initialScreen: Screen }) {
  const [currentScreen, setCurrentScreen] = useState<Screen>(initialScreen);
  const [position, setPosition] = useState<[number, number]>([36.578, 136.648]);
  const [hasLocation, setHasLocation] = useState(false);
  const [locationError, setLocationError] = useState<string | null>(null);
  const [userColor, setUserColor] = useState<string>(defaultMemberColor);
  const [inviteRoomCode, setInviteRoomCode] = useState(
    () => new URLSearchParams(window.location.search).get('room')?.trim().toUpperCase() ?? null,
  );
  const [user, setUser] = useState<User | null>(null);
  const [roomCode, setRoomCode] = useState<string | null>(null);
  const [soloActivityName, setSoloActivityName] = useState<string | null>(null);
  const [teamRoom, setTeamRoom] = useState<Room | null>(null);
  const [hasTeamInviteQr, setHasTeamInviteQr] = useState(false);
  const [showTeamInviteQr, setShowTeamInviteQr] = useState(false);
  const [roomSnapshot, setRoomSnapshot] = useState<RoomSnapshot | null>(null);
  const [otherTracks, setOtherTracks] = useState<RoomRouteTrack[]>([]);
  const [roomDataError, setRoomDataError] = useState('');
  const [sessionLoaded, setSessionLoaded] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [authError, setAuthError] = useState('');
  const [activityHistory, setActivityHistory] = useState<ActivityHistory | null>(null);
  const [activityError, setActivityError] = useState('');
  const [isActivityLoading, setIsActivityLoading] = useState(false);
  const [activityReloadKey, setActivityReloadKey] = useState(0);
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
  const otherTracksRef = useRef(new Map<number, RoomRouteTrack>());
  const activeTrackBufferRef = useRef<TrackBuffer | null>(null);
  const activeRouteIdRef = useRef<number | null>(null);
  const activeRouteMemberIdRef = useRef<number | null>(null);
  const activeRouteColorRef = useRef(userColor);
  const preservedMetersRef = useRef(0);
  const socketRef = useRef<WebSocket | null>(null);
  const roomSocketRef = useRef<WebSocket | null>(null);
  const pendingMemberColorRef = useRef<string | null>(null);
  const watchIdRef = useRef<number | null>(null);
  const pendingPointsRef = useRef<Point[]>([]);
  const routeStartedRef = useRef(false);
  const routeIdRef = useRef<number | null>(null);
  const ownMemberIdRef = useRef<number | null>(null);
  const recordingRoomCodeRef = useRef<string | null>(null);
  const recordingBufferRef = useRef(trackBuffer);
  const reconnectTimerRef = useRef<number | null>(null);
  const reconnectAttemptRef = useRef(0);
  const shouldReconnectRef = useRef(false);
  const stopRequestedRef = useRef(false);
  const recordingRef = useRef(false);
  const connectingRef = useRef(false);
  const recordingStartedAtRef = useRef<number | null>(null);
  const stopRecordingRef = useRef<() => void>(() => {});

  const syncOwnMemberColor = (snapshot: RoomSnapshot) => {
    const ownMember = snapshot.members.find((member) => member.id === snapshot.you.id);
    if (!ownMember) return;
    const ownColor = normalizeMemberColor(ownMember.color);
    setUserColor(ownColor);
    activeRouteColorRef.current = ownColor;
  };

  const preserveActiveTrack = () => {
    const routeId = activeRouteIdRef.current;
    const buffer = activeTrackBufferRef.current;
    const memberId = activeRouteMemberIdRef.current;
    if (routeId === null || !buffer || memberId === null) return;

    preservedMetersRef.current += buffer.totalMeters;
    buffer.finalize();
    otherTracksRef.current.set(routeId, {
      routeId,
      memberId,
      color: activeRouteColorRef.current,
      buffer,
    });
    setOtherTracks([...otherTracksRef.current.values()]);
    setTrackBuffer(new TrackBuffer());
    activeRouteIdRef.current = null;
    activeRouteMemberIdRef.current = null;
  };

  useEffect(() => {
    let isMounted = true;
    const requestVersion = authRequestVersion.current;

    getCurrentSession()
      .then((session) => {
        if (isMounted && authRequestVersion.current === requestVersion) {
          setUser(session?.user ?? null);
          setRoomCode(session?.room.code ?? null);
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
          setLocationError(null);
        },
        (error) => {
          console.error('位置情報取得エラー:', error);
          setLocationError('現在地を取得できません。位置情報の許可を確認してください。');
        },
        { enableHighAccuracy: true }
      );
    } else {
      setLocationError('このブラウザーでは位置情報を利用できません。');
    }

    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    if (currentScreen !== 'result') return;
    if (!sessionLoaded) {
      setIsActivityLoading(true);
      return;
    }
    if (!user) {
      setActivityHistory(null);
      setActivityError('活動履歴を見るにはログインしてください。');
      setIsActivityLoading(false);
      return;
    }

    let isActive = true;
    setIsActivityLoading(true);
    setActivityHistory(null);
    setActivityError('');
    getActivityHistory()
      .then((history) => {
        if (isActive) setActivityHistory(history);
      })
      .catch((error: unknown) => {
        console.error('活動履歴を取得できませんでした:', error);
        if (isActive) {
          setActivityHistory(null);
          setActivityError(error instanceof Error ? error.message : '活動履歴を取得できませんでした');
        }
      })
      .finally(() => {
        if (isActive) setIsActivityLoading(false);
      });

    return () => {
      isActive = false;
    };
  }, [activityReloadKey, currentScreen, sessionLoaded, user]);

  useEffect(() => {
    const activeRoomCode = currentScreen === 'solo_record' ||
      (currentScreen === 'setting' && window.location.pathname === '/solo')
      ? roomCode
      : teamRoom?.code ?? null;
    const isRoomScreen = currentScreen === 'before_record' || currentScreen === 'solo_record' ||
      (currentScreen === 'setting' && window.location.pathname !== '/result');
    if (!sessionLoaded || !activeRoomCode || !isRoomScreen) {
      setRoomSnapshot(null);
      otherTracksRef.current.clear();
      setOtherTracks([]);
      setRoomDataError('');
      return;
    }

    let isActive = true;
    let myMemberId: number | null = null;
    setRoomSnapshot(null);
    otherTracksRef.current.clear();
    setOtherTracks([]);
    setRoomDataError('');
    let socket: WebSocket;
    try {
      socket = connectToRoom(activeRoomCode);
      roomSocketRef.current = socket;
    } catch (error) {
      console.error('チーム情報の接続を開始できませんでした:', error);
      setRoomDataError('チーム情報を取得できませんでした。接続を確認してください。');
      return;
    }

    socket.onopen = () => {
      if (!isActive || roomSocketRef.current !== socket) return;
      const pendingColor = pendingMemberColorRef.current;
      if (pendingColor !== null) {
        sendMember(socket, pendingColor);
        pendingMemberColorRef.current = null;
      }
    };

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
        const parsedSnapshot = parseRoomSnapshot(message);
        if (!parsedSnapshot) {
          console.error('サーバーから不正なチーム情報が返されました:', message);
          setRoomDataError('サーバーから不正なチーム情報が返されました。');
          return;
        }
        const { snapshot, routePoints } = parsedSnapshot;
        if (snapshot.room.code.toUpperCase() !== activeRoomCode.toUpperCase()) {
          console.error('別のルームの情報を受信しました:', snapshot.room.code);
          setRoomDataError('別のチーム情報を受信しました。再読み込みしてください。');
          return;
        }
        myMemberId = snapshot.you.id;
        syncOwnMemberColor(snapshot);
        otherTracksRef.current.clear();
        for (const route of snapshot.routes) {
          // The active route is drawn locally; render ended routes from the snapshot.
          if (route.member === myMemberId && route.endedAt === null) continue;
          const member = snapshot.members.find((current) => current.id === route.member);
          if (!member || member.banned) continue;
          const buffer = new TrackBuffer();
          for (const point of routePoints.get(route.id) ?? []) {
            buffer.add(point);
          }
          if (route.endedAt !== null) {
            buffer.finalize();
          }
          otherTracksRef.current.set(route.id, {
            routeId: route.id,
            memberId: route.member,
            color: member.color,
            buffer,
          });
        }
        setOtherTracks([...otherTracksRef.current.values()]);
        setRoomSnapshot(snapshot);
        setRoomDataError('');
      } else if (message.t === 'route') {
        const route = parseRouteStartedEvent(message);
        if (!route) {
          console.error('サーバーから不正な記録開始情報が返されました:', message);
          return;
        }
        if (route.member === myMemberId) {
          activeRouteIdRef.current = route.id;
          activeRouteMemberIdRef.current = route.member;
          activeRouteColorRef.current = route.color;
        }
        if (route.member !== myMemberId) {
          const current = otherTracksRef.current.get(route.id);
          otherTracksRef.current.set(route.id, current
            ? { ...current, color: route.color }
            : {
              routeId: route.id,
              memberId: route.member,
              color: route.color,
              buffer: new TrackBuffer(),
            });
          setOtherTracks([...otherTracksRef.current.values()]);
        }
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          routes: [
            ...snapshot.routes.filter((current) => current.id !== route.id),
            {
              id: route.id,
              member: route.member,
              color: route.color,
              startedAt: route.startedAt,
              endedAt: null,
              meters: 0,
              elapsedMs: 0,
            },
          ],
        }) : snapshot);
      } else if (message.t === 'member' && 'member' in message) {
        const member = parseRoomMember(message.member);
        if (!member) {
          console.error('サーバーから不正なメンバー情報が返されました:', message);
          return;
        }
        if (member.id === myMemberId) {
          const ownColor = normalizeMemberColor(member.color);
          setUserColor(ownColor);
          activeRouteColorRef.current = ownColor;
        }
        for (const [routeId, track] of otherTracksRef.current) {
          if (track.memberId === member.id && track.color !== member.color) {
            otherTracksRef.current.set(routeId, { ...track, color: member.color });
          }
        }
        setOtherTracks([...otherTracksRef.current.values()]);
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          members: [
            ...snapshot.members.filter((current) => current.id !== member.id),
            member,
          ],
        }) : snapshot);
      } else if (message.t === 'points') {
        const points = parsePointsEvent(message);
        if (!points) {
          console.error('サーバーから不正な軌跡データが返されました:', message);
          return;
        }
        const track = otherTracksRef.current.get(points.routeId);
        if (!track) return;
        for (const point of points.points) {
          track.buffer.add(point);
        }
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
      } else if (message.t === 'route_end') {
        const routeEnd = parseRouteEndEvent(message);
        if (!routeEnd) {
          console.error('サーバーから不正な記録終了情報が返されました:', message);
          return;
        }
        if (routeEnd.memberId === myMemberId && activeRouteIdRef.current === routeEnd.routeId) {
          preserveActiveTrack();
        } else {
          otherTracksRef.current.get(routeEnd.routeId)?.buffer.finalize();
        }
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          routes: snapshot.routes.map((route) => route.id === routeEnd.routeId
            ? { ...route, endedAt: routeEnd.endedAt }
            : route),
          members: snapshot.members.map((member) => member.id === routeEnd.memberId
            ? { ...member, recording: false }
            : member),
        }) : snapshot);
      } else if (message.t === 'recording_gate') {
        const enabled = parseRecordingGateEvent(message);
        if (enabled === null) return;
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          room: { ...snapshot.room, recordingEnabled: enabled },
        }) : snapshot);
      } else if (message.t === 'route_deleted' && 'route' in message && typeof message.route === 'number') {
        otherTracksRef.current.delete(message.route);
        setOtherTracks([...otherTracksRef.current.values()]);
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          routes: snapshot.routes.filter((route) => route.id !== message.route),
        }) : snapshot);
      } else if (message.t === 'closed') {
        const closedAt = 'closedAt' in message && typeof message.closedAt === 'string'
          ? message.closedAt
          : new Date().toISOString();
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          room: { ...snapshot.room, closedAt },
        }) : snapshot);
      } else if (message.t === 'reopened') {
        setRoomSnapshot((snapshot) => snapshot ? ({
          ...snapshot,
          room: { ...snapshot.room, closedAt: null },
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
      if (roomSocketRef.current === socket) roomSocketRef.current = null;
      socket.close();
    };
  }, [currentScreen, roomCode, sessionLoaded, teamRoom?.code]);

  useEffect(() => {
    const handlePopState = () => {
      const searchParams = new URLSearchParams(window.location.search);
      const roomCodeFromUrl = searchParams.get('room')?.trim().toUpperCase() ?? null;
      setInviteRoomCode(roomCodeFromUrl);
      setCurrentScreen(getScreenFromPath(window.location.pathname, searchParams.has('room')));
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const navigateToScreen = (screen: Screen, replaceHistory = false) => {
    const pathByScreen: Partial<Record<Screen, string>> = {
      home: '/',
      join: '/',
      before_record: '/',
      login: '/login',
      signin: '/signin',
      solo_record: '/solo',
      result: '/result',
    };
    const path = pathByScreen[screen];
    const shouldKeepInvite = inviteRoomCode !== null &&
      (screen === 'join' || screen === 'login' || screen === 'signin');
    const nextUrl = path && shouldKeepInvite
      ? `${path}?room=${encodeURIComponent(inviteRoomCode)}`
      : path;
    if (nextUrl && `${window.location.pathname}${window.location.search}` !== nextUrl) {
      if (replaceHistory) {
        window.history.replaceState({}, '', nextUrl);
      } else {
        window.history.pushState({}, '', nextUrl);
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
    navigateToScreen(inviteRoomCode !== null ? 'join' : 'home', inviteRoomCode !== null);
  };

  const handleTeamCreated = (room: Room) => {
    setTeamRoom(room);
    setHasTeamInviteQr(true);
    setShowTeamInviteQr(true);
    navigateToScreen('before_record', true);
  };

  const handleJoinSuccess = (room: Room, member: Pick<RoomMember, 'color'>) => {
    setTeamRoom(room);
    setHasTeamInviteQr(false);
    setShowTeamInviteQr(false);
    setUserColor(normalizeMemberColor(member.color));
    setRoomDataError('');
    setInviteRoomCode(null);
    navigateToScreen('before_record', true);
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
      setTeamRoom(null);
      setHasTeamInviteQr(false);
      setShowTeamInviteQr(false);
      setRoomSnapshot(null);
      otherTracksRef.current.clear();
      activeTrackBufferRef.current = null;
      activeRouteIdRef.current = null;
      activeRouteMemberIdRef.current = null;
      setOtherTracks([]);
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

  function stopRecording(): void {
    if (!recordingRef.current && !connectingRef.current && !stopRequestedRef.current) return;

    stopRequestedRef.current = true;
    recordingRef.current = false;
    connectingRef.current = false;
    setIsRecording(false);
    setIsConnecting(true);
    setRecordingMessage('記録を終了しています...');
    if (watchIdRef.current !== null) {
      navigator.geolocation.clearWatch(watchIdRef.current);
      watchIdRef.current = null;
    }

    const socket = socketRef.current;
    preserveActiveTrack();
    if (recordingStartedAtRef.current !== null) {
      setElapsedSeconds(Math.floor((Date.now() - recordingStartedAtRef.current) / 1000));
      recordingStartedAtRef.current = null;
    }

    const roomCodeToStop = recordingRoomCodeRef.current;
    const routeId = routeIdRef.current;
    if (socket && socket.readyState === WebSocket.OPEN && routeStartedRef.current && roomCodeToStop && routeId !== null) {
      void finishRecordingOnSocket(socket, roomCodeToStop, routeId);
      return;
    }

    if (socket && socket.readyState < WebSocket.OPEN) {
      shouldReconnectRef.current = true;
      return;
    }
    if (socket && socket.readyState === WebSocket.OPEN && routeId === null) return;

    if (roomCodeToStop) {
      shouldReconnectRef.current = true;
      void persistPendingPoints(roomCodeToStop, routeId)
        .catch((error: unknown) => {
          console.error('未送信の位置情報を保存できませんでした:', error);
          setRecordingError('未送信の位置情報を保存できませんでした。記録を終了できません。');
        })
        .finally(() => scheduleReconnect());
      return;
    }
    completeRecordingStop(socket);
  }
  stopRecordingRef.current = stopRecording;

  useEffect(() => {
    const keepRecordingOnScreen = (currentScreen === 'result' || currentScreen === 'setting') &&
      (recordingRef.current || connectingRef.current);
    if (currentScreen !== 'solo_record' && !keepRecordingOnScreen) {
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

  async function persistPendingPoints(room: string, routeId: number | null): Promise<void> {
    if (routeId === null || pendingPointsRef.current.length === 0) return;
    const points = pendingPointsRef.current;
    pendingPointsRef.current = [];
    try {
      const dropped = await enqueuePoints(room, routeId, points);
      if (dropped > 0) {
        setRecordingError(`保存容量の上限に達したため、古い位置情報 ${dropped} 点を削除しました。`);
      }
    } catch (error) {
      pendingPointsRef.current = points.concat(pendingPointsRef.current);
      throw error;
    }
  }

  async function drainQueuedPoints(socket: WebSocket, room: string, routeId: number): Promise<void> {
    await drainOutbox(room, routeId, (points) => {
      if (socketRef.current !== socket || socket.readyState !== WebSocket.OPEN) {
        throw new Error('WebSocket disconnected while draining the outbox');
      }
      sendPoints(socket, points);
    });
  }

  function completeRecordingStop(socket: WebSocket | null): void {
    const room = recordingRoomCodeRef.current;
    const routeId = routeIdRef.current;
    if (watchIdRef.current !== null) {
      navigator.geolocation.clearWatch(watchIdRef.current);
      watchIdRef.current = null;
    }
    if (recordingStartedAtRef.current !== null) {
      setElapsedSeconds(Math.floor((Date.now() - recordingStartedAtRef.current) / 1000));
      recordingStartedAtRef.current = null;
    }
    if (room && routeId !== null) {
      void persistPendingPoints(room, routeId).catch((error: unknown) => {
        console.error('終了時の未送信位置情報を保存できませんでした:', error);
        setRecordingError('終了時の未送信位置情報を保存できませんでした。');
      });
    }
    shouldReconnectRef.current = false;
    stopRequestedRef.current = false;
    recordingRef.current = false;
    connectingRef.current = false;
    routeStartedRef.current = false;
    routeIdRef.current = null;
    recordingRoomCodeRef.current = null;
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    reconnectAttemptRef.current = 0;
    if (socketRef.current === socket) socketRef.current = null;
    setIsRecording(false);
    setIsConnecting(false);
    setRecordingMessage(hasRecorded ? '記録を終了しました' : '');
    if (socket && socket.readyState < WebSocket.CLOSING) socket.close();
  }

  async function finishRecordingOnSocket(socket: WebSocket, room: string, routeId: number): Promise<void> {
    try {
      await persistPendingPoints(room, routeId);
      await drainQueuedPoints(socket, room, routeId);
    } catch (error) {
      console.error('終了前の位置情報を送信できませんでした:', error);
      setRecordingError('未送信の位置情報を保存しましたが、送信できませんでした。');
    }

    try {
      if (socketRef.current === socket && socket.readyState === WebSocket.OPEN) {
        sendStop(socket);
      }
      completeRecordingStop(socket);
    } catch (error) {
      console.error('記録終了をサーバーへ送信できませんでした:', error);
      setRecordingError('記録終了をサーバーへ送信できませんでした。再接続して終了します。');
      shouldReconnectRef.current = true;
      setIsConnecting(true);
      scheduleReconnect();
    }
  }

  function scheduleReconnect(): void {
    if (!shouldReconnectRef.current || reconnectTimerRef.current !== null) return;
    const delay = Math.min(30_000, 500 * 2 ** reconnectAttemptRef.current);
    reconnectAttemptRef.current += 1;
    setIsConnecting(true);
    setRecordingMessage('接続が切れました。再接続しています...');
    reconnectTimerRef.current = window.setTimeout(() => {
      reconnectTimerRef.current = null;
      const room = recordingRoomCodeRef.current;
      if (room && shouldReconnectRef.current) {
        connectRecordingSocket(room, recordingBufferRef.current);
      }
    }, delay);
  }

  function connectRecordingSocket(room: string, buffer: TrackBuffer): void {
    let socket: WebSocket;
    try {
      socket = connectToRoom(room);
      socketRef.current = socket;
    } catch (error) {
      console.error('WebSocketの再接続を開始できませんでした:', error);
      scheduleReconnect();
      return;
    }

    socket.onopen = () => {
      if (socketRef.current !== socket) return;
      setRecordingMessage('接続を確認しています...');
    };

    socket.onmessage = (event: MessageEvent<string>) => {
      if (socketRef.current !== socket) return;
      let message: unknown;
      try {
        message = JSON.parse(event.data);
      } catch (error) {
        console.error('WebSocketメッセージを解析できませんでした:', error);
        setRecordingError('サーバーから不正な応答が届きました。再接続しています。');
        socket.close();
        return;
      }

      if (typeof message !== 'object' || message === null || !('t' in message)) return;
      if (message.t === 'snapshot') {
        const parsedSnapshot = parseRoomSnapshot(message);
        if (!parsedSnapshot || parsedSnapshot.snapshot.room.code.toUpperCase() !== room.toUpperCase()) {
          console.error('再接続時に不正なルーム情報を受信しました:', message);
          setRecordingError('再接続時に不正なルーム情報を受信しました。');
          socket.close();
          return;
        }
        const { snapshot } = parsedSnapshot;

        ownMemberIdRef.current = snapshot.you.id;
        syncOwnMemberColor(snapshot);
        const activeRoute = snapshot.routes.find(
          (route) => route.member === snapshot.you.id && route.endedAt === null
        );
        if (activeRoute) {
          if (routeIdRef.current !== null && routeIdRef.current !== activeRoute.id) {
            const previousRouteId = routeIdRef.current;
            console.warn('再接続先の記録IDが変わりました。古いrouteのoutboxは保持します。', {
              previous: previousRouteId,
              active: activeRoute.id,
            });
            if (watchIdRef.current !== null) {
              navigator.geolocation.clearWatch(watchIdRef.current);
              watchIdRef.current = null;
            }
            void persistPendingPoints(room, previousRouteId)
              .then(() => {
                if (socketRef.current !== socket || socket.readyState !== WebSocket.OPEN) return;
                preserveActiveTrack();
                const nextBuffer = new TrackBuffer();
                recordingBufferRef.current = nextBuffer;
                activeTrackBufferRef.current = nextBuffer;
                activeRouteIdRef.current = activeRoute.id;
                activeRouteMemberIdRef.current = activeRoute.member;
                activeRouteColorRef.current = activeRoute.color;
                reconnectAttemptRef.current = 0;
                setTrackBuffer(nextBuffer);
                routeIdRef.current = activeRoute.id;
                routeStartedRef.current = true;
                if (stopRequestedRef.current) {
                  void finishRecordingOnSocket(socket, room, activeRoute.id);
                  return;
                }
                recordingRef.current = true;
                connectingRef.current = false;
                setIsRecording(true);
                setIsConnecting(false);
                setHasRecorded(true);
                setRecordingMessage('位置情報を記録してサーバーへ送信中です');
                startLocationWatch(nextBuffer);
                void resumeActiveRoute(socket, room, activeRoute.id);
              })
              .catch((error: unknown) => {
                console.error('記録ID切り替え前の位置情報を保存できませんでした:', error);
                setRecordingError('未送信の位置情報を保存できませんでした。再接続を続けます。');
                socket.close();
              });
            return;
          }
          routeIdRef.current = activeRoute.id;
          const activeBuffer = activeRouteIdRef.current === null ? new TrackBuffer() : buffer;
          if (activeBuffer !== buffer) {
            recordingBufferRef.current = activeBuffer;
            setTrackBuffer(activeBuffer);
          }
          activeTrackBufferRef.current = activeBuffer;
          activeRouteIdRef.current = activeRoute.id;
          activeRouteMemberIdRef.current = activeRoute.member;
          activeRouteColorRef.current = activeRoute.color;
          routeStartedRef.current = true;
          reconnectAttemptRef.current = 0;
          if (stopRequestedRef.current) {
            void finishRecordingOnSocket(socket, room, activeRoute.id);
            return;
          }
          recordingRef.current = true;
          connectingRef.current = false;
          setIsRecording(true);
          setIsConnecting(false);
          setRecordingMessage('位置情報を記録してサーバーへ送信中です');
          startLocationWatch(activeBuffer);
          void resumeActiveRoute(socket, room, activeRoute.id);
          return;
        }

        if (stopRequestedRef.current) {
          completeRecordingStop(socket);
          return;
        }
        if (routeIdRef.current !== null) {
          const previousRouteId = routeIdRef.current;
          if (watchIdRef.current !== null) {
            navigator.geolocation.clearWatch(watchIdRef.current);
            watchIdRef.current = null;
          }
          void persistPendingPoints(room, previousRouteId)
            .then(() => {
              if (socketRef.current !== socket || socket.readyState !== WebSocket.OPEN) return;
              if (stopRequestedRef.current) {
                completeRecordingStop(socket);
                return;
              }
              preserveActiveTrack();
              const nextBuffer = new TrackBuffer();
              recordingBufferRef.current = nextBuffer;
              activeTrackBufferRef.current = nextBuffer;
              activeRouteIdRef.current = null;
              activeRouteMemberIdRef.current = null;
              setTrackBuffer(nextBuffer);
              routeIdRef.current = null;
              sendStart(socket);
              connectingRef.current = true;
              setIsConnecting(true);
            })
            .catch((error: unknown) => {
              console.error('再接続前の位置情報を保存できませんでした:', error);
              setRecordingError('未送信の位置情報を保存できませんでした。再接続を続けます。');
              socket.close();
            });
          return;
        }

        try {
          sendStart(socket);
          connectingRef.current = true;
          setIsConnecting(true);
        } catch (error) {
          console.error('記録開始メッセージを送信できませんでした:', error);
          setRecordingError('記録開始をサーバーへ送信できませんでした。');
          socket.close();
        }
        return;
      }

      if (message.t === 'route') {
        const route = parseRouteStartedEvent(message);
        if (!route || route.member !== ownMemberIdRef.current) return;
        const startedRouteId = route.id;
        if (routeIdRef.current !== null && routeIdRef.current !== startedRouteId) {
          console.error('新しいroute開始前に古い未送信点を保存できませんでした:', {
            previous: routeIdRef.current,
            started: startedRouteId,
          });
          setRecordingError('記録IDが一致しないため、未送信データは保留されています。');
          socket.close();
          return;
        }
        routeIdRef.current = startedRouteId;
        activeRouteIdRef.current = startedRouteId;
        activeRouteMemberIdRef.current = route.member;
        activeRouteColorRef.current = route.color;
        activeTrackBufferRef.current = recordingBufferRef.current;
        routeStartedRef.current = true;
        reconnectAttemptRef.current = 0;
        if (stopRequestedRef.current) {
          void finishRecordingOnSocket(socket, room, startedRouteId);
          return;
        }
        recordingRef.current = true;
        connectingRef.current = false;
        recordingStartedAtRef.current ??= Date.now();
        setIsConnecting(false);
        setIsRecording(true);
        setHasRecorded(true);
        setRecordingMessage('位置情報を記録してサーバーへ送信中です');
        startLocationWatch(recordingBufferRef.current);
        void resumeActiveRoute(socket, room, startedRouteId);
      } else if (message.t === 'route_end' && 'route' in message && message.route === routeIdRef.current) {
        const endedRouteId = routeIdRef.current;
        void persistPendingPoints(room, endedRouteId)
          .catch((error: unknown) => {
            console.error('管理者による記録終了前の未送信位置情報を保存できませんでした:', error);
            setRecordingError('記録終了前の未送信位置情報を保存できませんでした。');
          })
          .finally(() => completeRecordingStop(socket));
      } else if (message.t === 'error') {
        const messageText = 'message' in message && typeof message.message === 'string'
          ? message.message
          : 'サーバーが記録を受け付けませんでした。';
        setRecordingError(messageText);
        shouldReconnectRef.current = false;
        completeRecordingStop(socket);
      }
    };

    socket.onerror = () => {
      if (socketRef.current === socket) {
        setRecordingMessage('接続が切れました。再接続しています...');
      }
    };
    socket.onclose = () => {
      if (socketRef.current !== socket) return;
      socketRef.current = null;
      routeStartedRef.current = false;
      if (!shouldReconnectRef.current) return;
      const routeId = routeIdRef.current;
      if (routeId !== null) {
        void persistPendingPoints(room, routeId)
          .catch((error: unknown) => {
            console.error('切断中の位置情報を保存できませんでした:', error);
            setRecordingError('切断中の位置情報を保存できませんでした。');
          })
          .finally(() => scheduleReconnect());
      } else {
        scheduleReconnect();
      }
    };
  }

  async function resumeActiveRoute(socket: WebSocket, room: string, routeId: number): Promise<void> {
    try {
      await persistPendingPoints(room, routeId);
      await drainQueuedPoints(socket, room, routeId);
    } catch (error) {
      console.error('未送信の位置情報を再送できませんでした:', error);
      setRecordingError('未送信の位置情報を再送できませんでした。再接続を続けます。');
      socket.close();
    }
  }

  useEffect(() => {
    if (
      !isRecording ||
      (currentScreen !== 'solo_record' && currentScreen !== 'result' && currentScreen !== 'setting')
    ) return;

    // 表示の更新はここだけ。点が増えるたびには再レンダリングしない（仕様9章）
    const updateStats = () => {
      if (recordingStartedAtRef.current !== null) {
        setElapsedSeconds(Math.floor((Date.now() - recordingStartedAtRef.current) / 1000));
      }
      setDistanceTravelled(preservedMetersRef.current + trackBuffer.totalMeters);
    };
    updateStats();
    const statsTimerId = window.setInterval(updateStats, 1000);

    // 溜めた点をまとめて送る（仕様11章 FLUSH_MS）
    const flushPoints = () => {
      const socket = socketRef.current;
      const room = recordingRoomCodeRef.current;
      const routeId = routeIdRef.current;
      if (!room || routeId === null) return;
      void persistPendingPoints(room, routeId)
        .then(() => {
          if (socket && routeStartedRef.current && socket.readyState === WebSocket.OPEN) {
            return drainQueuedPoints(socket, room, routeId);
          }
        })
        .catch((error: unknown) => {
          console.error('未送信の位置情報を保存または再送できませんでした:', error);
          setRecordingError('未送信の位置情報を保存または再送できませんでした。接続を確認してください。');
        });
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
    preservedMetersRef.current = 0;
    recordingBufferRef.current = buffer;
    activeTrackBufferRef.current = buffer;
    activeRouteIdRef.current = null;
    activeRouteMemberIdRef.current = null;
    activeRouteColorRef.current = userColor;
    setTrackBuffer(buffer);
    setDistanceTravelled(0);
    setElapsedSeconds(0);
    setHasRecorded(false);
    recordingStartedAtRef.current = null;
    pendingPointsRef.current = [];
    routeStartedRef.current = false;
    routeIdRef.current = null;
    ownMemberIdRef.current = null;
    recordingRoomCodeRef.current = recordingRoomCode;
    reconnectAttemptRef.current = 0;
    shouldReconnectRef.current = true;
    stopRequestedRef.current = false;
    connectingRef.current = true;
    recordingRef.current = false;
    setIsConnecting(true);
    connectRecordingSocket(recordingRoomCode, buffer);
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
                position={position}
                hasLocation={hasLocation}
                locationError={locationError}
                username={user?.username}
                onLogout={handleLogout}
                isLoggingOut={isLoggingOut}
                onNavigateToLogin={() => navigateToScreen('login')}
                onNavigateToRegister={() => navigateToScreen('signin')}
                onStartSolo={(activityName) => {
                  setSoloActivityName(activityName);
                  navigateToScreen('solo_record');
                }}
                onNavigateToResult={() => navigateToScreen('result')}
                onTeamCreated={handleTeamCreated}
              />
            </div>
          </div>
        )}

        {currentScreen === 'join' && (
          <div style={{ flex: 1, overflowY: 'auto' }}>
            <Suspense fallback={<div className="p-6 text-sm text-[#556B5D]">招待画面を読み込んでいます...</div>}>
              <JoinScreen
                roomCode={inviteRoomCode}
                user={user}
                isSessionLoading={!sessionLoaded}
                onJoinSuccess={handleJoinSuccess}
                onNavigateToLogin={() => navigateToScreen('login')}
                onNavigateToRegister={() => navigateToScreen('signin')}
                onBack={() => {
                  setInviteRoomCode(null);
                  navigateToScreen('home');
                }}
              />
            </Suspense>
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
            <Result
              username={user?.username}
              history={activityHistory}
              isLoading={isActivityLoading}
              error={activityError}
              isRecording={isRecording}
              isConnecting={isConnecting}
              onRetry={() => setActivityReloadKey((value) => value + 1)}
              onBackToHome={() => navigateToScreen('home')}
            />
          </div>
        )}

        {/* 4. 記録前画面 ＆ マップ */}
        {(currentScreen === 'before_record' || currentScreen === 'solo_record') && (
          <div style={{ flex: 1, position: 'relative', width: '100%', overflow: 'hidden' }}>
            <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 1 }}>
              <MapView
                position={position}
                hasLocation={hasLocation}
                trackBuffer={trackBuffer}
                trackColor={userColor}
                otherTracks={otherTracks.filter((track) =>
                  !roomSnapshot?.members.find((member) => member.id === track.memberId)?.banned
                )}
                inviteUrl={currentScreen === 'before_record' && hasTeamInviteQr && teamRoom
                  ? buildInviteUrl(teamRoom.code)
                  : undefined}
                showInviteQr={showTeamInviteQr}
                onShowInviteQr={() => setShowTeamInviteQr(true)}
                onDismissInviteQr={() => setShowTeamInviteQr(false)}
              />
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
                teamName={currentScreen === 'solo_record' ? soloActivityName ?? undefined : roomSnapshot?.room.title}
                hasRoom={Boolean(currentScreen === 'solo_record' ? roomCode : teamRoom)}
                members={roomSnapshot?.members.filter((member) => !member.banned) ?? []}
                roomClosed={Boolean(roomSnapshot?.room.closedAt)}
                recordingEnabled={currentScreen === 'solo_record' || Boolean(roomSnapshot?.room.recordingEnabled)}
                isOwner={currentScreen === 'solo_record' || roomSnapshot?.room.ownerMemberId === roomSnapshot?.you.id}
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
              isRecording={isRecording}
              isConnecting={isConnecting}
              initialColor={userColor}
              onBack={() => navigateToScreen(window.location.pathname === '/solo' ? 'solo_record' : 'before_record')}
              onSaveAndReturn={(selectedColor) => {
                setUserColor(selectedColor);
                pendingMemberColorRef.current = selectedColor;
                const socket = roomSocketRef.current ?? socketRef.current;
                if (socket?.readyState === WebSocket.OPEN) {
                  sendMember(socket, selectedColor);
                  pendingMemberColorRef.current = null;
                }
                navigateToScreen(window.location.pathname === '/solo' ? 'solo_record' : 'before_record');
              }}
            />
          </div>
        )}

      </div>
    </div>
  );
}
