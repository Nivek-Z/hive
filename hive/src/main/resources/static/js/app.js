// 蜂巢 Hive · 前端应用核心：状态 + 交互 + 实时事件
import { api, getToken, setToken, setUnauthorizedHandler } from "./api.js";
import { createSocket } from "./ws.js";
import {
    $, el, esc, fmtTime, fmtDay, isSameDayIso, hexAvatar, renderContent,
    showModal, closeModal, confirmModal, showCtx, closeCtx, toast,
    showEmojiPop, hideEmojiPop, openLightbox, bindLightbox, colorRow,
} from "./ui.js";

// 权限位（与后端 Permissions.java 一致）
const P = {
    ADMIN: 1, MANAGE_HIVE: 2, MANAGE_CHANNELS: 4, MANAGE_ROLES: 8,
    KICK: 16, MUTE: 32, DEL_MSG: 64, INVITE: 128,
    MENTION_ALL: 256, SEND: 512, ATTACH: 1024, REACT: 2048,
};

const state = {
    me: null,
    mode: "hive",          // hive=蜂巢模式 / home=好友私信模式
    hives: [],
    detail: null,          // 当前蜂巢详情
    members: [],
    friends: [],
    requests: [],          // 收到的好友申请
    dms: [],               // 私信会话列表
    currentHiveId: null,
    currentChannelId: null,
    messages: [],
    online: new Set(),
    unreads: new Map(),    // channelId -> count
    collapsed: new Set(),  // 折叠的分区 id
    replyTo: null,
    typing: new Map(),     // userId -> {name, timer}
    oldestId: null,
    hasMore: true,
    loadingHistory: false,
    socket: null,
    lastTypingSent: 0,
};

const can = (bit) => {
    const p = state.detail?.myPermissions ?? 0;
    return (p & P.ADMIN) !== 0 || (p & bit) === bit;
};
const isOwner = () => state.detail && state.me && state.detail.ownerId === state.me.id;

/* ============================================================
   启动
   ============================================================ */

async function boot() {
    bindAuth();
    bindStatic();
    setUnauthorizedHandler(logout);
    if (getToken()) {
        try {
            state.me = await api.get("/users/me");
            return enterApp();
        } catch { /* token 失效，落到登录页 */ }
    }
    showAuth();
}

function showAuth() {
    $("auth-view").classList.remove("hidden");
    $("app-view").classList.add("hidden");
}

async function enterApp() {
    $("auth-view").classList.add("hidden");
    $("app-view").classList.remove("hidden");
    renderMeBar();
    state.socket = createSocket({ onEvent: handleWsEvent, onDown: () => {}, onOpen: onWsOpen });
    state.hives = await api.get("/hives");
    renderRail();
    refreshDms();
    refreshFriendsData();
    if (state.hives.length) {
        await selectHive(state.hives[0].id);
    } else {
        renderEmptyWorld();
    }
}

function logout() {
    state.socket?.close();
    setToken(null);
    location.reload();
}

/* ============================================================
   登录 / 注册
   ============================================================ */

function bindAuth() {
    let mode = "login";
    const applyMode = () => {
        const reg = mode === "register";
        $("auth-title").textContent = reg ? "加入蜂巢" : "欢迎回巢";
        $("auth-sub").textContent = reg ? "注册一个新账号，开始嗡嗡" : "登录你的蜂巢账号";
        $("auth-nickname-field").classList.toggle("hidden", !reg);
        $("auth-submit").textContent = reg ? "注 册" : "登 录";
        $("auth-switch-text").textContent = reg ? "已有账号？" : "还没有账号？";
        $("auth-switch-link").textContent = reg ? "去登录" : "立即注册";
        $("auth-error").textContent = "";
    };
    $("auth-switch-link").onclick = () => { mode = mode === "login" ? "register" : "login"; applyMode(); };

    $("auth-form").onsubmit = async (e) => {
        e.preventDefault();
        const username = $("auth-username").value.trim();
        const password = $("auth-password").value;
        $("auth-error").textContent = "";
        $("auth-submit").disabled = true;
        try {
            const body = mode === "register"
                ? { username, password, nickname: $("auth-nickname").value.trim() || username }
                : { username, password };
            const resp = await api.post(`/auth/${mode}`, body);
            setToken(resp.token);
            state.me = resp.user;
            await enterApp();
        } catch (err) {
            $("auth-error").textContent = err.message;
        } finally {
            $("auth-submit").disabled = false;
        }
    };
}

/* ============================================================
   蜂巢条 / 选择蜂巢
   ============================================================ */

function renderRail() {
    const list = $("rail-list");
    list.innerHTML = "";
    for (const h of state.hives) {
        const active = h.id === state.currentHiveId && state.mode === "hive";
        const item = el("div", "rail-item" + (active ? " active" : ""));
        const pill = el("span", "pill");
        const hex = el("button", "rail-hex");
        hex.style.background = `linear-gradient(150deg, ${h.iconColor}, ${h.iconColor}88)`;
        hex.textContent = [...h.name][0] ?? "蜂";
        hex.title = h.name;
        hex.onclick = () => selectHive(h.id);
        item.append(pill, hex);
        list.appendChild(item);
    }
}

async function selectHive(hiveId) {
    state.mode = "hive";
    state.currentHiveId = hiveId;
    $("btn-home").classList.remove("active");
    $("btn-hive-menu").style.visibility = "";
    $("members-title").textContent = "成员";
    setComposerVisible(true);
    renderRail();
    try {
        state.detail = await api.get(`/hives/${hiveId}`);
        state.members = await api.get(`/hives/${hiveId}/members`);
    } catch (err) {
        toast(err.message, "err");
        return;
    }
    state.unreads = new Map((state.detail.unreads ?? []).map((u) => [u.channelId, u.count]));
    $("hive-name").textContent = state.detail.name;
    renderTree();
    renderMembers();
    const firstText = state.detail.channels.find((c) => c.type === "TEXT");
    await selectChannel(firstText ? firstText.id : null);
}

