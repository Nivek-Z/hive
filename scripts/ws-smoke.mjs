// M3 WebSocket 冒烟测试（Node 22+，零依赖）
// 用法：先启动应用，然后 node scripts/ws-smoke.mjs
const BASE = "http://localhost:8080/api";

async function login(username, password) {
    const r = await fetch(`${BASE}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
    });
    const j = await r.json();
    if (j.code !== 0) throw new Error("login failed: " + JSON.stringify(j));
    return j.data.token;
}

function connect(name, token) {
    return new Promise((resolve, reject) => {
        const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);
        const inbox = [];
        const waiters = [];
        ws.onmessage = (ev) => {
            const msg = JSON.parse(ev.data);
            console.log(`  [${name}] <=`, msg.type, JSON.stringify(msg.data).slice(0, 110));
            const idx = waiters.findIndex((w) => w.pred(msg));
            if (idx >= 0) waiters.splice(idx, 1)[0].resolve(msg);
            else inbox.push(msg);
        };
        ws.onopen = () =>
            resolve({
                ws,
                send: (type, data) => ws.send(JSON.stringify({ type, data })),
                wait: (pred, timeoutMs = 6000) => {
                    const idx = inbox.findIndex(pred);
                    if (idx >= 0) return Promise.resolve(inbox.splice(idx, 1)[0]);
                    return new Promise((res, rej) => {
                        const t = setTimeout(() => rej(new Error(`[${name}] 等待消息超时`)), timeoutMs);
                        waiters.push({ pred, resolve: (m) => { clearTimeout(t); res(m); } });
                    });
                },
            });
        ws.onerror = () => reject(new Error(`[${name}] ws error`));
    });
}

const ok = (label) => console.log("PASS " + label);
const rest = (path, opt = {}, token) =>
    fetch(`${BASE}${path}`, {
        ...opt,
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}`, ...(opt.headers || {}) },
    }).then((r) => r.json());

// ---- 开始 ----
const tokenA = await login("afeng", "123456");
const tokenW = await login("wengweng", "123456");

const a = await connect("阿蜂", tokenA);
const readyA = await a.wait((m) => m.type === "READY");
if (readyA.data.user.username !== "afeng") throw new Error("READY 用户不符");
ok("READY 携带用户信息与在线列表");

const w = await connect("嗡嗡", tokenW);
await w.wait((m) => m.type === "READY");
await a.wait((m) => m.type === "PRESENCE" && m.data.userId === 3 && m.data.online === true);
ok("PRESENCE 上线广播");

// 发消息 → 对方实时收到
a.send("MSG_SEND", { channelId: 2, content: "大家好，我是阿蜂", nonce: "n1" });
const msgW = await w.wait((m) => m.type === "MSG_NEW" && m.data.nonce === "n1");
if (msgW.data.message.senderNickname !== "阿蜂") throw new Error("发送者快照缺失");
ok("MSG_NEW 实时广播（含发送者快照与 nonce）");
const msgId = msgW.data.message.id;

// 正在输入
w.send("TYPING", { channelId: 2 });
await a.wait((m) => m.type === "TYPING" && m.data.channelId === 2 && m.data.userId === 3);
ok("TYPING 正在输入广播");

// 回复引用
w.send("MSG_SEND", { channelId: 2, content: "你好呀！", replyToId: msgId, nonce: "n2" });
const replyA = await a.wait((m) => m.type === "MSG_NEW" && m.data.nonce === "n2");
if (replyA.data.message.replyToId !== msgId || replyA.data.message.replySenderNickname !== "阿蜂")
    throw new Error("回复引用摘要缺失");
ok("回复引用带原消息摘要");

// REST 历史
const hist = await rest("/channels/2/messages", {}, tokenA);
if (!(hist.data.length >= 2 && hist.data[hist.data.length - 1].content === "你好呀！"))
    throw new Error("历史分页异常: " + JSON.stringify(hist.data.map((m) => m.content)));
ok("REST 历史分页（时间正序）");

// 表情回应 → 实时更新
await rest(`/messages/${msgId}/reactions`, { method: "POST", body: JSON.stringify({ emoji: "👍" }) }, tokenW);
const ru = await a.wait((m) => m.type === "REACTION_UPDATE" && m.data.messageId === msgId);
if (ru.data.reactions[0].emoji !== "👍" || ru.data.reactions[0].count !== 1)
    throw new Error("回应聚合异常");
ok("表情回应实时聚合更新");

// 撤回 → MSG_DELETED
await rest(`/messages/${msgId}`, { method: "DELETE" }, tokenA);
await w.wait((m) => m.type === "MSG_DELETED" && m.data.messageId === msgId);
ok("撤回实时广播");

// 已读 + 未读数
await rest("/channels/2/read", { method: "POST", body: JSON.stringify({ lastMessageId: 99999999 }) }, tokenW);
const detail = await rest("/hives/1", {}, tokenW);
if (!Array.isArray(detail.data.unreads)) throw new Error("detail 缺少 unreads");
ok("已读状态 + 未读统计");

// 禁言拦截
await rest("/hives/1/members/3/mute", { method: "POST", body: JSON.stringify({ minutes: 5 }) }, tokenA);
w.send("MSG_SEND", { channelId: 2, content: "我还能说话吗" });
await w.wait((m) => m.type === "ERROR" && m.data.message.includes("禁言"));
ok("禁言拦截发言（WS ERROR）");
await rest("/hives/1/members/3/mute", { method: "DELETE" }, tokenA);

// 下线 PRESENCE
w.ws.close();
await a.wait((m) => m.type === "PRESENCE" && m.data.userId === 3 && m.data.online === false);
ok("PRESENCE 下线广播");

a.ws.close();
console.log("\nALL PASS - M3 WebSocket 冒烟测试全部通过");
process.exit(0);
