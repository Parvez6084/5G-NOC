import { useEffect, useRef, useState, useCallback } from 'react';
import type { WSMessage } from '../types';

const WS_URL = 'ws://localhost:8082/ws';

export function useNocSocket(onMessage: (msg: WSMessage) => void) {
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage; // avoid stale closures without re-connecting

  const connect = useCallback(() => {
    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onclose = () => {
      setConnected(false);
      setTimeout(connect, 2000); // auto-reconnect, matches the NOC "always watching" premise
    };
    ws.onmessage = (event) => {
      const msg: WSMessage = JSON.parse(event.data);
      onMessageRef.current(msg);
    };
  }, []);

  useEffect(() => {
    connect();
    return () => wsRef.current?.close();
  }, [connect]);

  return { connected };
}