async function refreshHiveDetail() {
    if (!state.currentHiveId) return;
    try {
        state.detail = await api.get(`/hives/${state.currentHiveId}`);
        state.members = await api.get(`/hives/${state.currentHiveId}/members`);
        state.unreads = new Map((state.detail.unreads ?? []).map((u) => [u.channelId, u.count]));
        $("hive-name").textContent = state.detail.name;
        renderTree();
        renderMembers();
        // 当前频道被删除时回落到第一个文字频道
        if (state.currentChannelId &&
            !state.detail.channels.some((c) => c.id === state.currentChannelId)) {
            const firstText = state.detail.channels.find((c) => c.type === "TEXT");
            await selectChannel(firstText ? firstText.id : null);
        }
    } catch { /* 可能已被移出该蜂巢，由 KICKED 事件处理 */ }
}

function renderEmptyWorld() {
    state.detail = null;
    state.currentChannelId = null;
    $("hive-name").textContent = "—";
    $("channel-tree").innerHTML = "";
    $("member-list").innerHTML = "";
    $("member-count").textContent = "";
    $("channel-name").textContent = "欢迎来到蜂巢";
    $("channel-topic").textContent = "";
    const listEl = $("msg-list");
    listEl.innerHTML = "";
    const empty = el("div", "empty-channel");
    empty.appendChild(el("div", "big-hex", "🐝"));
    empty.appendChild(el("div", "", "你还没有加入任何蜂巢"));
    const hint = el("div", "", "点左下角 ＋ 创建一个，或用邀请码加入伙伴的蜂巢");
    hint.style.cssText = "margin-top:8px;font-size:12.5px;";
    empty.appendChild(hint);
    listEl.appendChild(empty);
}

/* ============================================================
   频道树
   ============================================================ */

function renderTree() {
    const tree = $("channel-tree");
    tree.innerHTML = "";
    if (!state.detail) return;
    const byParent = new Map();
    for (const c of state.detail.channels) {
        const key = c.parentId ?? 0;
        if (!byParent.has(key)) byParent.set(key, []);
        byParent.get(key).push(c);
    }
    const renderLevel = (parentKey, container) => {
        for (const c of byParent.get(parentKey) ?? []) {
            container.appendChild(c.type === "CATEGORY" ? buildCategory(c, byParent, renderLevel) : buildChannel(c));
        }
    };
    renderLevel(0, tree);
}

function buildCategory(cat, byParent, renderLevel) {
    const wrap = el("div", "tree-cat" + (state.collapsed.has(cat.id) ? " collapsed" : ""));
    const head = el("div", "tree-cat-head");
    head.appendChild(el("span", "cat-arrow", "▼"));
    head.appendChild(el("span", "cat-name", cat.name));
    if (can(P.MANAGE_CHANNELS)) {
        const add = el("button", "cat-add", "＋");
        add.title = "在此分区内新建";
        add.onclick = (e) => { e.stopPropagation(); openCreateChannelModal(cat.id, cat.name); };
        head.appendChild(add);
    }
    head.onclick = () => {
        state.collapsed.has(cat.id) ? state.collapsed.delete(cat.id) : state.collapsed.add(cat.id);
        wrap.classList.toggle("collapsed");
    };
    head.oncontextmenu = (e) => { e.preventDefault(); channelCtxMenu(e, cat); };
    wrap.appendChild(head);
    const children = el("div", "tree-children");
    renderLevel(cat.id, children);
    wrap.appendChild(children);
    return wrap;
}

function buildChannel(c) {
    const row = el("div", "tree-channel" + (c.id === state.currentChannelId ? " active" : ""));
    row.appendChild(el("span", "ch-hex", "⬡"));
    row.appendChild(el("span", "ch-name", c.name));
    const unread = state.unreads.get(c.id);
    if (unread) row.appendChild(el("span", "unread-badge", unread > 99 ? "99+" : String(unread)));
    row.onclick = () => selectChannel(c.id);
    row.oncontextmenu = (e) => { e.preventDefault(); channelCtxMenu(e, c); };
    return row;
}

function channelCtxMenu(e, c) {
    if (!can(P.MANAGE_CHANNELS)) return;
    showCtx(e.clientX, e.clientY, [
        { icon: "✏️", label: `编辑「${c.name}」`, onClick: () => openEditChannelModal(c) },
        "-",
        {
            icon: "🗑", label: "删除", danger: true, onClick: () =>
                confirmModal("删除频道", c.type === "CATEGORY"
                    ? `确定删除分区「${c.name}」吗？其子频道会上移一层，不会被删除。`
                    : `确定删除频道「${c.name}」吗？频道内消息将一并删除。`,
                    async () => {
                        try { await api.del(`/channels/${c.id}`); } catch (err) { toast(err.message, "err"); }
                    }),
        },
    ]);
}

/* ============================================================
   频道与消息
   ============================================================ */

async function selectChannel(channelId) {
    state.currentChannelId = channelId;
    state.replyTo = null;
    state.messages = [];
    state.oldestId = null;
    state.hasMore = true;
    state.typing.clear();
    renderTyping();
    hideReplyBar();
    renderTree();

    const ch = state.detail?.channels.find((c) => c.id === channelId);
    $("channel-name").textContent = ch ? ch.name : "选择一个频道";
    $("channel-topic").textContent = ch?.topic ?? "";
    $("composer-input").placeholder = ch ? `给 ⬡${ch.name} 发消息…` : "选择一个频道开始聊天";

    if (!channelId) {
        $("msg-list").innerHTML = "";
        if (state.detail) {
            const e1 = el("div", "empty-channel");
            e1.appendChild(el("div", "big-hex", "⬡"));
            e1.appendChild(el("div", "", "这个蜂巢还没有文字频道"));
            $("msg-list").appendChild(e1);
        }
        return;
    }
    await loadHistory(true);
    markRead();
}

async function loadHistory(initial) {
    if (!state.currentChannelId || state.loadingHistory) return;
    if (!initial && !state.hasMore) return;
    state.loadingHistory = true;
    const cid = state.currentChannelId;
    try {
        const qs = initial ? "?limit=50" : `?limit=50&before=${state.oldestId}`;
        const page = await api.get(`/channels/${cid}/messages${qs}`);
        if (cid !== state.currentChannelId) return; // 期间切换了频道
        state.hasMore = page.length === 50;
        if (page.length) state.oldestId = page[0].id;
        if (initial) {
            state.messages = page;
            rerenderMessages(true);
        } else {
            state.messages = [...page, ...state.messages];
            const sc = $("msg-scroll");
            const oldHeight = sc.scrollHeight;
            const oldTop = sc.scrollTop;
            rerenderMessages(false);
            sc.scrollTop = sc.scrollHeight - oldHeight + oldTop;
        }
    } catch (err) {
        toast(err.message, "err");
    } finally {
        state.loadingHistory = false;
    }
}

