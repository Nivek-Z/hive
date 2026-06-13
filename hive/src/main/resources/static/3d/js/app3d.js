// ============================================================
// 蜂巢 HIVE · 白境 PORCELAIN — 主应用
// 复用后端契约层 /js/api.js 与 /js/ws.js（与经典版共享，不修改）
// 世界渲染见 world.js，运镜见 cinema.js
// ============================================================
import { api, getToken, setToken, setUnauthorizedHandler } from "/js/api.js";
import { createSocket } from "/js/ws.js";
import * as THREE from "three";
import { World } from "./world.js";
import { Cinema } from "./cinema.js";

/* ============ 小工具 ============ */
const $ = (id) => document.getElementById(id);
const esc = (s) => String(s ?? "").replace(/[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const pad2 = (n) => String(n).padStart(2, "0");
const no = (i) => pad2(i + 1);

function el(tag, cls, text) {
    const e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text !== undefined) e.textContent = text;
    return e;
}
const fmtTime = (iso) => {
    const d = new Date(iso);
    return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
};
const fmtDay = (iso) => {
    const d = new Date(iso);
    return `${d.getFullYear()} / ${pad2(d.getMonth() + 1)} / ${pad2(d.getDate())}`;
};
const sameDay = (a, b) => a && b && new Date(a).toDateString() === new Date(b).toDateString();

/** 文本渲染：转义 + 链接 + @提及 + 换行 */
function renderContent(text) {
    let h = esc(text);
    h = h.replace(/(https?:\/\/[^\s<]+)/g, '<a href="$1" target="_blank" rel="noopener">$1</a>');
    h = h.replace(/@([\w一-龥]+)/g, '<span class="mention">@$1</span>');
    return h.replace(/\n/g, "<br>");
}

const AVA_PALETTE = ["#8a8a7e", "#a39a82", "#7e8a88", "#948a96", "#8e9a7e"];
function avaEl(u, extra = "") {
    const d = el("span", "ava" + (extra ? ` ${extra}` : ""));
    if (u?.avatarUrl) {
        const img = document.createElement("img");
        img.src = u.avatarUrl; img.alt = "";
        d.appendChild(img);
    } else {
        const name = u?.nickname ?? "?";
        d.style.background = u?.avatarColor ?? AVA_PALETTE[(name.codePointAt(0) ?? 0) % AVA_PALETTE.length];
        d.textContent = [...name][0] ?? "?";
    }
    return d;
}

function toast(msg, kind = "") {
    const t = el("div", `toast ${kind}`, msg);
    $("toasts").appendChild(t);
    setTimeout(() => t.classList.add("bye"), 3200);
    setTimeout(() => t.remove(), 3700);
}

/* ============ 权限位（与后端 Permissions.java 一致） ============ */
const P = {
    ADMIN: 1, MANAGE_HIVE: 2, MANAGE_CHANNELS: 4, MANAGE_ROLES: 8,
    KICK: 16, MUTE: 32, DEL_MSG: 64, INVITE: 128,
    MENTION_ALL: 256, SEND: 512, ATTACH: 1024, REACT: 2048,
};
const PERM_DEFS = [
    { bit: P.ADMIN, label: "管理员（全部权限）" },
    { bit: P.MANAGE_HIVE, label: "管理蜂巢资料" },
    { bit: P.MANAGE_CHANNELS, label: "管理频道" },
    { bit: P.MANAGE_ROLES, label: "管理角色" },
    { bit: P.KICK, label: "踢出成员" },
    { bit: P.MUTE, label: "禁言成员" },
    { bit: P.DEL_MSG, label: "删除他人消息" },
    { bit: P.INVITE, label: "创建邀请" },
    { bit: P.MENTION_ALL, label: "@全体成员" },
    { bit: P.SEND, label: "发送消息" },
    { bit: P.ATTACH, label: "发送图片" },
    { bit: P.REACT, label: "添加表情回应" },
];
const PALETTE = ["#FFB300", "#FF7043", "#EC407A", "#AB47BC",
    "#5C6BC0", "#29B6F6", "#26A69A", "#9CCC65"];
const can = (bit) => {
    const p = state.detail?.myPermissions ?? 0;
    return (p & P.ADMIN) !== 0 || (p & bit) === bit;
};
const isOwner = () => state.detail && state.me && state.detail.ownerId === state.me.id;

/** 成员名字颜色 = 其最高 position 的非默认角色颜色 */
function roleColorOf(userId) {
    if (state.mode !== "hive" || !state.detail) return null;
    const m = state.members.find((x) => x.userId === userId);
    if (!m) return null;
    const top = (state.detail.roles ?? [])
        .filter((r) => !r.isDefault && m.roleIds?.includes(r.id))
        .sort((a, b) => b.position - a.position)[0];
    return top?.color ?? null;
}

/* ============ 状态 ============ */
const state = {
    me: null,
    mode: "hive",            // hive | home
    hives: [],
    detail: null,
    members: [],
    dms: [], friends: [], requests: [],
    currentHiveId: null,
    currentChannelId: null,
    dmPeer: null,
    messages: [], oldestId: null, hasMore: true, loadingHistory: false,
    online: new Set(),
    unreads: new Map(),      // channelId -> n（本次会话内）
    typing: new Map(),       // userId -> {name, timer}
    replyTo: null,
    socket: null,
    lastTypingSent: 0,
    chHive: new Map(),       // channelId -> hiveId
    chIndex: new Map(),      // channelId -> "01" 编号
};

/* ============ 世界与运镜 ============ */
let world, cinema;
const SHOT = { az: 0.42, elHive: 0.58, distHive: 47, elCh: 0.5, distCh: 31 };

try {
    world = new World($("gl"));
} catch (e) {
    console.error("[白境] 初始化失败", e);
    document.body.innerHTML =
        `<div style="padding:80px;font:14px/2 monospace">渲染初始化失败：${esc(e.message)}<br>` +
        `（若提示 WebGL 相关，请检查浏览器硬件加速）—— <a href="/">回到经典版</a></div>`;
    throw e;
}
cinema = new Cinema(world.camera);

/* ============ 主题：白天 / 黑夜 / 跟随系统 ============ */
const THEME_KEY = "hive3d_theme";
const themeMedia = matchMedia("(prefers-color-scheme: dark)");
const themeSetting = () => localStorage.getItem(THEME_KEY) ?? "auto";

function applyTheme(instant = false) {
    const s = themeSetting();
    const mode = s === "auto" ? (themeMedia.matches ? "dark" : "light") : s;
    document.documentElement.dataset.theme = mode;
    world.setTheme(mode, instant);
}
themeMedia.addEventListener("change", () => { if (themeSetting() === "auto") applyTheme(); });

function setThemeSetting(s) {
    localStorage.setItem(THEME_KEY, s);
    applyTheme();
}

const shotHive = (center, extra = {}) =>
    ({ center, dist: SHOT.distHive, az: SHOT.az, el: SHOT.elHive, dur: 2.1, lift: 14, targetLift: 1.5, ...extra });
const shotChannel = (tilePos, center) => ({
    center: center.clone().lerp(tilePos, 0.6),
    dist: SHOT.distCh, az: SHOT.az, el: SHOT.elCh,
    dur: 1.6, lift: 6, targetLift: 1.2, shift: 5,
});

/* ============ 3D 锚定标签层 ============ */
const labels = new Map(); // key -> {kind, v3, dom, hiveId, refId}

function addLabel(key, { kind, v3, num, text, hiveId, refId, onClick }) {
    if (labels.has(key)) labels.get(key).dom.remove();
    const dom = el("button", `tag is-${kind}`);
    dom.appendChild(el("i", "", num));
    dom.appendChild(el("span", "", text));
    const badge = el("em", "off");
    dom.appendChild(badge);
    dom.onclick = onClick;
    $("labels").appendChild(dom);
    labels.set(key, { kind, v3, dom, badge, hiveId, refId });
}

function labelBadge(key, n) {
    const L = labels.get(key);
    if (!L) return;
    L.badge.textContent = n > 99 ? "99+" : String(n);
    L.badge.classList.toggle("off", !n);
}

const _pt = { x: 0, y: 0, visible: false, dist: 0 };
function syncLabels() {
    for (const L of labels.values()) {
        world.project(L.v3, _pt);
        let show = _pt.visible && _pt.dist < 200;
        if (L.kind === "channel") show = show && state.mode === "hive" && L.hiveId === state.currentHiveId && _pt.dist < 70;
        if (L.kind === "dm") show = show && state.mode === "home" && _pt.dist < 70;
        if (L.kind === "hive") show = show && _pt.dist > 18;
        L.dom.classList.toggle("faded", !show);
        if (show) L.dom.style.transform = `translate3d(${_pt.x + 10}px, ${_pt.y - 14}px, 0)`;
        L.dom.classList.toggle("is-active",
            (L.kind === "hive" && L.refId === state.currentHiveId && state.mode === "hive") ||
            ((L.kind === "channel" || L.kind === "dm") && L.refId === state.currentChannelId));
    }
}

/* ============ 主循环 ============ */
let lastT = performance.now();
function loop(t) {
    const dt = Math.min((t - lastT) / 1000, 0.05);
    lastT = t;
    const now = t / 1000;
    cinema.update(dt, now);
    // 总览巡航中：世界焦点（阳光/蜜蜂）跟随镜头目标游走
    if (state.mode === "atlas") world.setFocus(cinema._target);
    world.update(dt, now);
    syncLabels();
    tickHover();
    requestAnimationFrame(loop);
}

/* ============ 启动 ============ */
async function boot() {
    bindAuth();
    bindPanel();
    bindComposer();
    bindCanvas();
    bindKonami();
    setUnauthorizedHandler(logout);
    addEventListener("resize", () => world.resize());
    applyTheme(true);
    $("lk-settings").onclick = openSettings;

    cinema.jumpTo({ center: new THREE.Vector3(30, 0, -10), dist: 165, az: 1.15, el: 0.95 });
    requestAnimationFrame(loop);
    setTimeout(() => $("boot-cover").classList.add("gone"), 250);
    setTimeout(() => $("boot-cover").remove(), 2200);

    if (getToken()) {
        try {
            state.me = await api.get("/users/me");
            return enterWorld();
        } catch { /* token 过期 → 登录页 */ }
    }
    showAuth();
}

function showAuth() { $("auth").classList.remove("off"); }

function logout() {
    state.socket?.close();
    setToken(null);
    location.reload();
}

/* ============ 登录 / 注册 ============ */
function bindAuth() {
    let mode = "login";
    const apply = () => {
        $("a-sub").textContent = mode === "register" ? "注册新账号 · 开始嗡嗡" : "登入白境 · 你的实时聊天社区";
        $("a-nick-field").classList.toggle("off", mode !== "register");
        $("a-submit").textContent = mode === "register" ? "注册并进入" : "进入白境";
        $("a-switch-text").textContent = mode === "register" ? "已有账号？" : "还没有账号？";
        $("a-switch").textContent = mode === "register" ? "去登录" : "立即注册";
        $("a-error").textContent = "";
    };
    $("a-switch").onclick = () => { mode = mode === "login" ? "register" : "login"; apply(); };
    $("a-form").onsubmit = async (e) => {
        e.preventDefault();
        $("a-error").textContent = "";
        $("a-submit").disabled = true;
        try {
            const body = {
                username: $("a-user").value.trim(),
                password: $("a-pass").value,
            };
            if (mode === "register") body.nickname = $("a-nick").value.trim() || body.username;
            const resp = await api.post(`/auth/${mode}`, body);
            setToken(resp.token);
            state.me = resp.user;
            $("auth").classList.add("off");
            await enterWorld();
        } catch (err) {
            $("a-error").textContent = err.message;
        } finally {
            $("a-submit").disabled = false;
        }
    };
}

/* ============ 进入世界 ============ */
async function enterWorld() {
    $("auth").classList.add("off");
    renderUserChip();
    state.socket = createSocket({
        onEvent: handleWs,
        onOpen: () => $("conn-dot").className = "conn-dot on",
        onDown: () => $("conn-dot").className = "conn-dot down",
    });

    state.hives = await api.get("/hives");
    world.buildHives(state.hives);
    world.buildHome();

    // 蜂巢标签 + 索引
    state.hives.forEach((h, i) => registerHiveLabel(h, i));
    addLabel("home", {
        kind: "hive", refId: "home", num: "00",
        text: "私域", v3: world.homeCenter().setY(4.4),
        onClick: enterHome,
    });
    renderHiveIndex();
    refreshHomeData(true);

    if (state.hives.length) {
        // 开场：高空俯瞰 → 长镜头降落到第一个蜂巢
        const c0 = world.hiveCenter(state.hives[0].id);
        cinema.jumpTo({ center: c0, dist: 185, az: SHOT.az + 0.6, el: 1.08 });
        await selectHive(state.hives[0].id, { dur: 3.4, lift: 34 });
    } else {
        cinema.fly(shotHive(world.homeCenter(), { dur: 2.6 }));
        toast("还没有加入任何蜂巢 · 可在经典版创建", "honey");
    }
}

function renderUserChip() {
    const chip = $("user-chip");
    chip.classList.remove("off");
    chip.innerHTML = "";
    chip.style.cursor = "pointer";
    chip.title = "个人资料 / 热力图";
    chip.onclick = openProfile;
    chip.appendChild(avaEl(state.me, "uc-ava"));
    chip.appendChild(el("span", "uc-name", state.me.nickname));
    const out = el("button", "uc-out", "退出");
    out.onclick = (e) => { e.stopPropagation(); logout(); };
    chip.appendChild(out);
}

/** ↑↑↓↓←→←→BA → 后端彩蛋（EGG 事件回来点亮全林荫道） */
function bindKonami() {
    const SEQ = ["ArrowUp", "ArrowUp", "ArrowDown", "ArrowDown",
        "ArrowLeft", "ArrowRight", "ArrowLeft", "ArrowRight", "b", "a"];
    let pos = 0;
    addEventListener("keydown", (e) => {
        const key = e.key.length === 1 ? e.key.toLowerCase() : e.key;
        pos = key === SEQ[pos] ? pos + 1 : (key === SEQ[0] ? 1 : 0);
        if (pos === SEQ.length) {
            pos = 0;
            api.post("/eggs/konami").catch(() => {});
        }
    });
}

/* ============ 蜂巢索引（左上编号导航） ============ */
function renderHiveIndex() {
    const box = $("hive-index");
    box.innerHTML = "";
    const at = el("button", "hx-item" + (state.mode === "atlas" ? " is-active" : ""));
    at.appendChild(el("i", "", "◎"));
    at.appendChild(el("span", "", "总览 · ATLAS"));
    at.onclick = enterAtlas;
    box.appendChild(at);
    const home = el("button", "hx-item" + (state.mode === "home" ? " is-active" : ""));
    home.appendChild(el("i", "", "00"));
    home.appendChild(el("span", "", "私域 · 好友私信"));
    const hb = homeBadgeCount();
    if (hb) home.appendChild(el("em", "", String(hb)));
    home.onclick = enterHome;
    box.appendChild(home);
    box.appendChild(el("div", "hx-rule"));
    state.hives.forEach((h, i) => {
        const active = state.mode === "hive" && h.id === state.currentHiveId;
        const item = el("button", "hx-item" + (active ? " is-active" : ""));
        item.appendChild(el("i", "", no(i)));
        const dot = el("span", "hx-dot");
        dot.style.background = h.iconColor ?? "#9a9a90";
        item.appendChild(dot);
        item.appendChild(el("span", "", h.name));
        const n = hiveUnread(h.id);
        if (n) item.appendChild(el("em", "", n > 99 ? "99+" : String(n)));
        item.onclick = () => selectHive(h.id);
        box.appendChild(item);
    });
    updateTitle();
}

/** 浏览器标签页未读角标 */
function updateTitle() {
    let n = 0;
    state.unreads.forEach((c) => { n += c; });
    n += state.dms.reduce((s, d) => s + (d.channelId === state.currentChannelId ? 0 : (d.unread ?? 0)), 0);
    document.title = n > 0 ? `(${n > 99 ? "99+" : n}) 蜂巢 HIVE · 白境` : "蜂巢 HIVE · 白境";
}

const hiveUnread = (hiveId) => {
    let n = 0;
    state.unreads.forEach((c, cid) => { if (state.chHive.get(cid) === hiveId) n += c; });
    return n;
};
const homeBadgeCount = () =>
    state.requests.length + state.dms.reduce((s, d) => s + (d.channelId === state.currentChannelId ? 0 : (d.unread ?? 0)), 0);

/* ============ 选择蜂巢 ============ */
async function selectHive(hiveId, flyOpts = {}) {
    if (state.mode === "hive" && hiveId === state.currentHiveId && !flyOpts.dur) return;
    state.mode = "hive";
    state.currentHiveId = hiveId;
    cinema.setCruise(null);
    closeChat({ fly: false });

    const center = world.hiveCenter(hiveId);
    const dur = flyOpts.dur ?? 2.1;
    // 跨巢光流：从当前焦点流向目的地，与运镜同节奏，到达时点亮点阵池
    const fromC = world.focusCenter.clone();
    if (fromC.distanceToSquared(center) > 4) {
        world.streamTo(fromC, center, dur);
        setTimeout(() => world.pulse(hiveId, 1.3), dur * 950);
    }
    world.currentHiveId = hiveId;
    cinema.fly(shotHive(center, flyOpts));
    world.setFocus(center);

    const [detail, members] = await Promise.all([
        api.get(`/hives/${hiveId}`),
        api.get(`/hives/${hiveId}/members`),
    ]);
    if (state.currentHiveId !== hiveId) return; // 期间切走了
    state.detail = detail;
    state.members = members;
    world.setHiveMembers(hiveId, members.length);
    // 周活跃度 → 点阵池常驻微光（静默失败不影响主流程）
    api.get(`/hives/${hiveId}/stats`).then((s) => {
        if (state.currentHiveId !== hiveId) return;
        const weekly = (s.daily ?? []).reduce((sum, r) => sum + (r.count ?? 0), 0);
        world.setHiveAmbient(hiveId, Math.min(weekly / 200, 1));
    }).catch(() => {});

    const texts = (detail.channels ?? []).filter((c) => c.type === "TEXT");
    world.ensureChannels(hiveId, texts);
    texts.forEach((c, i) => {
        state.chHive.set(c.id, hiveId);
        state.chIndex.set(c.id, no(i));
        addLabel(`ch:${c.id}`, {
            kind: "channel", refId: c.id, hiveId, num: no(i),
            text: c.name, v3: world.channelAnchor(c.id),
            onClick: () => openChannelById(c.id),
        });
        labelBadge(`ch:${c.id}`, state.unreads.get(c.id) ?? 0);
    });

    world.setBees(members.filter((m) => state.online.has(m.userId)).length);
    renderHiveIndex();
    renderStageCopy(no(state.hives.findIndex((h) => h.id === hiveId)), detail.name,
        `成员 ${members.length} · 频道 ${texts.length} · 在线 ${members.filter((m) => state.online.has(m.userId)).length}`);
}

function renderStageCopy(num, name, meta) {
    const sc = $("stage-copy");
    sc.classList.remove("off", "swap");
    void sc.offsetWidth; // 重触发入场动画
    sc.classList.add("swap");
    $("sc-no").textContent = num + " —";
    $("sc-name").textContent = name;
    $("sc-meta").textContent = meta;
    renderStageLinks();
}

/** 功能微链接：按上下文/权限渲染两行 */
function renderStageLinks() {
    const box = $("sc-links");
    box.innerHTML = "";
    const row = (items) => {
        const r = el("div", "sc-row");
        items.forEach(([label, fn], i) => {
            if (i) r.appendChild(el("i", "", "·"));
            const b = el("button", "", label);
            b.onclick = fn;
            r.appendChild(b);
        });
        box.appendChild(r);
    };
    if (state.mode === "hive" && state.detail) {
        const ctx = [["搜索", openSearch], ["统计", openStats]];
        if (can(P.INVITE)) ctx.push(["邀请", openInvite]);
        if (can(P.MANAGE_HIVE) || can(P.MANAGE_CHANNELS) || isOwner()) ctx.push(["管理", openHiveManage]);
        if (can(P.MANAGE_ROLES)) ctx.push(["角色", openRoles]);
        row(ctx);
    }
    row([["成就", openAchievements], ["建巢", openCreateHive], ["加入", openJoinHive]]);
}

/* ============ 打开频道 ============ */
function openChannelById(cid) {
    const owner = state.chHive.get(cid);
    if (owner && owner !== state.currentHiveId) {
        return selectHive(owner).then(() => openChannelById(cid));
    }
    const c = state.detail?.channels.find((x) => x.id === cid);
    if (c) openChannel(c);
}

async function openChannel(c) {
    state.currentChannelId = c.id;
    resetChatState();
    state.dmPeer = null;

    $("p-back").classList.add("off");
    $("p-no").textContent = state.chIndex.get(c.id) ?? "—";
    $("p-name").textContent = c.name;
    $("p-topic").textContent = c.topic ?? "";
    $("p-home").classList.add("off");
    $("p-scroll").classList.remove("off");
    $("composer").classList.remove("off");
    setComposer(can(P.SEND), `给 #${c.name} 发消息…`);
    renderMemberStrip();

    world.setActiveChannel(c.id);
    const tilePos = world.channelWorldPos(c.id);
    if (tilePos) cinema.fly(shotChannel(tilePos, world.hiveCenter(state.currentHiveId)));
    setTimeout(() => $("panel").classList.add("open"), 240);

    await loadHistory(true);
    markRead(c.id);
    state.unreads.delete(c.id);
    labelBadge(`ch:${c.id}`, 0);
    renderHiveIndex();
}

function resetChatState() {
    state.messages = [];
    state.oldestId = null;
    state.hasMore = true;
    state.replyTo = null;
    state.typing.clear();
    renderTyping();
    $("reply-line").classList.add("off");
    $("p-msgs").innerHTML = "";
}

function setComposer(enabled, placeholder) {
    const inp = $("input");
    inp.disabled = !enabled;
    inp.placeholder = enabled ? placeholder : "你没有发言权限";
}

function closeChat({ fly = true } = {}) {
    $("panel").classList.remove("open", "glassy");
    state.currentChannelId = null;
    state.dmPeer = null;
    world.setActiveChannel(null);
    if (fly && state.mode === "hive" && state.currentHiveId) {
        cinema.fly(shotHive(world.hiveCenter(state.currentHiveId)));
    }
    if (fly && state.mode === "home") {
        cinema.fly(shotHive(world.homeCenter()));
    }
}

/* ============ 私域（好友 + 私信） ============ */
async function enterHome() {
    state.mode = "home";
    cinema.setCruise(null);
    closeChat({ fly: false });
    const center = world.homeCenter();
    const fromC = world.focusCenter.clone();
    if (fromC.distanceToSquared(center) > 4) {
        world.streamTo(fromC, center, 2.4);
        setTimeout(() => world.pulse("home", 1.3), 2300);
    }
    world.currentHiveId = "home";
    cinema.fly(shotHive(center, { dur: 2.4, lift: 20 }));
    world.setFocus(center);

    await refreshHomeData();
    world.ensureDmTiles(state.dms);
    state.dms.forEach((d) => {
        state.chHive.set(d.channelId, "home");
        addLabel(`dm:${d.channelId}`, {
            kind: "dm", refId: d.channelId, num: "·",
            text: d.nickname, v3: world.channelAnchor(d.channelId) ?? center.clone().setY(3),
            onClick: () => openDm(d),
        });
        labelBadge(`dm:${d.channelId}`, d.unread ?? 0);
    });
    world.setBees(state.friends.filter((f) => state.online.has(f.userId)).length);

    renderHiveIndex();
    renderStageCopy("00", "私域",
        `好友 ${state.friends.length} · 会话 ${state.dms.length} · 申请 ${state.requests.length}`);
    renderHomePanel();
    setTimeout(() => $("panel").classList.add("open"), 240);
}

async function refreshHomeData(silent = false) {
    try {
        const [dms, friends, requests] = await Promise.all([
            api.get("/dms"), api.get("/friends"), api.get("/friends/requests"),
        ]);
        state.dms = dms; state.friends = friends; state.requests = requests;
    } catch (e) {
        if (!silent) toast(e.message, "err");
    }
    renderHiveIndex();
}

function renderHomePanel() {
    state.currentChannelId = null;
    $("p-back").classList.add("off");
    $("p-no").textContent = "00";
    $("p-name").textContent = "私域";
    $("p-topic").textContent = "好友与私信";
    $("p-scroll").classList.add("off");
    $("composer").classList.add("off");
    $("p-members").innerHTML = "";
    const box = $("p-home");
    box.classList.remove("off");
    box.innerHTML = "";

    if (state.requests.length) {
        box.appendChild(el("div", "ph-sec", `好友申请 — ${state.requests.length}`));
        for (const r of state.requests) {
            const row = el("div", "ph-row");
            row.appendChild(avaEl(r));
            row.appendChild(el("span", "ph-name", r.nickname ?? r.username));
            row.appendChild(el("span", "ph-prev", ""));
            const no_ = el("button", "ph-x", "拒绝");
            no_.onclick = async (e) => {
                e.stopPropagation();
                try { await api.del(`/friends/requests/${r.id}`); } catch (err) { toast(err.message, "err"); }
                await refreshHomeData(true);
                renderHomePanel();
            };
            row.appendChild(no_);
            const ok = el("button", "ph-badge", "接受");
            ok.onclick = async (e) => {
                e.stopPropagation();
                try { await api.post(`/friends/requests/${r.id}/accept`); toast("已接受", "honey"); } catch (err) { toast(err.message, "err"); }
                enterHome();
            };
            row.appendChild(ok);
            box.appendChild(row);
        }
    }

    box.appendChild(el("div", "ph-sec", `私信会话 — ${state.dms.length}`));
    if (!state.dms.length) box.appendChild(el("div", "ph-empty", "还没有私信 · 点好友开始"));
    for (const d of state.dms) {
        const row = el("button", "ph-row");
        row.appendChild(avaEl(d, state.online.has(d.userId) ? "online" : ""));
        row.appendChild(el("span", "ph-name", d.nickname));
        const prev = !d.lastContent ? "开始聊天吧" : d.lastContent.startsWith("/uploads/") ? "[图片]" : d.lastContent;
        row.appendChild(el("span", "ph-prev", prev));
        const unread = d.unread ?? 0;
        if (unread) row.appendChild(el("span", "ph-badge", String(unread)));
        row.onclick = () => openDm(d);
        box.appendChild(row);
    }

    const fSec = el("div", "ph-sec", `好友 — ${state.friends.length}`);
    const addBtn = el("button", "ph-add", "＋ 添加好友");
    addBtn.onclick = openAddFriend;
    fSec.appendChild(addBtn);
    box.appendChild(fSec);
    if (!state.friends.length) box.appendChild(el("div", "ph-empty", "还没有好友 · 点右上「＋ 添加好友」"));
    for (const f of state.friends) {
        const row = el("div", "ph-row");
        row.style.cursor = "pointer";
        row.appendChild(avaEl(f, state.online.has(f.userId) ? "online" : ""));
        row.appendChild(el("span", "ph-name", f.nickname));
        row.appendChild(el("span", "ph-prev", state.online.has(f.userId) ? "在线" : "离线"));
        const rm = el("button", "ph-x", "移除");
        rm.onclick = (e) => {
            e.stopPropagation();
            confirmM("移除好友", `确定移除好友 ${f.nickname} 吗？`, async () => {
                try { await api.del(`/friends/${f.userId}`); } catch (err) { toast(err.message, "err"); }
                await refreshHomeData(true);
                renderHomePanel();
            });
        };
        row.appendChild(rm);
        row.onclick = async () => {
            try {
                const { channelId } = await api.post(`/dms/${f.userId}`);
                await refreshHomeData(true);
                const conv = state.dms.find((d) => d.channelId === channelId);
                if (conv) openDm(conv);
            } catch (err) { toast(err.message, "err"); }
        };
        box.appendChild(row);
    }
}

async function openDm(conv) {
    state.currentChannelId = conv.channelId;
    resetChatState();
    state.dmPeer = conv;

    $("p-back").classList.remove("off");
    $("p-no").textContent = "DM";
    $("p-name").textContent = conv.nickname;
    $("p-topic").textContent = `@${conv.username}`;
    $("p-home").classList.add("off");
    $("p-scroll").classList.remove("off");
    $("composer").classList.remove("off");
    setComposer(true, `给 ${conv.nickname} 发消息…`);
    $("p-members").innerHTML = "";

    world.ensureDmTiles(state.dms);
    world.setActiveChannel(conv.channelId);
    const tilePos = world.channelWorldPos(conv.channelId);
    if (tilePos) cinema.fly(shotChannel(tilePos, world.homeCenter()));

    await loadHistory(true);
    markRead(conv.channelId);
    conv.unread = 0;
    labelBadge(`dm:${conv.channelId}`, 0);
    renderHiveIndex();
}

/* ============ 总览模式：高空巡航俯瞰全部群落 ============ */
async function enterAtlas() {
    if (!state.hives.length || state.mode === "atlas") return;
    state.mode = "atlas";
    cinema.setCruise(null);
    closeChat({ fly: false });
    world.currentHiveId = null;   // 总览中所有群落整簇可点（含当前蜂巢）

    const a = world.hiveCenter(state.hives[0].id);
    const b = world.hiveCenter(state.hives[state.hives.length - 1].id);
    const len = Math.max(a.distanceTo(b), 1);
    const dist = Math.min(85 + len * 0.3, 170);
    const cur = state.currentHiveId ? world.hiveCenter(state.currentHiveId) : a;

    // 入场：爬升至高空俯瞰当前蜂巢 → 落定后沿林荫道开始巡航（起点严格衔接）
    cinema.fly({ center: cur, dist, az: SHOT.az, el: 0.74, dur: 2.6, lift: 28, targetLift: 1.5 });
    const s0 = Math.min(Math.max(cur.distanceTo(a) / len, 0), 1);
    cinema.setCruise({ a, b, s0, dir: s0 > 0.5 ? -1 : 1, lift: 1.5 });

    world.setBees(Math.min(state.online.size, 14));
    renderHiveIndex();
    renderStageCopy("◎", "白境总览",
        `蜂巢 ${state.hives.length} · 在线 ${state.online.size} · 点击群落降落 / ESC 返回`);
}

/* ============ 成员微点条 ============ */
function renderMemberStrip() {
    const box = $("p-members");
    box.innerHTML = "";
    if (state.mode !== "hive") return;
    const sorted = [...state.members].sort((a, b) =>
        Number(state.online.has(b.userId)) - Number(state.online.has(a.userId)));
    for (const m of sorted.slice(0, 14)) {
        const a = avaEl(m, state.online.has(m.userId) ? "online" : "");
        a.title = m.nickname;
        a.style.cursor = "pointer";
        a.onclick = (e) => { e.stopPropagation(); showMemberPop(a, m); };
        box.appendChild(a);
    }
    const onlineN = state.members.filter((m) => state.online.has(m.userId)).length;
    box.appendChild(el("span", "pm-count", `${onlineN} ONLINE / ${state.members.length}`));
}

/* ============ 消息流 ============ */
async function loadHistory(initial) {
    if (!state.currentChannelId || state.loadingHistory) return;
    if (!initial && !state.hasMore) return;
    state.loadingHistory = true;
    const cid = state.currentChannelId;
    try {
        const qs = initial ? "?limit=50" : `?limit=50&before=${state.oldestId}`;
        const page = await api.get(`/channels/${cid}/messages${qs}`);
        if (cid !== state.currentChannelId) return;
        state.hasMore = page.length === 50;
        if (page.length) state.oldestId = page[0].id;
        if (initial) {
            // 用近 24h 历史条数播种柱体高度（消息活跃度可视化）
            const dayAgo = Date.now() - 86400000;
            const recent = page.filter((x) => new Date(x.createdAt).getTime() > dayAgo).length;
            world.seedChannelActivity(cid, recent);
            state.messages = page;
            renderMessages(true);
        } else {
            state.messages = [...page, ...state.messages];
            const sc = $("p-scroll");
            const oldH = sc.scrollHeight, oldTop = sc.scrollTop;
            renderMessages(false);
            sc.scrollTop = sc.scrollHeight - oldH + oldTop;
        }
    } catch (err) {
        toast(err.message, "err");
    } finally {
        state.loadingHistory = false;
    }
}

function renderMessages(toBottom) {
    const box = $("p-msgs");
    box.innerHTML = "";
    if (!state.messages.length) {
        const e = el("div", "empty-channel");
        e.appendChild(el("div", "eh", "⬡"));
        e.appendChild(el("div", "", "这里静悄悄的 · 发出第一声嗡嗡"));
        box.appendChild(e);
        return;
    }
    let prev = null;
    for (const m of state.messages) {
        if (!prev || !sameDay(prev.createdAt, m.createdAt)) {
            box.appendChild(el("div", "date-rule", fmtDay(m.createdAt)));
        }
        box.appendChild(msgRow(m, prev));
        prev = m;
    }
    if (toBottom) {
        const sc = $("p-scroll");
        sc.scrollTop = sc.scrollHeight;
    }
}

function appendMessage(m) {
    const box = $("p-msgs");
    box.querySelector(".empty-channel")?.remove();
    const prev = state.messages.at(-2) ?? null;
    if (!prev || !sameDay(prev.createdAt, m.createdAt)) {
        box.appendChild(el("div", "date-rule", fmtDay(m.createdAt)));
    }
    const row = msgRow(m, prev);
    row.classList.add("fresh");
    box.appendChild(row);
    const sc = $("p-scroll");
    if (sc.scrollHeight - sc.scrollTop - sc.clientHeight < 180 || m.senderId === state.me.id) {
        sc.scrollTop = sc.scrollHeight;
    }
}

function msgRow(m, prev) {
    if (m.type === "SYSTEM") {
        const sys = el("div", "msg-sys", m.content);
        sys.dataset.id = m.id ?? "";
        return sys;
    }
    const grouped = prev && prev.type !== "SYSTEM" && prev.senderId === m.senderId &&
        !m.replyToId && sameDay(prev.createdAt, m.createdAt) &&
        (new Date(m.createdAt) - new Date(prev.createdAt)) < 5 * 60 * 1000;

    const row = el("div", "msg" + (grouped ? "" : " first") +
        (m.pending ? " pending" : "") + (m.failed ? " failed" : ""));
    row.dataset.id = m.id ?? "";
    if (m.nonce) row.dataset.nonce = m.nonce;

    if (m.replyToId) {
        const q = el("div", "msg-quote");
        const snip = (m.replyContent ?? "").startsWith("/uploads/") ? "[图片]" : (m.replyContent ?? "");
        q.innerHTML = `↩ <b>${esc(m.replySenderNickname ?? "未知")}</b> — ${esc(snip)}`;
        row.appendChild(q);
    }
    if (!grouped) {
        const meta = el("div", "msg-meta");
        const author = el("span", "msg-author", m.senderNickname ?? "未知");
        const rc = roleColorOf(m.senderId);
        if (rc) author.style.color = rc;
        meta.appendChild(author);
        meta.appendChild(el("span", "msg-time", fmtTime(m.createdAt)));
        row.appendChild(meta);
    }
    if (m.type === "IMAGE") {
        const img = el("img", "msg-img");
        img.loading = "lazy";
        img.decoding = "async";
        img.src = m.content; img.alt = "图片";
        img.onclick = () => { $("lightbox-img").src = m.content; $("lightbox").classList.remove("off"); };
        row.appendChild(img);
    } else {
        const c = el("div", "msg-content");
        c.innerHTML = renderContent(m.content);
        row.appendChild(c);
    }
    if (m.reactions?.length) {
        const rs = el("div", "msg-reacts");
        for (const r of m.reactions) {
            const mine = r.userIds?.includes(state.me.id) ?? false;
            const count = r.count ?? r.userIds?.length ?? 0;
            const chip = el("button", "react-chip" + (mine ? " mine" : ""), `${r.emoji} ${count}`);
            chip.onclick = () => toggleReaction(m, r.emoji, mine);
            rs.appendChild(chip);
        }
        row.appendChild(rs);
    }
    // 悬停微操作
    if (m.id) {
        const acts = el("div", "msg-acts");
        if (state.mode !== "hive" || can(P.REACT)) {
            const rx = el("button", "", "回应");
            rx.onclick = (ev) => { ev.stopPropagation(); showEmojiPop(rx, m); };
            acts.appendChild(rx);
        }
        const rep = el("button", "", "回复");
        rep.onclick = () => {
            state.replyTo = m;
            $("reply-text").textContent = `↩ ${m.senderNickname} — ${m.type === "IMAGE" ? "[图片]" : (m.content ?? "").slice(0, 42)}`;
            $("reply-line").classList.remove("off");
            $("input").focus();
        };
        acts.appendChild(rep);
        if (m.senderId === state.me.id || (state.mode === "hive" && can(P.DEL_MSG))) {
            const del = el("button", "", "撤回");
            del.onclick = async () => {
                try { await api.del(`/messages/${m.id}`); } catch (err) { toast(err.message, "err"); }
            };
            acts.appendChild(del);
        }
        row.appendChild(acts);
    }
    return row;
}

async function toggleReaction(m, emoji, mine) {
    try {
        if (mine) await api.del(`/messages/${m.id}/reactions/${encodeURIComponent(emoji)}`);
        else await api.post(`/messages/${m.id}/reactions`, { emoji });
    } catch (err) { toast(err.message, "err"); }
}

/* ============ 表情回应选择器 ============ */
const EMOJIS = ["👍", "❤️", "😂", "🎉", "🐝", "😮"];

function showEmojiPop(anchor, m) {
    const pop = $("emoji-pop");
    pop.innerHTML = "";
    for (const e of EMOJIS) {
        const b = el("button", "", e);
        b.onclick = () => {
            hideEmojiPop();
            const mine = m.reactions?.find((r) => r.emoji === e && r.userIds?.includes(state.me.id));
            toggleReaction(m, e, !!mine);
        };
        pop.appendChild(b);
    }
    pop.classList.remove("off");
    const r = anchor.getBoundingClientRect();
    pop.style.left = `${Math.max(12, Math.min(r.left - 90, innerWidth - 250))}px`;
    pop.style.top = `${r.top - 48}px`;
    setTimeout(() => addEventListener("click", hideEmojiPop, { once: true }), 0);
}
function hideEmojiPop() { $("emoji-pop").classList.add("off"); }

const markRead = (cid) => api.post(`/channels/${cid}/read`).catch(() => {});

/* ============ 输入区 ============ */
function bindComposer() {
    const inp = $("input");
    const autosize = () => { inp.style.height = "auto"; inp.style.height = `${Math.min(inp.scrollHeight, 110)}px`; };
    inp.addEventListener("input", () => {
        autosize();
        const now = Date.now();
        if (now - state.lastTypingSent > 2500 && state.currentChannelId) {
            state.lastTypingSent = now;
            state.socket?.send("TYPING", { channelId: state.currentChannelId });
        }
    });
    inp.addEventListener("keydown", (e) => {
        if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendMessage(); }
    });
    $("send").onclick = sendMessage;
    $("reply-cancel").onclick = () => { state.replyTo = null; $("reply-line").classList.add("off"); };

    // 发送图片
    $("attach").onclick = () => {
        if (!state.currentChannelId) return;
        if (state.mode === "hive" && !can(P.ATTACH)) return toast("你没有发送图片的权限", "err");
        $("file").click();
    };
    $("file").onchange = async (e) => {
        const f = e.target.files[0];
        e.target.value = "";
        if (!f || !state.currentChannelId) return;
        try {
            const { url } = await api.upload(f);
            state.socket.send("MSG_SEND", {
                channelId: state.currentChannelId, content: url,
                type: "IMAGE", nonce: `img${Date.now()}`,
            });
        } catch (err) { toast(err.message, "err"); }
    };
}

