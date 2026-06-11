// UI 原语：DOM 构建 / 格式化 / 模态框 / 右键菜单 / 提示 / 表情面板 / 灯箱
export const $ = (id) => document.getElementById(id);

export function el(tag, cls, text) {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined) n.textContent = text;
    return n;
}

export const esc = (s) => String(s ?? "")
    .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#39;");

// ---------- 时间格式化 ----------

const sameDay = (a, b) =>
    a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();

export function fmtTime(iso) {
    const d = new Date(iso);
    const hm = `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
    return sameDay(d, new Date()) ? hm : `${d.getMonth() + 1}-${String(d.getDate()).padStart(2, "0")} ${hm}`;
}

export function fmtDay(iso) {
    const d = new Date(iso);
    const today = new Date();
    if (sameDay(d, today)) return "今天";
    const yesterday = new Date(today.getTime() - 86400000);
    if (sameDay(d, yesterday)) return "昨天";
    return `${d.getFullYear()} 年 ${d.getMonth() + 1} 月 ${d.getDate()} 日`;
}

export const isSameDayIso = (a, b) => sameDay(new Date(a), new Date(b));

// ---------- 六边形头像 ----------

export function hexAvatar(user, sizeCls = "") {
    // user: {nickname/senderNickname, avatarColor, avatarUrl}
    const node = el("div", `hexavatar ${sizeCls}`);
    const name = user.nickname || user.senderNickname || "?";
    if (user.avatarUrl) {
        node.style.backgroundImage = `url(${user.avatarUrl})`;
    } else {
        node.style.background = user.avatarColor || "#FFB300";
        node.textContent = [...name][0] ?? "?";
    }
    return node;
}

// ---------- 消息正文渲染（转义 → 链接 → @提及） ----------

export function renderContent(text) {
    let html = esc(text);
    html = html.replace(/(https?:\/\/[^\s<]+)/g, (m) => `<a href="${m}" target="_blank" rel="noopener">${m}</a>`);
    html = html.replace(/@([^\s@<>]{1,16})/g, '<span class="mention">@$1</span>');
    return html;
}

// ---------- 模态框 ----------

let activeModalClose = null;

export function showModal({ title, sub, body, actions = [], onClose }) {
    closeModal();
    const mask = el("div", "modal-mask");
    const box = el("div", "modal");
    if (title) box.appendChild(el("div", "modal-title", title));
    if (sub) box.appendChild(el("div", "modal-sub", sub));
    if (body) box.appendChild(body);

    if (actions.length) {
        const bar = el("div", "modal-actions");
        for (const a of actions) {
            const btn = el("button", a.kind === "danger" ? "btn-danger" : a.kind === "ghost" ? "btn-ghost" : "btn-honey", a.label);
            btn.onclick = () => a.onClick ? a.onClick(close) : close();
            bar.appendChild(btn);
        }
        box.appendChild(bar);
    }
    mask.appendChild(box);
    mask.onmousedown = (e) => { if (e.target === mask) close(); };
    $("modal-root").appendChild(mask);

    function close() {
        mask.remove();
        activeModalClose = null;
        onClose?.();
    }
    activeModalClose = close;
    // 自动聚焦第一个输入框
    setTimeout(() => box.querySelector("input,textarea")?.focus(), 30);
    return close;
}

export const closeModal = () => activeModalClose?.();

export function confirmModal(title, text, onYes, yesLabel = "确认", danger = true) {
    const body = el("div", "", text);
    body.style.cssText = "color:var(--text-mid);font-size:13.5px;line-height:1.7;";
    showModal({
        title, body,
        actions: [
            { label: "取消", kind: "ghost" },
            { label: yesLabel, kind: danger ? "danger" : "", onClick: (close) => { close(); onYes(); } },
        ],
    });
}

// ---------- 右键菜单 ----------

export function showCtx(x, y, items) {
    closeCtx();
    const menu = el("div", "ctx-menu");
    for (const it of items) {
        if (it === "-") { menu.appendChild(el("div", "ctx-sep")); continue; }
        const btn = el("button", "ctx-item" + (it.danger ? " danger" : ""));
        btn.innerHTML = `<span>${it.icon ?? ""}</span><span>${esc(it.label)}</span>`;
        btn.onclick = () => { closeCtx(); it.onClick?.(); };
        menu.appendChild(btn);
    }
    $("ctx-root").appendChild(menu);
    const r = menu.getBoundingClientRect();
    menu.style.left = Math.min(x, innerWidth - r.width - 8) + "px";
    menu.style.top = Math.min(y, innerHeight - r.height - 8) + "px";
    setTimeout(() => addEventListener("mousedown", closeCtxOnce, { once: true }), 0);
}
const closeCtxOnce = () => closeCtx();
export const closeCtx = () => { $("ctx-root").innerHTML = ""; };

// ---------- 轻提示 ----------

export function toast(msg, kind = "") {
    const t = el("div", `toast ${kind}`, msg);
    $("toast-root").appendChild(t);
    setTimeout(() => { t.classList.add("bye"); setTimeout(() => t.remove(), 350); }, 2600);
}

// ---------- 表情面板 ----------

export const EMOJIS = ["😀", "😂", "🤣", "😅", "😊", "😍", "🤔", "😭",
    "😡", "😱", "🥰", "😎", "🙄", "😴", "🤡", "👻",
    "👍", "👎", "👏", "🙏", "❤️", "🔥", "🎉", "💯",
    "🐝", "🍯", "🌸", "🌙", "⭐", "🍉", "🧋", "🚀"];

export function showEmojiPop(anchor, onPick) {
    const pop = $("emoji-pop");
    pop.innerHTML = "";
    pop.classList.remove("hidden");
    for (const e of EMOJIS) {
        const cell = el("button", "emoji-cell", e);
        cell.onclick = (ev) => { ev.stopPropagation(); hideEmojiPop(); onPick(e); };
        pop.appendChild(cell);
    }
    const r = anchor.getBoundingClientRect();
    const pr = pop.getBoundingClientRect();
    pop.style.left = Math.max(8, Math.min(r.left, innerWidth - pr.width - 8)) + "px";
    pop.style.top = Math.max(8, r.top - pr.height - 8) + "px";
    setTimeout(() => addEventListener("mousedown", emojiAway), 0);
}
function emojiAway(e) {
    if (!$("emoji-pop").contains(e.target)) hideEmojiPop();
}
export function hideEmojiPop() {
    $("emoji-pop").classList.add("hidden");
    removeEventListener("mousedown", emojiAway);
}

// ---------- 灯箱 ----------

export function openLightbox(src) {
    $("lightbox-img").src = src;
    $("lightbox").classList.remove("hidden");
}
export function bindLightbox() {
    $("lightbox").onclick = () => $("lightbox").classList.add("hidden");
}

// ---------- 颜色选择行 ----------

export const PALETTE = ["#FFB300", "#FF7043", "#EC407A", "#AB47BC",
    "#5C6BC0", "#29B6F6", "#26A69A", "#9CCC65"];

export function colorRow(initial) {
    const row = el("div", "color-row");
    let picked = initial || PALETTE[0];
    for (const c of PALETTE) {
        const cell = el("button", "color-cell" + (c === picked ? " picked" : ""));
        cell.type = "button";
        cell.style.background = c;
        cell.onclick = () => {
            picked = c;
            row.querySelectorAll(".color-cell").forEach((n) => n.classList.remove("picked"));
            cell.classList.add("picked");
        };
        row.appendChild(cell);
    }
    return { node: row, get value() { return picked; } };
}