function rerenderMessages(scrollBottom) {
    const listEl = $("msg-list");
    listEl.innerHTML = "";
    if (!state.messages.length) {
        const empty = el("div", "empty-channel");
        empty.appendChild(el("div", "big-hex", "🍯"));
        empty.appendChild(el("div", "", "这里静悄悄的，发出第一声嗡嗡吧"));
        listEl.appendChild(empty);
        return;
    }
    let prev = null;
    for (const m of state.messages) {
        if (!prev || !isSameDayIso(prev.createdAt, m.createdAt)) {
            listEl.appendChild(el("div", "date-divider", fmtDay(m.createdAt)));
        }
        listEl.appendChild(buildMsgRow(m, prev));
        prev = m;
    }
    if (scrollBottom) {
        const sc = $("msg-scroll");
        sc.scrollTop = sc.scrollHeight;
    }
}

function appendMessage(m) {
    const listEl = $("msg-list");
    listEl.querySelector(".empty-channel")?.remove();
    const prev = state.messages.at(-2) ?? null;
    if (!prev || !isSameDayIso(prev.createdAt, m.createdAt)) {
        listEl.appendChild(el("div", "date-divider", fmtDay(m.createdAt)));
    }
    listEl.appendChild(buildMsgRow(m, prev));
    const sc = $("msg-scroll");
    if (sc.scrollHeight - sc.scrollTop - sc.clientHeight < 160 || m.senderId === state.me.id) {
        sc.scrollTop = sc.scrollHeight;
    }
}

function buildMsgRow(m, prev) {
    if (m.type === "SYSTEM") {
        const sys = el("div", "msg-system");
        sys.dataset.id = m.id ?? "";
        const pill = el("span", "sys-pill", m.content);
        sys.appendChild(pill);
        return sys;
    }
    const grouped = prev && prev.type !== "SYSTEM" && prev.senderId === m.senderId &&
        !m.replyToId && isSameDayIso(prev.createdAt, m.createdAt) &&
        (new Date(m.createdAt) - new Date(prev.createdAt)) < 5 * 60 * 1000;

    const row = el("div", "msg-row" + (grouped ? " grouped" : "") + (m.pending ? " pending" : ""));
    row.dataset.id = m.id ?? "";
    if (m.nonce) row.dataset.nonce = m.nonce;

    const avatar = hexAvatar({ nickname: m.senderNickname, avatarColor: m.senderAvatarColor, avatarUrl: m.senderAvatarUrl }, "");
    avatar.classList.add("msg-avatar");
    row.appendChild(avatar);

    const body = el("div", "msg-body");
    if (m.replyToId) {
        const quote = el("div", "msg-reply-quote");
        quote.innerHTML = `↩ <b>${esc(m.replySenderNickname ?? "未知")}</b>：${esc(m.replyContent ?? "")}`;
        quote.onclick = () => jumpToMessage(m.replyToId);
        body.appendChild(quote);
    }
    const meta = el("div", "msg-meta");
    meta.appendChild(el("span", "msg-author", m.senderNickname ?? "未知用户"));
    meta.appendChild(el("span", "msg-time", fmtTime(m.createdAt)));
    body.appendChild(meta);

    if (m.type === "IMAGE") {
        const img = el("img", "msg-image");
        img.src = m.content;
        img.alt = "图片消息";
        img.onclick = () => openLightbox(m.content);
        body.appendChild(img);
    } else {
        const content = el("div", "msg-content");
        content.innerHTML = renderContent(m.content);
        body.appendChild(content);
    }

    if (m.reactions?.length) {
        const box = el("div", "reactions");
        for (const r of m.reactions) {
            const mine = r.userIds?.includes(state.me.id);
            const chip = el("button", "reaction-chip" + (mine ? " mine" : ""));
            chip.innerHTML = `${esc(r.emoji)}<span class="rc-count">${r.count}</span>`;
            chip.onclick = () => toggleReaction(m, r.emoji, mine);
            box.appendChild(chip);
        }
        body.appendChild(box);
    }
    row.appendChild(body);

    // 悬浮工具条（待发送的临时消息没有）
    if (m.id) {
        const bar = el("div", "msg-toolbar");
        if (can(P.REACT)) {
            const react = el("button", "tool-btn", "😀");
            react.title = "添加回应";
            react.onclick = (e) => showEmojiPop(e.currentTarget, (emoji) => toggleReaction(m, emoji, false));
            bar.appendChild(react);
        }
        const reply = el("button", "tool-btn", "↩");
        reply.title = "回复";
        reply.onclick = () => setReplyTo(m);
        bar.appendChild(reply);
        if (m.senderId === state.me.id || can(P.DEL_MSG)) {
            const del = el("button", "tool-btn", "🗑");
            del.title = "撤回 / 删除";
            del.onclick = () => confirmModal("撤回消息", "确定撤回这条消息吗？",
                async () => { try { await api.del(`/messages/${m.id}`); } catch (err) { toast(err.message, "err"); } });
            bar.appendChild(del);
        }
        row.appendChild(bar);
    }
    return row;
}

function jumpToMessage(id) {
    const node = $("msg-list").querySelector(`[data-id="${id}"]`);
    if (!node) { toast("原消息不在当前已加载的历史里"); return; }
    node.scrollIntoView({ behavior: "smooth", block: "center" });
    node.style.background = "rgba(255,179,0,.14)";
    setTimeout(() => { node.style.background = ""; }, 1200);
}

async function toggleReaction(m, emoji, mine) {
    if (!m.id) return;
    try {
        if (mine) await api.del(`/messages/${m.id}/reactions/${encodeURIComponent(emoji)}`);
        else await api.post(`/messages/${m.id}/reactions`, { emoji });
        // 最终状态由 REACTION_UPDATE 广播统一刷新
    } catch (err) {
        toast(err.message, "err");
    }
}