function sendMessage() {
    const inp = $("input");
    const text = inp.value.trim();
    if (!text || !state.currentChannelId || inp.disabled) return;
    inp.value = "";
    inp.style.height = "auto";

    if (text.startsWith("/")) { // 斜杠命令：结果由系统消息广播回来
        state.socket.send("MSG_SEND", { channelId: state.currentChannelId, content: text });
        return;
    }
    const nonce = `n${Date.now()}${Math.random().toString(36).slice(2, 6)}`;
    const temp = {
        id: null, nonce,
        channelId: state.currentChannelId,
        senderId: state.me.id,
        senderNickname: state.me.nickname,
        senderAvatarColor: state.me.avatarColor,
        type: "TEXT", content: text,
        replyToId: state.replyTo?.id ?? null,
        replySenderNickname: state.replyTo?.senderNickname,
        replyContent: state.replyTo?.content,
        createdAt: new Date().toISOString(),
        reactions: [], pending: true,
    };
    state.messages.push(temp);
    appendMessage(temp);
    state.socket.send("MSG_SEND", {
        channelId: temp.channelId, content: text, replyToId: temp.replyToId, nonce,
    });
    state.replyTo = null;
    $("reply-line").classList.add("off");
}

/* ============ 正在输入 ============ */
function renderTyping() {
    const box = $("p-typing");
    const names = [...state.typing.values()].map((t) => t.name);
    box.innerHTML = names.length
        ? `${esc(names.slice(0, 3).join("、"))} 正在输入<span class="dots"></span>`
        : "";
}

