// M6 成就/命令/彩蛋冒烟（Node 22+，零依赖）
const BASE = "http://localhost:8080/api";
let passed = 0;
const ok = (n) => { passed++; console.log(`PASS ${n}`); };
const fail = (n, d) => { console.error(`FAIL ${n}: ${d}`); process.exit(1); };

async function rest(token, method, path, body) {
    const res = await fetch(`${BASE}${path}`, {
        method,
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: body === undefined ? undefined : JSON.stringify(body),
    });
    return res.json();
}
async function client(username) {
    const r = await rest(null, "POST", "/auth/login", { username, password: "123456" });
    if (r.code !== 0) fail("登录", JSON.stringify(r));
    const inbox = []; const waiters = [];
    const ws = new WebSocket(`ws://localhost:8080/ws?token=${r.data.token}`);
    ws.onmessage = (ev) => {
        const m = JSON.parse(ev.data);
        inbox.push(m);
        for (let i = waiters.length - 1; i >= 0; i--) {
            if (waiters[i].pred(m)) { waiters[i].resolve(m); waiters.splice(i, 1); }
        }
    };
    await new Promise((res2, rej) => { ws.onopen = res2; ws.onerror = rej; });
    return {
        me: r.data.user, token: r.data.token, ws,
        rest: (m2, p, b) => rest(r.data.token, m2, p, b),
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

const x = await client("xiaomi");
const CH = 2; // 演示蜂巢「大厅」

// 1. 小蜜首次发言 → FIRST_BUZZ 成就实时弹出
x.send("MSG_SEND", { channelId: CH, content: "小蜜的第一条消息！", nonce: "m1" });
const ach = await x.wait((m) => m.type === "ACHIEVEMENT_UNLOCKED");
if (ach.data.code !== "FIRST_BUZZ") fail("首次发言成就", JSON.stringify(ach.data));
ok(`成就实时解锁推送（${ach.data.emoji} ${ach.data.name} +${ach.data.points}）`);

// 2. /roll 命令 → 系统消息广播
x.send("MSG_SEND", { channelId: CH, content: "/roll", nonce: "m2" });
const rollMsg = await x.wait((m) => m.type === "MSG_NEW" && m.data.message.type === "SYSTEM" && m.data.message.content.includes("🎲"));
if (!rollMsg.data.message.content.includes("掷出了")) fail("/roll", rollMsg.data.message.content);
ok(`/roll 命令 → 系统消息（${rollMsg.data.message.content.slice(0, 24)}…）`);

// 3. /fortune 每日固定
x.send("MSG_SEND", { channelId: CH, content: "/fortune", nonce: "m3" });
const f1 = await x.wait((m) => m.type === "MSG_NEW" && m.data.message.content.includes("🔮"));
ok("/fortune 今日运势");

// 4. 未知命令 → ERROR
x.send("MSG_SEND", { channelId: CH, content: "/fly", nonce: "m4" });
await x.wait((m) => m.type === "ERROR" && m.data.message.includes("/help"));
ok("未知命令返回错误提示");

// 5. 🎉 关键词 → EGG confetti 广播
x.send("MSG_SEND", { channelId: CH, content: "提前祝大家答辩顺利 🎉", nonce: "m5" });
const egg = await x.wait((m) => m.type === "EGG");
if (egg.data.effect !== "confetti") fail("EGG", JSON.stringify(egg.data));
ok("关键词彩蛋 EGG(confetti) 广播");

// 6. Konami 接口 → 隐藏成就
const k = await x.rest("POST", "/eggs/konami");
if (k.code !== 0) fail("konami", JSON.stringify(k));
const kAch = await x.wait((m) => m.type === "ACHIEVEMENT_UNLOCKED" && m.data.code === "KONAMI");
ok(`Konami 隐藏成就（${kAch.data.name}）`);

// 7. 成就墙：已解锁的隐藏成就显示真名，未解锁的打码
const wall = await x.rest("GET", "/users/me/achievements");
const konami = wall.data.find((a) => a.code === "KONAMI");
const lucky = wall.data.find((a) => a.code === "LUCKY_DICE");
if (konami.name === "？？？") fail("已解锁隐藏成就应显示真名", JSON.stringify(konami));
if (lucky.unlockedAt === null && lucky.name !== "？？？") fail("未解锁隐藏成就应打码", JSON.stringify(lucky));
ok("成就墙打码逻辑正确");

x.close();
console.log(`\nALL PASS (${passed}) - M6 后端冒烟全部通过`);