function markRead() {
    const cid = state.currentChannelId;
    if (!cid) return;
    const last = [...state.messages].reverse().find((m) => m.id);
    state.unreads.delete(cid);
    renderTree();
    if (last) api.post(`/channels/${cid}/read`, { lastMessageId: last.id }).catch(() => {});
}

/* ============================================================
   发送消息
   ============================================================ */

function sendMessage() {
    const input = $("composer-input");
    const text = input.value.trim();
    if (!text || !state.currentChannelId) return;
    if (!can(P.SEND)) { toast("你没有发言权限", "err"); return; }
    const nonce = `n${Date.now()}${Math.random().toString(36).slice(2, 6)}`;
    const temp = {
        id: null, nonce,
        channelId: state.currentChannelId,
        senderId: state.me.id,
        senderNickname: state.me.nickname,
        senderAvatarColor: state.me.avatarColor,
        senderAvatarUrl: state.me.avatarUrl,
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
        channelId: temp.channelId, content: text,
        replyToId: temp.replyToId, nonce,
    });
    input.value = "";
    autoSize(input);
    hideReplyBar();
    state.replyTo = null;
}

function setReplyTo(m) {
    state.replyTo = m;
    $("reply-name").textContent = m.senderNickname ?? "";
    $("reply-snippet").textContent = m.type === "IMAGE" ? "[图片]" : (m.content ?? "").slice(0, 40);
    $("reply-bar").classList.remove("hidden");
    $("composer-input").focus();
}
function hideReplyBar() {
    $("reply-bar").classList.add("hidden");
}

async function sendImage(file) {
    if (!file || !state.currentChannelId) return;
    if (!can(P.ATTACH)) { toast("你没有发送图片的权限", "err"); return; }
    try {
        const { url } = await api.upload(file);
        state.socket.send("MSG_SEND", {
            channelId: state.currentChannelId, content: url,
            type: "IMAGE", nonce: `img${Date.now()}`,
        });
    } catch (err) {
        toast(err.message, "err");
    }
}

/* ============================================================
   WebSocket 事件
   ============================================================ */

function onWsOpen() {
    // 断线重连后刷新当前视图（首连时 currentChannelId 还未设置，不会触发）
    if (state.currentHiveId && state.currentChannelId) {
        refreshHiveDetail();
        loadHistory(true);
    }
}

