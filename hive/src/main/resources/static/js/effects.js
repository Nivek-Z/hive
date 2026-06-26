// 特效层：全屏彩蛋粒子 + 成就金色弹窗 + Konami 秘技监听（纯 CSS 动画，零依赖）
import { el, esc } from "./ui.js";

function layer(ttl) {
    const box = el("div", "fx-layer");
    document.body.appendChild(box);
    setTimeout(() => box.remove(), ttl);
    return box;
}

/** 全屏彩带（生日快乐 / 🎉 关键词触发） */
export function confetti() {
    const box = layer(4500);
    const colors = ["#ffb300", "#ff7043", "#ec407a", "#ab47bc",
        "#5c6bc0", "#29b6f6", "#9ccc65", "#ffd166"];
    for (let i = 0; i < 90; i++) {
        const p = el("i", "fx-confetti");
        const size = 6 + Math.random() * 8;
        p.style.cssText = `left:${Math.random() * 100}vw;width:${size}px;height:${size * (0.5 + Math.random())}px;`
            + `background:${colors[i % colors.length]};`
            + `animation-duration:${2.2 + Math.random() * 1.8}s;animation-delay:${Math.random() * 0.9}s;`;
        box.appendChild(p);
    }
}

/** 蜜蜂雨（🐝 关键词 / Konami 秘技触发） */
export function beeRain() {
    const box = layer(5200);
    for (let i = 0; i < 28; i++) {
        const b = el("i", "fx-bee", "🐝");
        b.style.cssText = `left:${Math.random() * 100}vw;font-size:${16 + Math.random() * 18}px;`
            + `animation-duration:${2.8 + Math.random() * 2.2}s;animation-delay:${Math.random() * 1.2}s;`;
        box.appendChild(b);
    }
}

/** 成就解锁金色横幅（多个同时弹出时向下堆叠） */
export function achievementToast(a) {
    const t = el("div", "ach-toast");
    const stacked = document.querySelectorAll(".ach-toast").length;
    if (stacked > 0) t.style.top = `${24 + stacked * 78}px`;
    t.innerHTML = `<span class="ach-emoji">${esc(a.emoji)}</span>`
        + `<span class="ach-text"><b>成就解锁 · ${esc(a.name)}</b><i>${esc(a.description)}</i></span>`
        + `<span class="ach-points">+${a.points ?? 0}</span>`;
    document.body.appendChild(t);
    setTimeout(() => t.classList.add("bye"), 4200);
    setTimeout(() => t.remove(), 4700);
}

const KONAMI = ["ArrowUp", "ArrowUp", "ArrowDown", "ArrowDown",
    "ArrowLeft", "ArrowRight", "ArrowLeft", "ArrowRight", "b", "a"];

/** 监听 ↑↑↓↓←→←→BA */
export function bindKonami(onTrigger) {
    let pos = 0;
    addEventListener("keydown", (e) => {
        const key = e.key.length === 1 ? e.key.toLowerCase() : e.key;
        pos = key === KONAMI[pos] ? pos + 1 : (key === KONAMI[0] ? 1 : 0);
        if (pos === KONAMI.length) {
            pos = 0;
            onTrigger();
        }
    });
}