/* ============ WebSocket 事件 ============ */
function handleWs(type, d) {
    switch (type) {
        case "READY":
            state.online = new Set(d.onlineUserIds ?? []);
            renderMemberStrip();
            break;

        case "MSG_NEW": {
            const m = d.message;
            // 输入提示清除
            const t = state.typing.get(m.senderId);
            if (t) { clearTimeout(t.timer); state.typing.delete(m.senderId); renderTyping(); }

            // 世界脉冲：消息落在哪个群落就点亮哪片点阵；对应柱体当场生长一点
            const owner = state.chHive.get(m.channelId);
            if (owner) world.pulse(owner === "home" ? "home" : owner, 1);
            world.bumpChannelActivity(m.channelId);

            if (d.nonce) {
                const i = state.messages.findIndex((x) => x.nonce === d.nonce);
                if (i >= 0) { state.messages[i] = m; renderMessages(true); return; }
            }
            if (m.channelId === state.currentChannelId) {
                state.messages.push(m);
                appendMessage(m);
                if (document.hasFocus()) markRead(m.channelId);
            } else {
                bumpUnread(m.channelId);
            }
            break;
        }

        case "MSG_DELETED":
            if (d.channelId === state.currentChannelId) {
                state.messages = state.messages.filter((x) => x.id !== d.messageId);
                renderMessages(false);
            }
            break;

        case "REACTION_UPDATE":
            if (d.channelId === state.currentChannelId) {
                const m = state.messages.find((x) => x.id === d.messageId);
                if (m) { m.reactions = d.reactions ?? []; renderMessages(false); }
            }
            break;

        case "TYPING": {
            if (d.userId === state.me.id) break;
            world.bob(d.channelId, performance.now() / 1000);
            if (d.channelId !== state.currentChannelId) break;
            const name = d.nickname || state.members.find((x) => x.userId === d.userId)?.nickname || "有人";
            clearTimeout(state.typing.get(d.userId)?.timer);
            state.typing.set(d.userId, {
                name,
                timer: setTimeout(() => { state.typing.delete(d.userId); renderTyping(); }, 3500),
            });
            renderTyping();
            break;
        }

        case "PRESENCE":
            d.online ? state.online.add(d.userId) : state.online.delete(d.userId);
            renderMemberStrip();
            if (state.mode === "hive" && state.detail) {
                world.setBees(state.members.filter((m) => state.online.has(m.userId)).length);
            }
            // 列表视图才整体重绘，避免打断正在进行的私聊
            if (state.mode === "home" && !state.currentChannelId) renderHomePanel();
            break;

        case "FRIEND_EVENT":
            if (d.kind === "REQUEST_NEW") toast(`${d.from.nickname} 请求加你为好友`, "honey");
            if (d.kind === "ACCEPTED") toast(`你和 ${d.friend.nickname} 已成为好友`, "honey");
            refreshHomeData(true).then(() => {
                if (state.mode === "home" && !state.currentChannelId) renderHomePanel();
            });
            break;

        case "ACHIEVEMENT_UNLOCKED": {
            const a = el("div", "ach");
            a.innerHTML = `<span class="ach-emoji">${esc(d.emoji)}</span>`
                + `<span><b>成就解锁 · ${esc(d.name)}</b><i>${esc(d.description)}</i></span>`
                + `<span class="ach-pt">+${d.points ?? 0}</span>`;
            document.body.appendChild(a);
            world.pulse(state.mode === "home" ? "home" : state.currentHiveId, 2);
            setTimeout(() => a.classList.add("bye"), 4200);
            setTimeout(() => a.remove(), 4700);
            break;
        }

        case "EGG":
            // 彩蛋：整条林荫道一起发光
            for (const h of state.hives) world.pulse(h.id, 2.2);
            world.pulse("home", 2.2);
            toast(d.effect === "bees" ? "🐝 蜂群掠过白境" : "🎉 彩蛋触发", "honey");
            break;

        case "HIVE_EVENT":
            refreshCurrentHive();
            break;

        case "ERROR":
            toast(d.message ?? "操作失败", "err");
            state.messages.filter((x) => x.pending).forEach((x) => { x.pending = false; x.failed = true; });
            renderMessages(false);
            break;
    }
}

