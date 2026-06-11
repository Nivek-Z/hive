// WebSocket 客户端：自动重连（指数退避）+ 心跳保活
import { getToken } from "./api.js";

export function createSocket(handlers) {
    let ws = null;
    let closedByUs = false;
    let retries = 0;
    let heartbeat = null;

    function connect() {
        const token = getToken();
        if (!token || closedByUs) return;
        const proto = location.protocol === "https:" ? "wss" : "ws";
        ws = new WebSocket(`${proto}://${location.host}/ws?token=${encodeURIComponent(token)}`);

        ws.onopen = () => {
            retries = 0;
            heartbeat = setInterval(() => send("PING", {}), 25000);
            handlers.onOpen?.();
        };
        ws.onmessage = (ev) => {
            try {
                const m = JSON.parse(ev.data);
                handlers.onEvent?.(m.type, m.data);
            } catch { /* 忽略坏帧 */ }
        };
        ws.onclose = () => {
            clearInterval(heartbeat);
            if (!closedByUs) {
                const delay = Math.min(1000 * 2 ** retries++, 15000);
                handlers.onDown?.(delay);
                setTimeout(connect, delay);
            }
        };
        ws.onerror = () => ws.close();
    }

    function send(type, data) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type, data }));
        }
    }

    function close() {
        closedByUs = true;
        clearInterval(heartbeat);
        ws?.close();
    }

    connect();
    return { send, close };
}
