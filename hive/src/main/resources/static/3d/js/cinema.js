// 白境 · 电影运镜系统
// 摄像机永远处于两种状态之一：飞行（贝塞尔弧线 + 缓动）或驻留（呼吸漂移 + 鼠标视差）
import * as THREE from "three";

const smootherstep = (t) => t * t * t * (t * (t * 6 - 15) + 10);

/** 由目标点 + 球坐标参数求机位 */
function orbitPos(center, dist, az, el, out) {
    out.set(
        center.x + Math.sin(az) * Math.cos(el) * dist,
        center.y + Math.sin(el) * dist,
        center.z + Math.cos(az) * Math.cos(el) * dist,
    );
    return out;
}

export class Cinema {
    constructor(camera) {
        this.cam = camera;
        this.reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;

        // 驻留基准（飞行结束后写入）
        this.basePos = new THREE.Vector3(60, 90, 120);
        this.purePos = this.basePos.clone();   // 未叠加鼠标视差的纯机位（飞行起点用它，避免起步跳变）
        this.baseTarget = new THREE.Vector3(0, 0, 0);
        this.baseAz = 0.42;
        this.baseEl = 0.6;
        this.baseDist = 120;

        // 飞行状态
        this.flight = null; // {p0,p1,p2, t0,t1, dur, t}
        this.idleT0 = 0;    // 驻留起始时刻（漂移相位/振幅从这里渐入，保证飞行结束无顿挫）
        this._lastTime = 0;
        this.cruise = null; // 总览巡航 {a,b,s,dir,speed,lift}

        // 平滑量
        this._target = this.baseTarget.clone();
        this._mouse = new THREE.Vector2();
        this._mouseSmooth = new THREE.Vector2();
        this._tmp = new THREE.Vector3();
        this._tmp2 = new THREE.Vector3();

        camera.position.copy(this.basePos);
        camera.lookAt(this.baseTarget);
    }

    setMouse(nx, ny) { this._mouse.set(nx, ny); }

    /** 立即就位（开场前的初始机位） */
    jumpTo({ center, dist, az, el, targetLift = 0 }) {
        this.baseTarget.copy(center).y += targetLift;
        this.baseAz = az; this.baseEl = el; this.baseDist = dist;
        orbitPos(this.baseTarget, dist, az, el, this.basePos);
        this.flight = null;
        this.cam.position.copy(this.basePos);
        this.purePos.copy(this.basePos);
        this._target.copy(this.baseTarget);
        this.cam.lookAt(this._target);
        this.idleT0 = this._lastTime;
    }

    /**
     * 飞往新机位：抬升-滑翔-降落的弧线
     * shot = { center, dist, az, el, dur, lift, targetLift, shift }
     * shift：沿摄像机右方向平移构图（给右侧面板让位）
     */
    fly(shot) {
        const { center, dist, az, el, dur = 2.0, lift = 10, targetLift = 0, shift = 0 } = shot;
        const t1 = center.clone(); t1.y += targetLift;

        // 构图平移：目标点沿摄像机右方向偏移。
        // 必须先平移、再据此计算终点机位 —— 终点与驻留基准若不一致，到位瞬间会横跳一个 shift
        if (shift !== 0) {
            t1.x += Math.cos(az) * shift;
            t1.z += -Math.sin(az) * shift;
        }
        const p2 = orbitPos(t1, dist, az, el, new THREE.Vector3());

        if (this.reduced) {
            this.baseTarget.copy(t1);
            this.baseAz = az; this.baseEl = el; this.baseDist = dist;
            this.basePos.copy(p2);
            this.flight = null;
            this.idleT0 = this._lastTime;
            return;
        }

        const p0 = this.purePos.clone();
        const p1 = p0.clone().lerp(p2, 0.5);
        p1.y = Math.max(p0.y, p2.y) + lift;       // 弧顶抬升

        this.flight = {
            p0, p1, p2,
            t0: this._target.clone(), t1,
            dur, t: 0,
        };
        this.baseAz = az; this.baseEl = el; this.baseDist = dist;
        this.basePos.copy(p2);
        this.baseTarget.copy(t1);
    }

    get flying() { return this.flight !== null; }

    /**
     * 总览巡航：飞行结束后目标点沿 a→b 线缓缓游走（乒乓往返）。
     * s0 取入场点在线段上的参数，与入场飞行终点严格衔接。
     */
    setCruise(c) {
        this.cruise = c ? {
            a: c.a.clone(), b: c.b.clone(),
            s: Math.min(Math.max(c.s0 ?? 0, 0), 1),
            dir: c.dir ?? 1, speed: 0, lift: c.lift ?? 1.5,
        } : null;
    }

    update(dt, time) {
        this._lastTime = time;
        // 鼠标视差缓动
        this._mouseSmooth.lerp(this._mouse, 1 - Math.exp(-dt * 4));

        if (this.flight) {
            const f = this.flight;
            f.t += dt;
            const k = smootherstep(Math.min(f.t / f.dur, 1));
            const u = 1 - k;
            // 二次贝塞尔
            this.cam.position.set(
                u * u * f.p0.x + 2 * u * k * f.p1.x + k * k * f.p2.x,
                u * u * f.p0.y + 2 * u * k * f.p1.y + k * k * f.p2.y,
                u * u * f.p0.z + 2 * u * k * f.p1.z + k * k * f.p2.z,
            );
            this._target.lerpVectors(f.t0, f.t1, k);
            if (f.t >= f.dur) { this.flight = null; this.idleT0 = time; }
        } else {
            // 驻留：极缓慢的轨道漂移 + 高度呼吸
            // 相位从驻留时刻起算（正弦零起步）且振幅渐入 → 与飞行终点位置/速度双连续
            const u = time - this.idleT0;
            const g = Math.min(u / 7, 1) * smootherstep(Math.min(u / 7, 1));
            let azWobble = 0;

            // 总览巡航：速度渐入 + 端点减速回弯 + 缓慢 S 形方位摆动（航拍长镜头）
            if (this.cruise) {
                const c = this.cruise;
                const len = Math.max(c.a.distanceTo(c.b), 1);
                c.speed = Math.min(c.speed + dt * 1.2, 4.2);
                const edge = Math.min(Math.min(c.s, 1 - c.s) * 6 + 0.1, 1);
                c.s += (c.dir * c.speed * edge * dt) / len;
                if (c.s >= 1) { c.s = 1; c.dir = -1; }
                if (c.s <= 0) { c.s = 0; c.dir = 1; }
                this.baseTarget.lerpVectors(c.a, c.b, c.s);
                this.baseTarget.y += c.lift;
                this._target.copy(this.baseTarget);
                azWobble = Math.sin(u * 0.11) * 0.16 * g;
            }

            const driftAz = this.baseAz + azWobble + Math.sin(u * 0.28) * 0.045 * g;
            const driftEl = this.baseEl + Math.sin(u * 0.37) * 0.012 * g;
            orbitPos(this.baseTarget, this.baseDist, driftAz, driftEl, this.cam.position);
            this.cam.position.y += Math.sin(u * 0.5) * 0.25 * g;
        }

        // 记录纯机位，再叠加视差（视差只作用于渲染，不进入飞行起点）
        this.purePos.copy(this.cam.position);
        const right = this._tmp.set(1, 0, 0).applyQuaternion(this.cam.quaternion);
        const up = this._tmp2.set(0, 1, 0);
        this.cam.position.addScaledVector(right, this._mouseSmooth.x * 1.1);
        this.cam.position.addScaledVector(up, -this._mouseSmooth.y * 0.6);

        this.cam.lookAt(this._target);
    }
}