function bumpUnread(cid) {
    if (state.chHive.get(cid) === "home" || !state.chHive.has(cid)) {
        // 私信或未知频道 → 刷新私域角标
        const conv = state.dms.find((x) => x.channelId === cid);
        if (conv) { conv.unread = (conv.unread ?? 0) + 1; labelBadge(`dm:${cid}`, conv.unread); }
        else refreshHomeData(true);
    } else {
        state.unreads.set(cid, (state.unreads.get(cid) ?? 0) + 1);
        labelBadge(`ch:${cid}`, state.unreads.get(cid));
    }
    renderHiveIndex();
}

async function refreshCurrentHive() {
    if (state.mode !== "hive" || !state.currentHiveId) return;
    const hiveId = state.currentHiveId;
    try {
        const [detail, members] = await Promise.all([
            api.get(`/hives/${hiveId}`), api.get(`/hives/${hiveId}/members`),
        ]);
        if (state.currentHiveId !== hiveId) return;
        state.detail = detail; state.members = members;
        const texts = (detail.channels ?? []).filter((c) => c.type === "TEXT");
        world.rebuildChannels(hiveId, texts);
        // 标签重建
        for (const [key, L] of [...labels]) {
            if (L.kind === "channel" && L.hiveId === hiveId) { L.dom.remove(); labels.delete(key); }
        }
        texts.forEach((c, i) => {
            state.chHive.set(c.id, hiveId);
            state.chIndex.set(c.id, no(i));
            addLabel(`ch:${c.id}`, {
                kind: "channel", refId: c.id, hiveId, num: no(i),
                text: c.name, v3: world.channelAnchor(c.id),
                onClick: () => openChannelById(c.id),
            });
        });
        renderMemberStrip();
    } catch { /* 静默 */ }
}