function handleWsEvent(type, d) {
    switch (type) {
        case "READY":
            state.online = new Set(d.onlineUserIds ?? []);
            renderMembers();
            break;

        case "MSG_NEW": {
            const m = d.message;
            clearTypingOf(m.senderId);
            // 私信消息：刷新会话列表（预览/排序/未读）与主页红点
            if (!state.detail?.channels.some((c) => c.id === m.channelId)) {
                setTimeout(refreshDms, 80);
            }
            if (d.nonce) {
                const idx = state.messages.findIndex((x) => x.nonce === d.nonce);
                if (idx >= 0) {
                    state.messages[idx] = m;
                    rerenderMessages(true);
                    return;
                }
            }
            if (m.channelId === state.currentChannelId) {
                state.messages.push(m);
                appendMessage(m);
                if (document.hasFocus()) markRead();
                else bumpUnread(m.channelId);
            } else {
                bumpUnread(m.channelId);
            }
            break;
        }

        case "MSG_DELETED":
            if (d.channelId === state.currentChannelId) {
                state.messages = state.messages.filter((x) => x.id !== d.messageId);
                rerenderMessages(false);
            }
            break;

        case "REACTION_UPDATE":
            if (d.channelId === state.currentChannelId) {
                const m = state.messages.find((x) => x.id === d.messageId);
                if (m) { m.reactions = d.reactions ?? []; rerenderMessages(false); }
            }
            break;

        case "TYPING": {
            if (d.channelId !== state.currentChannelId || d.userId === state.me.id) break;
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
            renderMembers();
            if (state.mode === "home") {
                renderHomeMembers();
                renderFriendsView();
            }
            break;

        case "FRIEND_EVENT":
            if (d.kind === "REQUEST_NEW") toast(`🐝 ${d.from.nickname} 请求加你为好友`, "ok");
            if (d.kind === "ACCEPTED") toast(`🎉 你和 ${d.friend.nickname} 已成为好友`, "ok");
            refreshAllHome();
            break;

        case "HIVE_EVENT":
            handleHiveEvent(d);
            break;

        case "ERROR":
            toast(d.message ?? "操作失败", "err");
            // 把仍处于待发送状态的消息标记失败
            state.messages.filter((x) => x.pending).forEach((x) => { x.pending = false; x.failed = true; });
            break;

        default:
            break;
    }
}

function handleHiveEvent(d) {
    if (d.kind === "KICKED") {
        state.hives = state.hives.filter((h) => h.id !== d.hiveId);
        renderRail();
        toast("你已被移出一个蜂巢", "err");
        if (state.currentHiveId === d.hiveId) {
            state.hives.length ? selectHive(state.hives[0].id) : renderEmptyWorld();
        }
        return;
    }
    if (d.hiveId === state.currentHiveId) {
        refreshHiveDetail();
    }
}

function bumpUnread(channelId) {
    // 私信未读由 refreshDms 统一计算
    if (!state.detail?.channels.some((c) => c.id === channelId)) return;
    state.unreads.set(channelId, (state.unreads.get(channelId) ?? 0) + 1);
    if (state.mode === "hive") renderTree();
}

function clearTypingOf(userId) {
    const t = state.typing.get(userId);
    if (t) { clearTimeout(t.timer); state.typing.delete(userId); renderTyping(); }
}

function renderTyping() {
    const bar = $("typing-bar");
    if (!state.typing.size) { bar.innerHTML = ""; return; }
    const names = [...state.typing.values()].map((t) => t.name).slice(0, 3).join("、");
    bar.innerHTML = `<span class="typing-dots"><i></i><i></i><i></i></span>${esc(names)} 正在输入…`;
}

/* ============================================================
   成员列表
   ============================================================ */

function renderMembers() {
    const listEl = $("member-list");
    listEl.innerHTML = "";
    if (!state.detail) { $("member-count").textContent = ""; return; }
    $("member-count").textContent = `· ${state.members.length}`;

    const onlineMembers = state.members.filter((m) => state.online.has(m.userId));
    const offlineMembers = state.members.filter((m) => !state.online.has(m.userId));

    const section = (label, arr, offline) => {
        if (!arr.length) return;
        listEl.appendChild(el("div", "section-label", `${label} — ${arr.length}`));
        for (const m of arr) listEl.appendChild(buildMemberRow(m, offline));
    };
    section("在线", onlineMembers, false);
    section("离线", offlineMembers, true);
}

function buildMemberRow(m, offline) {
    const row = el("div", "member-row" + (offline ? " offline" : ""));
    const av = hexAvatar(m, "small");
    row.appendChild(av);
    row.appendChild(el("span", "presence-dot"));

    const main = el("div", "member-main");
    const nameEl = el("div", "member-name");
    nameEl.appendChild(el("span", "", m.hiveNickname || m.nickname));
    if (m.owner) nameEl.appendChild(el("span", "crown", "👑"));
    if (m.mutedUntil && new Date(m.mutedUntil) > new Date()) nameEl.appendChild(el("span", "muted-mark", "🔇"));
    main.appendChild(nameEl);
    main.appendChild(el("div", "member-sub", `@${m.username}`));
    row.appendChild(main);

    row.oncontextmenu = (e) => { e.preventDefault(); memberCtxMenu(e, m); };
    row.onclick = (e) => memberCtxMenu(e, m);
    return row;
}

function memberCtxMenu(e, m) {
    const items = [];
    const muted = m.mutedUntil && new Date(m.mutedUntil) > new Date();
    items.push({ icon: "🐝", label: `@${m.username} · ${m.nickname}`, onClick: () => {} });
    if (m.userId !== state.me.id) {
        items.push("-");
        items.push({ icon: "💬", label: "发私信", onClick: () => openDmWith(m.userId) });
        if (!state.friends.some((f) => f.userId === m.userId)) {
            items.push({
                icon: "🤝", label: "加为好友", onClick: async () => {
                    try {
                        await api.post("/friends/requests", { username: m.username });
                        toast("好友申请已发送 🐝", "ok");
                    } catch (err) { toast(err.message, "err"); }
                },
            });
        }
    }
    if (m.userId !== state.me.id && !m.owner) {
        if (can(P.MUTE)) {
            items.push("-");
            if (muted) {
                items.push({ icon: "🔊", label: "解除禁言", onClick: () => moderate(`/members/${m.userId}/mute`, "del") });
            } else {
                items.push({ icon: "🔇", label: "禁言 10 分钟", onClick: () => moderate(`/members/${m.userId}/mute`, "post", { minutes: 10 }) });
                items.push({ icon: "🔕", label: "禁言 1 小时", onClick: () => moderate(`/members/${m.userId}/mute`, "post", { minutes: 60 }) });
            }
        }
        if (can(P.KICK)) {
            items.push("-");
            items.push({
                icon: "🚪", label: "踢出蜂巢", danger: true,
                onClick: () => confirmModal("踢出成员", `确定将 ${m.nickname} 请出蜂巢吗？`,
                    () => moderate(`/members/${m.userId}`, "del")),
            });
        }
    }
    showCtx(e.clientX, e.clientY, items);
}

async function moderate(path, method, body) {
    try {
        const full = `/hives/${state.currentHiveId}${path}`;
        method === "del" ? await api.del(full) : await api.post(full, body);
        toast("操作成功", "ok");
        refreshHiveDetail();
    } catch (err) {
        toast(err.message, "err");
    }
}

/* ============================================================
   弹窗：建巢 / 加入 / 邀请 / 建频道 / 蜂巢设置 / 个人资料
   ============================================================ */

function openCreateHiveModal() {
    const body = el("div");
    body.innerHTML = `
        <label class="field"><span>蜂巢名称</span><input id="m-hive-name" maxlength="30" placeholder="给你的蜂巢起个名字"></label>
        <label class="field"><span>简介（可选）</span><input id="m-hive-desc" maxlength="100" placeholder="一句话介绍"></label>
        <label class="field"><span>主题色</span></label>`;
    const colors = colorRow("#FFB300");
    body.appendChild(colors.node);
    showModal({
        title: "创建蜂巢", sub: "建好后自动生成默认频道和永久邀请码", body,
        actions: [
            { label: "取消", kind: "ghost" },
            {
                label: "创建", onClick: async (close) => {
                    const name = $("m-hive-name").value.trim();
                    if (!name) { toast("名称不能为空", "err"); return; }
                    try {
                        const detail = await api.post("/hives", {
                            name, description: $("m-hive-desc").value.trim(), iconColor: colors.value,
                        });
                        close();
                        state.hives = await api.get("/hives");
                        renderRail();
                        await selectHive(detail.id);
                        toast("蜂巢创建成功 🎉", "ok");
                    } catch (err) { toast(err.message, "err"); }
                },
            },
        ],
    });
}

function openJoinHiveModal() {
    const body = el("div");
    body.innerHTML = `<label class="field"><span>邀请码</span>
        <input id="m-join-code" maxlength="8" placeholder="8 位邀请码" style="text-transform:uppercase;letter-spacing:.3em;font-family:var(--mono);"></label>`;
    showModal({
        title: "加入蜂巢", sub: "向伙伴要一个邀请码", body,
        actions: [
            { label: "取消", kind: "ghost" },
            {
                label: "加入", onClick: async (close) => {
                    const code = $("m-join-code").value.trim().toUpperCase();
                    if (!code) return;
                    try {
                        const hive = await api.post(`/invites/${code}/join`);
                        close();
                        state.hives = await api.get("/hives");
                        renderRail();
                        await selectHive(hive.id);
                        toast(`欢迎加入「${hive.name}」🐝`, "ok");
                    } catch (err) { toast(err.message, "err"); }
                },
            },
        ],
    });
}

async function openInviteModal() {
    try {
        const invites = await api.get(`/hives/${state.currentHiveId}/invites`);
        const invite = invites[0] ?? await api.post(`/hives/${state.currentHiveId}/invites`, { maxUses: 0, expiresHours: 0 });
        const body = el("div");
        const codeBox = el("div", "invite-code-box", invite.code);
        codeBox.title = "点击复制";
        codeBox.onclick = async () => {
            try { await navigator.clipboard.writeText(invite.code); toast("已复制邀请码", "ok"); }
            catch { toast("复制失败，请手动选择"); }
        };
        body.appendChild(codeBox);
        body.appendChild(el("div", "invite-hint", "把这串代码发给伙伴，在「加入蜂巢」里输入即可"));
        showModal({ title: "邀请成员", body, actions: [{ label: "好的", kind: "ghost" }] });
    } catch (err) {
        toast(err.message, "err");
    }
}

function openCreateChannelModal(parentId, parentName) {
    const body = el("div");
    body.innerHTML = `
        <label class="field"><span>类型</span>
            <div style="display:flex;gap:10px;">
                <label style="flex:1;display:flex;gap:6px;align-items:center;border:1px solid var(--line);border-radius:10px;padding:10px;cursor:pointer;">
                    <input type="radio" name="m-ch-type" value="TEXT" checked> ⬡ 文字频道</label>
                <label style="flex:1;display:flex;gap:6px;align-items:center;border:1px solid var(--line);border-radius:10px;padding:10px;cursor:pointer;">
                    <input type="radio" name="m-ch-type" value="CATEGORY"> 📁 分区（可嵌套）</label>
            </div></label>
        <label class="field"><span>名称</span><input id="m-ch-name" maxlength="30" placeholder="新频道叫什么"></label>
        <label class="field"><span>主题（可选）</span><input id="m-ch-topic" maxlength="100" placeholder="这个频道用来聊什么"></label>`;
    showModal({
        title: parentId ? `在「${parentName}」内新建` : "新建频道 / 分区",
        sub: "分区里还可以再建分区，实现群中群", body,
        actions: [
            { label: "取消", kind: "ghost" },
            {
                label: "创建", onClick: async (close) => {
                    const name = $("m-ch-name").value.trim();
                    if (!name) { toast("名称不能为空", "err"); return; }
                    const type = document.querySelector('input[name="m-ch-type"]:checked').value;
                    try {
                        await api.post(`/hives/${state.currentHiveId}/channels`, {
                            name, type, parentId: parentId ?? null,
                            topic: $("m-ch-topic").value.trim(),
                        });
                        close(); // 树由 CHANNELS_CHANGED 广播刷新
                    } catch (err) { toast(err.message, "err"); }
                },
            },
        ],
    });
}

function openEditChannelModal(c) {
    const body = el("div");
    body.innerHTML = `
        <label class="field"><span>名称</span><input id="m-ch-name" maxlength="30" value="${esc(c.name)}"></label>
        <label class="field"><span>主题</span><input id="m-ch-topic" maxlength="100" value="${esc(c.topic ?? "")}"></label>`;
    showModal({
        title: "编辑频道", body,
        actions: [
            { label: "取消", kind: "ghost" },
            {
                label: "保存", onClick: async (close) => {
                    try {
                        await api.put(`/channels/${c.id}`, {
                            name: $("m-ch-name").value.trim(),
                            topic: $("m-ch-topic").value.trim(),
                        });
                        close();
                    } catch (err) { toast(err.message, "err"); }
                },
            },
        ],
    });
}

function openHiveSettingsModal() {
    const d = state.detail;
    const body = el("div");
    body.innerHTML = `
        <label class="field"><span>蜂巢名称</span><input id="m-hs-name" maxlength="30" value="${esc(d.name)}"></label>
        <label class="field"><span>简介</span><input id="m-hs-desc" maxlength="100" value="${esc(d.description ?? "")}"></label>
        <label class="field"><span>主题色</span></label>`;
    const colors = colorRow(d.iconColor);
    body.appendChild(colors.node);
    showModal({
        title: "蜂巢设置", body,
        actions: [
            { label: "取消", kind: "ghost" },
            {
                label: "保存", onClick: async (close) => {
                    try {
                        await api.put(`/hives/${d.id}`, {
                            name: $("m-hs-name").value.trim(),
                            description: $("m-hs-desc").value.trim(),
                            iconColor: colors.value,
                        });
                        close();
                        state.hives = await api.get("/hives");
                        renderRail();
                        refreshHiveDetail();
                        toast("已保存", "ok");
                    } catch (err) { toast(err.message, "err"); }
                },
            },
        ],
    });
}

function openProfileModal() {
    const body = el("div");
    body.innerHTML = `
        <label class="field"><span>昵称</span><input id="m-p-nick" maxlength="16" value="${esc(state.me.nickname)}"></label>
        <label class="field"><span>个性签名</span><input id="m-p-bio" maxlength="100" value="${esc(state.me.bio ?? "")}" placeholder="写点什么"></label>
        <label class="field"><span>头像颜色</span></label>`;
    const colors = colorRow(state.me.avatarColor);
    body.appendChild(colors.node);
    showModal({
        title: "个人资料", sub: `@${state.me.username}`, body,
        actions: [
            { label: "取消", kind: "ghost" },
            {
                label: "保存", onClick: async (close) => {
                    try {
                        state.me = await api.put("/users/me", {
                            nickname: $("m-p-nick").value.trim(),
                            bio: $("m-p-bio").value.trim(),
                            avatarColor: colors.value,
                        });
                        renderMeBar();
                        close();
                        toast("已保存", "ok");
                    } catch (err) { toast(err.message, "err"); }
                },
            },
        ],
    });
}

function renderMeBar() {
    const slot = $("me-avatar");
    slot.replaceWith(Object.assign(hexAvatar(state.me, "small"), { id: "me-avatar" }));
    $("me-name").textContent = state.me.nickname;
    $("me-username").textContent = `@${state.me.username}`;
}

function hiveMenu(e) {
    if (!state.detail) return;
    const items = [];
    if (can(P.INVITE)) items.push({ icon: "✉️", label: "邀请成员", onClick: openInviteModal });
    if (can(P.MANAGE_CHANNELS)) items.push({ icon: "⬡", label: "新建频道 / 分区", onClick: () => openCreateChannelModal(null, null) });
    if (can(P.MANAGE_HIVE)) items.push({ icon: "⚙️", label: "蜂巢设置", onClick: openHiveSettingsModal });
    if (items.length) items.push("-");
    if (isOwner()) {
        items.push({
            icon: "💥", label: "解散蜂巢", danger: true,
            onClick: () => confirmModal("解散蜂巢", `确定解散「${state.detail.name}」吗？所有频道与消息将永久删除！`, async () => {
                try {
                    await api.del(`/hives/${state.detail.id}`);
                    state.hives = state.hives.filter((h) => h.id !== state.detail.id);
                    renderRail();
                    state.hives.length ? selectHive(state.hives[0].id) : renderEmptyWorld();
                } catch (err) { toast(err.message, "err"); }
            }),
        });
    } else {
        items.push({
            icon: "🚪", label: "退出蜂巢", danger: true,
            onClick: () => confirmModal("退出蜂巢", `确定退出「${state.detail.name}」吗？`, async () => {
                try {
                    await api.post(`/hives/${state.detail.id}/leave`);
                    state.hives = state.hives.filter((h) => h.id !== state.detail.id);
                    renderRail();
                    state.hives.length ? selectHive(state.hives[0].id) : renderEmptyWorld();
                } catch (err) { toast(err.message, "err"); }
            }),
        });
    }
    const r = e.currentTarget.getBoundingClientRect();
    showCtx(r.left, r.bottom + 6, items);
}

/* ============================================================
   主页模式：好友 + 私信
   ============================================================ */

function setComposerVisible(visible) {
    document.querySelector(".composer").classList.toggle("hidden", !visible);
}

async function refreshDms() {
    try {
        state.dms = await api.get("/dms");
    } catch {
        return;
    }
    if (state.mode === "home") renderDmList();
    updateHomeDot();
}

async function refreshFriendsData() {
    try {
        const [friends, requests] = await Promise.all([
            api.get("/friends"), api.get("/friends/requests"),
        ]);
        state.friends = friends;
        state.requests = requests;
    } catch {
        return;
    }
    updateHomeDot();
}

async function refreshAllHome() {
    await Promise.all([refreshFriendsData(), refreshDms()]);
    if (state.mode === "home") {
        renderDmList();
        renderFriendsView();
        renderHomeMembers();
    }
}

/** 主页红点：未读私信总数 + 待处理申请数 */
function updateHomeDot() {
    const unread = state.dms.reduce(
        (sum, d) => sum + (d.channelId === state.currentChannelId ? 0 : (d.unread ?? 0)), 0);
    const n = unread + state.requests.length;
    $("home-dot").classList.toggle("hidden", n === 0);
}

async function enterHome() {
    state.mode = "home";
    state.currentChannelId = null;
    state.replyTo = null;
    hideReplyBar();
    state.typing.clear();
    renderTyping();
    $("btn-home").classList.add("active");
    $("btn-hive-menu").style.visibility = "hidden";
    $("hive-name").textContent = "好友与私信";
    $("channel-name").textContent = "好友";
    $("channel-topic").textContent = "";
    $("members-title").textContent = "好友";
    setComposerVisible(false);
    renderRail();
    await refreshAllHome();
    renderDmList();
    renderFriendsView();
    renderHomeMembers();
}

/** 侧栏：好友入口 + 私信会话列表 */
function renderDmList() {
    if (state.mode !== "home") return;
    const tree = $("channel-tree");
    tree.innerHTML = "";

    const friendsRow = el("div", "dm-row" + (!state.currentChannelId ? " active" : ""));
    friendsRow.appendChild(el("span", "ch-hex", "🐝"));
    friendsRow.appendChild(el("span", "ch-name", "好友"));
    if (state.requests.length) {
        friendsRow.appendChild(el("span", "unread-badge", String(state.requests.length)));
    }
    friendsRow.onclick = () => {
        state.currentChannelId = null;
        $("channel-name").textContent = "好友";
        $("channel-topic").textContent = "";
        setComposerVisible(false);
        renderDmList();
        renderFriendsView();
    };
    tree.appendChild(friendsRow);

    tree.appendChild(el("div", "section-label", `私信 — ${state.dms.length}`));
    for (const d of state.dms) {
        const row = el("div", "dm-row" + (d.channelId === state.currentChannelId ? " active" : ""));
        row.appendChild(hexAvatar(d, "small"));
        const main = el("div", "dm-main");
        const name = el("div", "dm-name");
        name.appendChild(el("span", "", d.nickname));
        if (d.lastAt) name.appendChild(el("span", "dm-time", fmtTime(d.lastAt)));
        main.appendChild(name);
        const preview = !d.lastContent ? "开始聊天吧"
            : d.lastContent.startsWith("/uploads/") ? "[图片]" : d.lastContent;
        main.appendChild(el("div", "dm-preview", preview));
        row.appendChild(main);
        const unread = d.channelId === state.currentChannelId ? 0 : (d.unread ?? 0);
        if (unread > 0) row.appendChild(el("span", "unread-badge", unread > 99 ? "99+" : String(unread)));
        row.onclick = () => selectDm(d);
        tree.appendChild(row);
    }
    if (!state.dms.length) {
        tree.appendChild(el("div", "friends-empty", "还没有私信会话"));
    }
}

/** 打开一个私信会话 */
async function selectDm(conv) {
    state.currentChannelId = conv.channelId;
    state.replyTo = null;
    hideReplyBar();
    state.messages = [];
    state.oldestId = null;
    state.hasMore = true;
    state.typing.clear();
    renderTyping();
    $("channel-name").textContent = conv.nickname;
    $("channel-topic").textContent = `与 @${conv.username} 的私聊`;
    $("composer-input").placeholder = `给 ${conv.nickname} 发消息…`;
    setComposerVisible(true);
    renderDmList();
    await loadHistory(true);
    markRead();
    conv.unread = 0;
    renderDmList();
    updateHomeDot();
}

/** 与某人开启私聊（好友列表/成员菜单入口） */
async function openDmWith(userId) {
    try {
        const { channelId } = await api.post(`/dms/${userId}`);
        if (state.mode !== "home") await enterHome();
        else await refreshDms();
        const conv = state.dms.find((d) => d.channelId === channelId);
        if (conv) await selectDm(conv);
    } catch (err) {
        toast(err.message, "err");
    }
}

/** 主区域：好友管理视图（添加/申请处理/好友列表） */
function renderFriendsView() {
    if (state.mode !== "home" || state.currentChannelId) return;
    const listEl = $("msg-list");
    listEl.innerHTML = "";
    const v = el("div", "friends-view");

    v.appendChild(el("h3", "", "添加好友"));
    const addRow = el("div", "add-friend-row");
    const input = el("input");
    input.placeholder = "输入对方用户名，例如 xiaomi";
    input.maxLength = 20;
    const btn = el("button", "mini-btn gold", "发送申请");
    const doAdd = async () => {
        const name = input.value.trim();
        if (!name) return;
        try {
            await api.post("/friends/requests", { username: name });
            toast("好友申请已发送 🐝", "ok");
            input.value = "";
            refreshAllHome();
        } catch (err) {
            toast(err.message, "err");
        }
    };
    btn.onclick = doAdd;
    input.addEventListener("keydown", (e) => { if (e.key === "Enter") doAdd(); });
    addRow.append(input, btn);
    v.appendChild(addRow);

    if (state.requests.length) {
        v.appendChild(el("h3", "", `待处理申请 — ${state.requests.length}`));
        for (const r of state.requests) {
            const row = el("div", "friend-row");
            row.appendChild(hexAvatar(r, "small"));
            const main = el("div", "friend-main");
            main.appendChild(el("div", "friend-name", r.nickname));
            main.appendChild(el("div", "friend-sub", `@${r.username}`));
            row.appendChild(main);
            const acts = el("div", "friend-actions");
            const yes = el("button", "mini-btn gold", "接受");
            yes.onclick = async () => {
                try { await api.post(`/friends/requests/${r.id}/accept`); refreshAllHome(); }
                catch (err) { toast(err.message, "err"); }
            };
            const no = el("button", "mini-btn red", "拒绝");
            no.onclick = async () => {
                try { await api.del(`/friends/requests/${r.id}`); refreshAllHome(); }
                catch (err) { toast(err.message, "err"); }
            };
            acts.append(yes, no);
            row.appendChild(acts);
            v.appendChild(row);
        }
    }

    v.appendChild(el("h3", "", `我的好友 — ${state.friends.length}`));
    if (!state.friends.length) {
        v.appendChild(el("div", "friends-empty", "还没有好友。输入用户名发送申请试试（演示账号互加：afeng / xiaomi / wengweng）"));
    }
    for (const f of state.friends) {
        const row = el("div", "friend-row");
        row.appendChild(hexAvatar(f, "small"));
        const main = el("div", "friend-main");
        main.appendChild(el("div", "friend-name", f.nickname));
        main.appendChild(el("div", "friend-sub",
            `@${f.username}${state.online.has(f.userId) ? " · 🟢 在线" : ""}`));
        row.appendChild(main);
        const acts = el("div", "friend-actions");
        const chat = el("button", "mini-btn gold", "发消息");
        chat.onclick = () => openDmWith(f.userId);
        const rm = el("button", "mini-btn red", "删除");
        rm.onclick = () => confirmModal("删除好友", `确定删除好友 ${f.nickname} 吗？`, async () => {
            try { await api.del(`/friends/${f.userId}`); refreshAllHome(); }
            catch (err) { toast(err.message, "err"); }
        });
        acts.append(chat, rm);
        row.appendChild(acts);
        v.appendChild(row);
    }
    listEl.appendChild(v);
}

/** 主页右栏：好友在线状态 */
function renderHomeMembers() {
    if (state.mode !== "home") return;
    const listEl = $("member-list");
    listEl.innerHTML = "";
    $("member-count").textContent = ` · ${state.friends.length}`;
    const online = state.friends.filter((f) => state.online.has(f.userId));
    const offline = state.friends.filter((f) => !state.online.has(f.userId));
    const section = (label, arr, off) => {
        if (!arr.length) return;
        listEl.appendChild(el("div", "section-label", `${label} — ${arr.length}`));
        for (const f of arr) {
            const row = el("div", "member-row" + (off ? " offline" : ""));
            row.appendChild(hexAvatar(f, "small"));
            row.appendChild(el("span", "presence-dot"));
            const main = el("div", "member-main");
            main.appendChild(el("div", "member-name", f.nickname));
            main.appendChild(el("div", "member-sub", `@${f.username}`));
            row.appendChild(main);
            row.onclick = () => openDmWith(f.userId);
            listEl.appendChild(row);
        }
    };
    section("在线", online, false);
    section("离线", offline, true);
}

/* ============================================================
   静态控件绑定
   ============================================================ */

function autoSize(ta) {
    ta.style.height = "auto";
    ta.style.height = Math.min(ta.scrollHeight, 130) + "px";
}

function bindStatic() {
    bindLightbox();
    $("btn-home").onclick = enterHome;
    $("btn-create-hive").onclick = openCreateHiveModal;
    $("btn-join-hive").onclick = openJoinHiveModal;
    $("btn-hive-menu").onclick = hiveMenu;
    $("btn-profile").onclick = openProfileModal;
    $("btn-logout").onclick = () => confirmModal("退出登录", "确定要退出吗？", logout, "退出");
    $("btn-send").onclick = sendMessage;
    $("btn-cancel-reply").onclick = () => { state.replyTo = null; hideReplyBar(); };
    $("btn-attach").onclick = () => $("file-input").click();
    $("file-input").onchange = (e) => { sendImage(e.target.files[0]); e.target.value = ""; };

    const input = $("composer-input");
    input.addEventListener("keydown", (e) => {
        if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendMessage(); }
    });
    input.addEventListener("input", () => {
        autoSize(input);
        const now = Date.now();
        if (input.value && now - state.lastTypingSent > 2500 && state.currentChannelId) {
            state.lastTypingSent = now;
            state.socket?.send("TYPING", { channelId: state.currentChannelId });
        }
    });

    $("btn-emoji").onclick = (e) => {
        e.stopPropagation();
        showEmojiPop(e.currentTarget, (emoji) => {
            const pos = input.selectionStart ?? input.value.length;
            input.value = input.value.slice(0, pos) + emoji + input.value.slice(pos);
            input.focus();
            input.selectionStart = input.selectionEnd = pos + emoji.length;
        });
    };

    $("msg-scroll").addEventListener("scroll", () => {
        if ($("msg-scroll").scrollTop < 60) loadHistory(false);
    });

    window.addEventListener("focus", () => { if (state.currentChannelId) markRead(); });
    window.addEventListener("keydown", (e) => {
        if (e.key === "Escape") { closeModal(); closeCtx(); hideEmojiPop(); }
    });
}

boot();
