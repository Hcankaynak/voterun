import { useEffect, useRef, useState } from "react";

// Subscribes to the board's WebSocket and keeps a live board snapshot in state.
// The backend pushes the full board on connect and after every change, so the
// client never has to reconcile partial updates. Auto-reconnects on drop.
export function useBoardSocket(boardId) {
  const [board, setBoard] = useState(null);
  const [connected, setConnected] = useState(false);
  const socketRef = useRef(null);
  const reconnectRef = useRef(null);

  useEffect(() => {
    if (!boardId) return;
    let closedByUs = false;

    const connect = () => {
      const proto = window.location.protocol === "https:" ? "wss" : "ws";
      const ws = new WebSocket(
        `${proto}://${window.location.host}/ws/boards/${boardId}`
      );
      socketRef.current = ws;

      ws.onopen = () => setConnected(true);
      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "board" && msg.board) setBoard(msg.board);
        } catch {
          /* ignore malformed frames */
        }
      };
      ws.onclose = () => {
        setConnected(false);
        if (!closedByUs) {
          reconnectRef.current = setTimeout(connect, 1500);
        }
      };
      ws.onerror = () => ws.close();
    };

    connect();

    return () => {
      closedByUs = true;
      clearTimeout(reconnectRef.current);
      socketRef.current?.close();
    };
  }, [boardId]);

  return { board, connected };
}