/* ============ 画布交互 ============ */
const pointer = { nx: 0, ny: 0, overCanvas: false, dirty: false };
let hoverable = false;

/** 悬停拾取节流到每帧一次（pointermove 可达 120+Hz，逐事件射线检测会拖累运镜） */
function tickHover() {
    if (!pointer.dirty) return;
    pointer.dirty = false;
    const cv = $("gl");
    if (!pointer.overCanvas) {
        world.setHover(null);
        if (hoverable) { hoverable = false; cv.style.cursor = "default"; }
        return;
    }
    const hit = world.pick(pointer.nx, pointer.ny);
    const h = !!hit?.kind;
    if (h !== hoverable) { hoverable = h; cv.style.cursor = h ? "pointer" : "default"; }
    world.setHover((hit?.kind === "channel" || hit?.kind === "dm") ? hit.channelId : null);
}

function bindCanvas() {
    const cv = $("gl");
    addEventListener("pointermove", (e) => {
        cinema.setMouse((e.clientX / innerWidth) * 2 - 1, (e.clientY / innerHeight) * 2 - 1);
        pointer.nx = (e.clientX / innerWidth) * 2 - 1;
        pointer.ny = -(e.clientY / innerHeight) * 2 + 1;
        pointer.overCanvas = e.target === cv;
        pointer.dirty = true;
    });
    cv.addEventListener("click", (e) => {
        const hit = world.pick((e.clientX / innerWidth) * 2 - 1, -(e.clientY / innerHeight) * 2 + 1);
        if (!hit?.kind) return;
        if (hit.kind === "hive") selectHive(hit.hiveId);
        else if (hit.kind === "channel") openChannelById(hit.channelId);
        else if (hit.kind === "home") enterHome();
        else if (hit.kind === "dm") {
            const conv = state.dms.find((d) => d.channelId === hit.channelId);
            if (conv) { if (state.mode !== "home") enterHome().then(() => openDm(conv)); else openDm(conv); }
        }
    });
}

