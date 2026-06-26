// M4 好友与私聊冒烟测试（Node 22+，零依赖）
// 覆盖：申请→实时通知→接受→好友列表→开DM→DM消息广播→会话列表未读→非好友拦截→非参与者越权拦截
const BASE = "http://localhost:8080/api";
let passed = 0;

function ok(name) { passed++; console.log(`PASS ${name}`); }
function fail(name, detail) { console.error(`FAIL ${name}: ${detail}`); process.exit(1); }

async function rest(token, method, path, body) {
    const res = await fetch(`${BASE}${path}`, {
        method,
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: body === undefined ? undefined : JSON.stringify(body),
    });
    return res.json();
}

async function client(username) {
    const login = await rest(null, "POST", "/auth/login", { username, password: "123456" });
    if (login.code !== 0) fail("登录", JSON.stringify(login));
    const token = login.data.token;
    const me = login.data.user;
    const inbox = [];
    const waiters = [];
    const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);
    ws.onmessage = (ev) => {
        const m = JSON.parse(ev.data);
        inbox.push(m);
        for (let i = waiters.length - 1; i >= 0; i--) {
            if (waiters[i].pred(m)) { waiters[i].resolve(m); waiters.splice(i, 1); }
        }
    };
    await new Promise((res, rej) => { ws.onopen = res; ws.onerror = rej; });
    return {
        me, token, ws,
        rest: (method, path, body) => rest(token, method, path, body),
        send: (type, data) => ws.send(JSON.stringify({ type, data })),
        wait: (pred, ms = 5000) => {
            const hit = inbox.find(pred);
            if (hit) return Promise.resolve(hit);
            return new Promise((resolve, reject) => {
                waiters.push({ pred, resolve });
                setTimeout(() => reject(new Error("等待超时")), ms);
            });
        },
        close: () => ws.close(),
    };
}

const a = await client("afeng");
const x = await client("xiaomi");
const w = await client("wengweng");

// 清理历史关系，保证可重复运行
await a.rest("DELETE", `/friends/${x.me.id}`).catch(() => {});

// 1. 申请 + 实时通知
const reqResp = await a.rest("POST", "/friends/requests", { username: "xiaomi" });
if (reqResp.code !== 0) fail("发送好友申请", JSON.stringify(reqResp));
const evReq = await x.wait((m) => m.type === "FRIEND_EVENT" && m.data.kind === "REQUEST_NEW");
if (evReq.data.from.username !== "afeng") fail("REQUEST_NEW 载荷", JSON.stringify(evReq.data));
ok("好友申请 + FRIEND_EVENT 实时通知");

// 2. 重复申请被拦截
const dup = await a.rest("POST", "/friends/requests", { username: "xiaomi" });
if (dup.code === 0) fail("重复申请应被拦截", JSON.stringify(dup));
ok("重复申请拦截");

// 3. 接受 → 双方收到 ACCEPTED
const incoming = await x.rest("GET", "/friends/requests");
const fr = incoming.data.find((r) => r.username === "afeng");
if (!fr) fail("待处理申请列表", JSON.stringify(incoming.data));
await x.rest("POST", `/friends/requests/${fr.id}/accept`);
await a.wait((m) => m.type === "FRIEND_EVENT" && m.data.kind === "ACCEPTED" && m.data.friend.username === "xiaomi");
await x.wait((m) => m.type === "FRIEND_EVENT" && m.data.kind === "ACCEPTED" && m.data.friend.username === "afeng");
ok("接受申请，双方实时收到 ACCEPTED");

// 4. 好友列表
const friends = await a.rest("GET", "/friends");
if (!friends.data.some((f) => f.username === "xiaomi")) fail("好友列表", JSON.stringify(friends.data));
ok("好友列表正确");

// 5. 打开 DM + 幂等
const dm1 = await a.rest("POST", `/dms/${x.me.id}`);
const dm2 = await a.rest("POST", `/dms/${x.me.id}`);
if (dm1.code !== 0 || dm1.data.channelId !== dm2.data.channelId) fail("DM 幂等", JSON.stringify({ dm1, dm2 }));
const dmCh = dm1.data.channelId;
ok("打开 DM 频道（重复打开返回同一频道）");

// 6. DM 消息只广播给两位参与者
a.send("MSG_SEND", { channelId: dmCh, content: "你好小蜜，这是私聊", nonce: "dm1" });
const dmMsg = await x.wait((m) => m.type === "MSG_NEW" && m.data.nonce === "dm1");
if (dmMsg.data.message.content !== "你好小蜜，这是私聊") fail("DM 消息内容", JSON.stringify(dmMsg.data));
const leakedToW = await w
    .wait((m) => m.type === "MSG_NEW" && m.data.nonce === "dm1", 800)
    .then(() => true)
    .catch(() => false);
if (leakedToW) fail("DM 隔离", "无关用户竟收到了私聊消息");
ok("DM 消息实时送达且不外泄");

// 7. 会话列表：未读与最后一条
const dms = await x.rest("GET", "/dms");
const conv = dms.data.find((d) => d.channelId === dmCh);
if (!conv || conv.username !== "afeng" || conv.unread < 1 || !conv.lastContent.includes("私聊"))
    fail("DM 会话列表", JSON.stringify(dms.data));
ok("DM 会话列表（对方信息/未读数/最后消息）");

// 8. 非好友不能开 DM
const deny1 = await w.rest("POST", `/dms/${a.me.id}`);
if (deny1.code === 0) fail("非好友 DM 拦截", JSON.stringify(deny1));
ok("非好友发起私聊被拦截");

// 9. 非参与者读不到 DM 历史
const deny2 = await w.rest("GET", `/channels/${dmCh}/messages?limit=10`);
if (deny2.code === 0) fail("DM 历史越权拦截", JSON.stringify(deny2));
ok("非参与者读取 DM 历史被拦截（403）");

// 10. 标记已读后未读清零
await x.rest("POST", `/channels/${dmCh}/read`, { lastMessageId: dmMsg.data.message.id });
const dmsAfter = await x.rest("GET", "/dms");
if (dmsAfter.data.find((d) => d.channelId === dmCh).unread !== 0) fail("DM 已读清零", JSON.stringify(dmsAfter.data));
ok("DM 标记已读，未读清零");

a.close(); x.close(); w.close();
console.log(`\nALL PASS (${passed}) - M4 好友与私聊冒烟测试全部通过`);
process.exit(0);