/* ============ 面板按钮 ============ */
function bindPanel() {
    // 磨砂滤镜在滑入动画结束后才开启：backdrop-filter 随动画逐帧重算是顿挫大头
    const panel = $("panel");
    panel.addEventListener("transitionend", (e) => {
        if (e.propertyName === "transform" && panel.classList.contains("open")) {
            panel.classList.add("glassy");
        }
    });
    $("p-close").onclick = () => closeChat();
    $("p-back").onclick = () => { renderHomePanel(); world.setActiveChannel(null); cinema.fly(shotHive(world.homeCenter())); };
    addEventListener("keydown", (e) => {
        if (e.key === "Escape") {
            if (!$("lightbox").classList.contains("off")) return $("lightbox").classList.add("off");
            if ($("m-overlay")) return closeModal();
            if ($("panel").classList.contains("open")) return closeChat();
            // 面板已关：Esc 在蜂巢与总览之间切换
            if (state.mode === "hive" && state.currentHiveId) return enterAtlas();
            if (state.mode === "atlas" && state.currentHiveId) return selectHive(state.currentHiveId);
        }
    });
    $("lightbox").onclick = () => $("lightbox").classList.add("off");
    $("p-scroll").addEventListener("scroll", () => {
        if ($("p-scroll").scrollTop < 60) loadHistory(false);
    });
    addEventListener("focus", () => {
        if (state.currentChannelId) markRead(state.currentChannelId);
    });
}

/* ============ 蜂巢标签注册（开场 / 新建 / 加入后复用） ============ */
function registerHiveLabel(h, i) {
    addLabel(`hive:${h.id}`, {
        kind: "hive", refId: h.id, num: no(i),
        text: h.name, v3: world.hiveAnchor(h.id),
        onClick: () => selectHive(h.id),
    });
    // 编号染上蜂巢身份色
    const L = labels.get(`hive:${h.id}`);
    if (L && h.iconColor) L.dom.querySelector("i").style.color = h.iconColor;
}

/** 新建/加入蜂巢后：重建群落与标签并飞过去 */
async function adoptNewHive(preferId) {
    state.hives = await api.get("/hives");
    world.buildHives(state.hives);
    state.hives.forEach((h, i) => registerHiveLabel(h, i));
    renderHiveIndex();
    const target = state.hives.find((x) => x.id === preferId) ?? state.hives.at(-1);
    if (target) await selectHive(target.id);
}

/* ============ 白瓷弹层 ============ */
function modal({ title, sub, build }) {
    closeModal();
    const ov = el("div", "m-overlay");
    ov.id = "m-overlay";
    const card = el("div", "m-card");
    const head = el("div", "m-head");
    head.appendChild(el("span", "m-title", title));
    head.appendChild(el("span", "m-sub", sub ?? ""));
    const x = el("button", "m-x", "✕");
    x.onclick = closeModal;
    head.appendChild(x);
    card.appendChild(head);
    const body = el("div", "m-body");
    card.appendChild(body);
    ov.appendChild(card);
    ov.onclick = (e) => { if (e.target === ov) closeModal(); };
    $("modal-root").appendChild(ov);
    build(body);
}
function closeModal() { $("m-overlay")?.remove(); }

/* ============ 搜索 ============ */
function openSearch() {
    if (state.mode !== "hive" || !state.currentHiveId) return toast("先进入一个蜂巢再搜索");
    modal({ title: "搜索", sub: "NGRAM 中文全文索引", build: (body) => {
        const inp = el("input", "m-input");
        inp.placeholder = "关键词 · 回车搜索";
        body.appendChild(inp);
        const box = el("div");
        body.appendChild(box);
        inp.onkeydown = async (e) => {
            if (e.key !== "Enter") return;
            const q = inp.value.trim();
            if (!q) return;
            box.innerHTML = "";
            box.appendChild(el("div", "hit-empty", "检索中 ···"));
            try {
                const hits = await api.get(`/search/messages?hiveId=${state.currentHiveId}&q=${encodeURIComponent(q)}`);
                box.innerHTML = "";
                if (!hits.length) return box.appendChild(el("div", "hit-empty", "没有找到相关消息"));
                for (const h of hits) {
                    const row = el("button", "hit");
                    const meta = el("div", "hit-meta");
                    meta.appendChild(el("b", "", `⬡ ${h.channelName ?? ""}`));
                    meta.appendChild(el("span", "", h.senderNickname ?? "系统"));
                    meta.appendChild(el("span", "", fmtTime(h.createdAt)));
                    row.appendChild(meta);
                    const c = el("div", "hit-content");
                    c.innerHTML = renderContent(h.content ?? "");
                    row.appendChild(c);
                    row.onclick = () => { closeModal(); openChannelById(h.channelId); };
                    box.appendChild(row);
                }
            } catch (err) { toast(err.message, "err"); }
        };
        setTimeout(() => inp.focus(), 80);
    } });
}

/* ============ 成就墙 ============ */
async function openAchievements() {
    let list;
    try { list = await api.get("/users/me/achievements"); } catch (err) { return toast(err.message, "err"); }
    const unlocked = list.filter((a) => a.unlockedAt);
    const pts = unlocked.reduce((s, a) => s + (a.points ?? 0), 0);
    modal({ title: "成就", sub: "有些成就藏得很深", build: (body) => {
        const sum = el("div", "aw-sum");
        sum.innerHTML = `已解锁 <b>${unlocked.length}</b> / ${list.length} · 点数 <b>${pts}</b>`;
        body.appendChild(sum);
        const grid = el("div", "aw-grid");
        for (const a of list) {
            const card = el("div", "aw-card" + (a.unlockedAt ? "" : " locked"));
            card.appendChild(el("span", "e", a.emoji));
            const t = el("div");
            t.appendChild(el("b", "", a.name));
            t.appendChild(el("i", "", a.unlockedAt ? `${a.description} · ${fmtDay(a.unlockedAt)}` : a.description));
            card.appendChild(t);
            card.appendChild(el("span", "pt", `+${a.points ?? 0}`));
            grid.appendChild(card);
        }
        body.appendChild(grid);
    } });
}

/* ============ 新建 / 加入 / 邀请 ============ */
function openCreateHive() {
    modal({ title: "新建蜂巢", sub: "立起一簇新的白瓷柱", build: (body) => {
        const inp = el("input", "m-input");
        inp.placeholder = "蜂巢名称";
        inp.maxLength = 30;
        body.appendChild(inp);
        const btn = el("button", "m-btn", "创 建");
        body.appendChild(btn);
        btn.onclick = async () => {
            const name = inp.value.trim();
            if (!name) return;
            btn.disabled = true;
            try {
                const h = await api.post("/hives", { name });
                closeModal();
                await adoptNewHive(h?.id ?? h);
                toast("蜂巢已立起", "honey");
            } catch (err) { toast(err.message, "err"); btn.disabled = false; }
        };
        setTimeout(() => inp.focus(), 80);
    } });
}

function openJoinHive() {
    modal({ title: "凭邀请码加入", sub: "输入伙伴给你的代码", build: (body) => {
        const inp = el("input", "m-input");
        inp.placeholder = "邀请码";
        body.appendChild(inp);
        const btn = el("button", "m-btn", "加 入");
        body.appendChild(btn);
        btn.onclick = async () => {
            const code = inp.value.trim();
            if (!code) return;
            btn.disabled = true;
            try {
                const r = await api.post(`/invites/${encodeURIComponent(code)}/join`);
                closeModal();
                await adoptNewHive(r?.hiveId ?? r?.id);
                toast("已加入蜂巢", "honey");
            } catch (err) { toast(err.message, "err"); btn.disabled = false; }
        };
        setTimeout(() => inp.focus(), 80);
    } });
}

async function openInvite() {
    if (state.mode !== "hive" || !state.currentHiveId) return toast("先进入一个蜂巢");
    try {
        const invites = await api.get(`/hives/${state.currentHiveId}/invites`);
        const inv = invites[0] ?? await api.post(`/hives/${state.currentHiveId}/invites`, { maxUses: 0, expiresHours: 0 });
        modal({ title: "邀请伙伴", sub: "点击代码复制", build: (body) => {
            const codeBox = el("div", "inv-code", inv.code);
            codeBox.onclick = async () => {
                try { await navigator.clipboard.writeText(inv.code); toast("已复制邀请码", "honey"); }
                catch { toast("复制失败，请手动选择"); }
            };
            body.appendChild(codeBox);
            body.appendChild(el("div", "m-hint", "把这串代码发给伙伴 — 在「加入」里输入即可进巢"));
        } });
    } catch (err) { toast(err.message, "err"); }
}

/* ============ 个人资料 + 聊天热力图 ============ */
function colorRowP(initial) {
    const node = el("div", "color-row");
    const state2 = { node, value: initial || PALETTE[0] };
    for (const c of PALETTE) {
        const cell = el("button", "color-cell" + (c === state2.value ? " picked" : ""));
        cell.type = "button";
        cell.style.background = c;
        cell.onclick = () => {
            state2.value = c;
            node.querySelectorAll(".color-cell").forEach((x) => x.classList.remove("picked"));
            cell.classList.add("picked");
        };
        node.appendChild(cell);
    }
    return state2;
}

function openProfile() {
    modal({ title: "个人资料", sub: `@${state.me.username}`, build: (body) => {
        const fNick = el("label", "m-field");
        fNick.appendChild(el("span", "", "昵称"));
        const iNick = el("input", "m-input");
        iNick.maxLength = 16; iNick.value = state.me.nickname ?? "";
        fNick.appendChild(iNick); body.appendChild(fNick);

        const fBio = el("label", "m-field");
        fBio.appendChild(el("span", "", "个性签名"));
        const iBio = el("input", "m-input");
        iBio.maxLength = 100; iBio.value = state.me.bio ?? ""; iBio.placeholder = "写点什么";
        fBio.appendChild(iBio); body.appendChild(fBio);

        body.appendChild(el("div", "m-sec", "头像颜色"));
        const colors = colorRowP(state.me.avatarColor);
        body.appendChild(colors.node);

        body.appendChild(el("div", "m-sec", "聊天热力图 · 近 26 周"));
        const grid = el("div", "heat-grid");
        body.appendChild(grid);
        api.get("/users/me/heatmap").then((rows) => {
            const byDate = new Map(rows.map((r) => [r.date, r.count]));
            for (let i = 26 * 7 - 1; i >= 0; i--) {
                const d = new Date(Date.now() - i * 86400000);
                const key = `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
                const c = byDate.get(key) ?? 0;
                const level = c === 0 ? 0 : c <= 2 ? 1 : c <= 5 ? 2 : c <= 9 ? 3 : 4;
                const cell = el("i", `heat-cell l${level}`);
                cell.title = `${key} · ${c} 条`;
                grid.appendChild(cell);
            }
        }).catch(() => {});

        const save = el("button", "m-btn", "保 存");
        save.onclick = async () => {
            save.disabled = true;
            try {
                state.me = await api.put("/users/me", {
                    nickname: iNick.value.trim(), bio: iBio.value.trim(), avatarColor: colors.value,
                });
                renderUserChip();
                closeModal();
                toast("已保存", "honey");
            } catch (err) { toast(err.message, "err"); save.disabled = false; }
        };
        body.appendChild(save);
    } });
}

/* ============ 活跃统计 ============ */
async function openStats() {
    if (state.mode !== "hive" || !state.currentHiveId) return toast("先进入一个蜂巢");
    let stats;
    try { stats = await api.get(`/hives/${state.currentHiveId}/stats`); }
    catch (err) { return toast(err.message, "err"); }
    modal({ title: "活跃统计", sub: state.detail?.name ?? "", build: (body) => {
        body.appendChild(el("div", "m-sec", "近 7 日消息量"));
        const byDate = new Map((stats.daily ?? []).map((r) => [r.date, r.count]));
        const days = [];
        for (let i = 6; i >= 0; i--) {
            const d = new Date(Date.now() - i * 86400000);
            const key = `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
            days.push({ label: `${d.getMonth() + 1}/${d.getDate()}`, count: byDate.get(key) ?? 0 });
        }
        const max = Math.max(1, ...days.map((x) => x.count));
        const bars = el("div", "bars");
        for (const d of days) {
            const col = el("div", "bar-col" + (d.count === max && d.count > 0 ? " max" : ""));
            col.appendChild(el("span", "bar-val", String(d.count)));
            const fill = el("div", "bar-fill");
            fill.style.height = `${Math.round((d.count / max) * 82)}%`;
            col.appendChild(fill);
            col.appendChild(el("span", "bar-day", d.label));
            bars.appendChild(col);
        }
        body.appendChild(bars);
        if (stats.topSpeakers?.length) {
            body.appendChild(el("div", "m-sec", "发言排行"));
            stats.topSpeakers.forEach((s, i) => {
                const r = el("div", "rank-row");
                r.appendChild(el("i", "", pad2(i + 1)));
                r.appendChild(el("b", "", s.name));
                r.appendChild(el("span", "", `${s.count} 条`));
                body.appendChild(r);
            });
        }
    } });
}

/* ============ 蜂巢管理（资料 / 频道 / 危险区） ============ */
function openHiveManage() {
    if (state.mode !== "hive" || !state.detail) return;
    const d = state.detail;
    modal({ title: "蜂巢管理", sub: d.name, build: (body) => {
        if (can(P.MANAGE_HIVE)) {
            body.appendChild(el("div", "m-sec", "资料"));
            const iName = el("input", "m-input");
            iName.maxLength = 30; iName.value = d.name;
            const fName = el("label", "m-field");
            fName.appendChild(el("span", "", "名称")); fName.appendChild(iName);
            body.appendChild(fName);
            const iDesc = el("input", "m-input");
            iDesc.maxLength = 100; iDesc.value = d.description ?? "";
            const fDesc = el("label", "m-field");
            fDesc.appendChild(el("span", "", "简介")); fDesc.appendChild(iDesc);
            body.appendChild(fDesc);
            const save = el("button", "m-btn", "保存资料");
            save.onclick = async () => {
                try {
                    await api.put(`/hives/${d.id}`, {
                        name: iName.value.trim(), description: iDesc.value.trim(), iconColor: d.iconColor,
                    });
                    toast("已保存", "honey");
                    state.hives = await api.get("/hives");
                    state.hives.forEach((h, i) => registerHiveLabel(h, i));
                    renderHiveIndex();
                    await refreshCurrentHive();
                    renderStageCopy(no(state.hives.findIndex((h) => h.id === d.id)), iName.value.trim(),
                        $("sc-meta").textContent);
                } catch (err) { toast(err.message, "err"); }
            };
            body.appendChild(save);
        }
        if (can(P.MANAGE_CHANNELS)) {
            body.appendChild(el("div", "m-sec", "频道"));
            const texts = (d.channels ?? []).filter((c) => c.type === "TEXT");
            texts.forEach((c, i) => {
                const r = el("div", "ch-row");
                r.appendChild(el("i", "", no(i)));
                r.appendChild(el("span", "ch-name", c.name));
                const ed = el("button", "mini-act", "编辑");
                ed.onclick = () => openChannelEdit(c);
                r.appendChild(ed);
                const del = el("button", "mini-act red", "删除");
                del.onclick = () => confirmM("删除频道", `确定删除「${c.name}」？消息将一并清除`, async () => {
                    try { await api.del(`/channels/${c.id}`); await refreshCurrentHive(); openHiveManage(); }
                    catch (err) { toast(err.message, "err"); }
                });
                r.appendChild(del);
                body.appendChild(r);
            });
            const iNew = el("input", "m-input");
            iNew.maxLength = 30; iNew.placeholder = "新频道名称";
            const fNew = el("label", "m-field");
            fNew.style.marginTop = "10px";
            fNew.appendChild(iNew); body.appendChild(fNew);
            const add = el("button", "m-btn ghost", "＋ 新建文字频道");
            add.onclick = async () => {
                const name = iNew.value.trim();
                if (!name) return;
                try {
                    await api.post(`/hives/${d.id}/channels`, { name, type: "TEXT", parentId: null, topic: "" });
                    await refreshCurrentHive();
                    openHiveManage();
                } catch (err) { toast(err.message, "err"); }
            };
            body.appendChild(add);
        }
        body.appendChild(el("div", "m-sec", "危险区"));
        if (isOwner()) {
            const boom = el("button", "m-btn danger", "解散蜂巢");
            boom.onclick = () => confirmM("解散蜂巢", `确定解散「${d.name}」？所有频道与消息将永久删除！`, async () => {
                try { await api.del(`/hives/${d.id}`); location.reload(); }
                catch (err) { toast(err.message, "err"); }
            });
            body.appendChild(boom);
        } else {
            const leave = el("button", "m-btn danger", "退出蜂巢");
            leave.onclick = () => confirmM("退出蜂巢", `确定退出「${d.name}」吗？`, async () => {
                try { await api.post(`/hives/${d.id}/leave`); location.reload(); }
                catch (err) { toast(err.message, "err"); }
            });
            body.appendChild(leave);
        }
    } });
}

function openChannelEdit(c) {
    modal({ title: "编辑频道", sub: `#${c.name}`, build: (body) => {
        const iName = el("input", "m-input");
        iName.maxLength = 30; iName.value = c.name;
        const fName = el("label", "m-field");
        fName.appendChild(el("span", "", "名称")); fName.appendChild(iName);
        body.appendChild(fName);
        const iTopic = el("input", "m-input");
        iTopic.maxLength = 100; iTopic.value = c.topic ?? "";
        const fTopic = el("label", "m-field");
        fTopic.appendChild(el("span", "", "主题")); fTopic.appendChild(iTopic);
        body.appendChild(fTopic);
        const save = el("button", "m-btn", "保 存");
        save.onclick = async () => {
            try {
                await api.put(`/channels/${c.id}`, { name: iName.value.trim(), topic: iTopic.value.trim() });
                await refreshCurrentHive();
                if (state.currentChannelId === c.id) {
                    $("p-name").textContent = iName.value.trim();
                    $("p-topic").textContent = iTopic.value.trim();
                }
                openHiveManage();
            } catch (err) { toast(err.message, "err"); }
        };
        body.appendChild(save);
    } });
}

/** 极简确认弹层 */
function confirmM(title, text, onOk) {
    modal({ title, sub: "CONFIRM", build: (body) => {
        body.appendChild(el("div", "m-hint", text));
        const row = el("div", "m-row2");
        const no_ = el("button", "m-btn ghost", "取 消");
        no_.onclick = closeModal;
        const ok = el("button", "m-btn danger", "确 定");
        ok.onclick = () => { closeModal(); onOk(); };
        row.appendChild(no_); row.appendChild(ok);
        body.appendChild(row);
    } });
}

/* ============ 角色与权限管理 ============ */
function openRoles() {
    if (state.mode !== "hive" || !state.detail) return;
    modal({ title: "角色管理", sub: "成员权限 = 所有角色的并集", build: (body) => {
        for (const r of (state.detail.roles ?? [])) {
            const row = el("div", "role-row");
            const dot = el("span", "role-dot");
            dot.style.background = r.color ?? "#9a9a90";
            row.appendChild(dot);
            row.appendChild(el("span", "role-name", r.name));
            if (r.isDefault) row.appendChild(el("span", "role-tag", "默认 · 所有成员"));
            const ed = el("button", "mini-act", "编辑");
            ed.onclick = () => openRoleEdit(r);
            row.appendChild(ed);
            if (!r.isDefault) {
                const del = el("button", "mini-act red", "删除");
                del.onclick = () => confirmM("删除角色", `确定删除「${r.name}」？拥有者将失去对应权限`, async () => {
                    try { await api.del(`/roles/${r.id}`); await refreshCurrentHive(); openRoles(); }
                    catch (err) { toast(err.message, "err"); }
                });
                row.appendChild(del);
            }
            body.appendChild(row);
        }
        const add = el("button", "m-btn ghost", "＋ 新建角色");
        add.onclick = () => openRoleEdit(null);
        body.appendChild(add);
    } });
}

function openRoleEdit(role) {
    modal({ title: role ? `编辑角色` : "新建角色", sub: role?.name ?? "NEW ROLE", build: (body) => {
        const iName = el("input", "m-input");
        iName.maxLength = 20; iName.value = role?.name ?? ""; iName.placeholder = "例如：元老 / 巡逻蜂";
        const fName = el("label", "m-field");
        fName.appendChild(el("span", "", "角色名称")); fName.appendChild(iName);
        body.appendChild(fName);

        body.appendChild(el("div", "m-sec", "名字颜色"));
        const colors = colorRowP(role?.color ?? "#5C6BC0");
        body.appendChild(colors.node);

        body.appendChild(el("div", "m-sec", "权限"));
        const grid = el("div", "perm-grid");
        const checks = new Map();
        for (const p of PERM_DEFS) {
            const item = el("label", "perm-item");
            const cb = document.createElement("input");
            cb.type = "checkbox";
            cb.checked = role
                ? (role.permissions & p.bit) === p.bit
                : [P.SEND, P.ATTACH, P.REACT].includes(p.bit);
            checks.set(p.bit, cb);
            item.appendChild(cb);
            item.appendChild(el("span", "", p.label));
            grid.appendChild(item);
        }
        body.appendChild(grid);

        const save = el("button", "m-btn", "保 存");
        save.onclick = async () => {
            const name = iName.value.trim();
            if (!name) return toast("名称不能为空", "err");
            let permissions = 0;
            checks.forEach((cb, bit) => { if (cb.checked) permissions |= bit; });
            try {
                if (role) await api.put(`/roles/${role.id}`, { name, color: colors.value, permissions });
                else await api.post(`/hives/${state.currentHiveId}/roles`, { name, color: colors.value, permissions });
                await refreshCurrentHive();
                openRoles();
                toast("角色已保存", "honey");
            } catch (err) { toast(err.message, "err"); }
        };
        body.appendChild(save);
    } });
}

function openAssignRoles(m) {
    const assignable = (state.detail?.roles ?? []).filter((r) => !r.isDefault);
    modal({ title: "分配角色", sub: m.nickname, build: (body) => {
        if (!assignable.length) {
            body.appendChild(el("div", "m-hint", "还没有可分配的角色 — 先在「角色」里创建"));
            return;
        }
        const checks = new Map();
        for (const r of assignable) {
            const item = el("label", "perm-item");
            const cb = document.createElement("input");
            cb.type = "checkbox";
            cb.checked = m.roleIds?.includes(r.id) ?? false;
            checks.set(r.id, cb);
            const dot = el("span", "role-dot");
            dot.style.background = r.color ?? "#9a9a90";
            item.appendChild(cb); item.appendChild(dot);
            item.appendChild(el("span", "", r.name));
            body.appendChild(item);
        }
        const save = el("button", "m-btn", "保 存");
        save.onclick = async () => {
            const roleIds = [...checks.entries()].filter(([, cb]) => cb.checked).map(([id]) => id);
            try {
                await api.put(`/hives/${state.currentHiveId}/members/${m.userId}/roles`, { roleIds });
                await refreshCurrentHive();
                closeModal();
                toast("已保存", "honey");
            } catch (err) { toast(err.message, "err"); }
        };
        body.appendChild(save);
    } });
}

/* ============ 成员小卡（私信 / 加好友 / 角色 / 禁言 / 踢出） ============ */
function showMemberPop(anchor, m) {
    hideMemberPop();
    const pop = el("div", "member-pop");
    pop.id = "member-pop";
    const head = el("div", "mp-head");
    head.appendChild(avaEl(m, state.online.has(m.userId) ? "online" : ""));
    const nm = el("span", "mp-name", m.nickname ?? m.username);
    const rc = roleColorOf(m.userId);
    if (rc) nm.style.color = rc;
    head.appendChild(nm);
    if (m.owner) head.appendChild(el("span", "", "👑"));
    pop.appendChild(head);
    pop.appendChild(el("div", "mp-user",
        `@${m.username}` + (m.mutedUntil && new Date(m.mutedUntil) > new Date() ? " · 禁言中" : "")));

    const act = (label, fn, red = false) => {
        const b = el("button", "mp-act" + (red ? " red" : ""), label);
        b.onclick = () => { hideMemberPop(); fn(); };
        pop.appendChild(b);
    };
    if (m.userId !== state.me.id) {
        act("发私信 ↗", () => openDmWith(m.userId));
        if (!state.friends.some((f) => f.userId === m.userId)) {
            act("加为好友", async () => {
                try { await api.post("/friends/requests", { username: m.username }); toast("好友申请已发送", "honey"); }
                catch (err) { toast(err.message, "err"); }
            });
        }
    }
    if (can(P.MANAGE_ROLES)) act("分配角色", () => openAssignRoles(m));
    if (m.userId !== state.me.id && !m.owner) {
        const muted = m.mutedUntil && new Date(m.mutedUntil) > new Date();
        if (can(P.MUTE)) {
            muted
                ? act("解除禁言", () => moderate(`/members/${m.userId}/mute`, "del"))
                : act("禁言 10 分钟", () => moderate(`/members/${m.userId}/mute`, "post", { minutes: 10 }), true);
        }
        if (can(P.KICK)) {
            act("踢出蜂巢", () => confirmM("踢出成员", `确定将 ${m.nickname} 请出蜂巢吗？`,
                () => moderate(`/members/${m.userId}`, "del")), true);
        }
    }
    document.body.appendChild(pop);
    const r = anchor.getBoundingClientRect();
    pop.style.left = `${Math.min(r.left, innerWidth - 250)}px`;
    pop.style.top = `${Math.min(r.bottom + 8, innerHeight - pop.offsetHeight - 12)}px`;
    setTimeout(() => addEventListener("click", hideMemberPop, { once: true }), 0);
}
function hideMemberPop() { document.getElementById("member-pop")?.remove(); }

async function moderate(path, method, body) {
    try {
        const full = `/hives/${state.currentHiveId}${path}`;
        method === "del" ? await api.del(full) : await api.post(full, body);
        toast("操作成功", "honey");
        await refreshCurrentHive();
    } catch (err) { toast(err.message, "err"); }
}

async function openDmWith(userId) {
    try {
        const { channelId } = await api.post(`/dms/${userId}`);
        await enterHome();
        const conv = state.dms.find((x) => x.channelId === channelId);
        if (conv) openDm(conv);
    } catch (err) { toast(err.message, "err"); }
}

/* ============ 设置（外观主题） ============ */
function openSettings() {
    modal({ title: "设置", sub: "SETTINGS", build: (body) => {
        body.appendChild(el("div", "m-sec", "外观主题"));
        const row = el("div", "seg-row");
        const opts = [
            ["light", "☀ 白天 · 瓷白"],
            ["dark", "☾ 黑夜 · 墨岩"],
            ["auto", "◐ 跟随系统"],
        ];
        for (const [val, label] of opts) {
            const b = el("button", "seg" + (themeSetting() === val ? " on" : ""), label);
            b.onclick = () => {
                setThemeSetting(val);
                row.querySelectorAll(".seg").forEach((x) => x.classList.remove("on"));
                b.classList.add("on");
            };
            row.appendChild(b);
        }
        body.appendChild(row);
        body.appendChild(el("div", "m-hint",
            "黑夜为月光照明 · 琥珀灯火更醒目 — 跟随系统会随操作系统自动切换"));
    } });
}

/* ============ 添加好友 ============ */
function openAddFriend() {
    modal({ title: "添加好友", sub: "输入对方用户名", build: (body) => {
        const inp = el("input", "m-input");
        inp.placeholder = "username";
        body.appendChild(inp);
        const btn = el("button", "m-btn", "发送申请");
        btn.onclick = async () => {
            const u = inp.value.trim();
            if (!u) return;
            btn.disabled = true;
            try {
                await api.post("/friends/requests", { username: u });
                closeModal();
                toast("好友申请已发送", "honey");
            } catch (err) { toast(err.message, "err"); btn.disabled = false; }
        };
        body.appendChild(btn);
        setTimeout(() => inp.focus(), 80);
    } });
}

boot();
