// 白境 · 瓷白三维世界（光秃柱体基线）
// 蜂巢 = 一根干净的白瓷六棱柱（高度随成员数生长）
// 频道 = 环绕的矮六棱柱（高度随消息活跃度生长）
// 私域 = 林荫道反向的圆柱群落
// 保留：白天/黑夜主题、跨巢光流、消息脉冲、在线萤点、高级感光照与大气层
import * as THREE from "three";

const HONEY = 0xffb300;

/** 双主题色表：暖灰米现代白 / 墨岩黑夜（月光 + 琥珀灯火） */
const THEMES = {
    light: {
        bg: 0xe9eae5, ground: 0xe2e3dd, tile: 0xf5f5ef, decor: 0xdcddd6,
        line: 0xc2c2b6, runway: 0xc6c6bb, motes: 0xb9b9ae,
        hemiSky: 0xf6f7f4, hemiGround: 0xd8d8d0, hemiI: 1.05,
        sun: 0xfff3df, sunI: 1.3, exposure: 0.98,
        fogNear: 70, fogFar: 250, tileGlow: 0x000000,
        skyTop: 0xdcdfdc, rim: 0xd5dfeb, rimI: 0.45,
        envI: 0.5, ao: 0.2, star: 0,
    },
    dark: {
        bg: 0x161614, ground: 0x232320, tile: 0x787567, decor: 0x45443c,
        line: 0x6a675a, runway: 0x706d5f, motes: 0x6d6a5c,
        hemiSky: 0x707694, hemiGround: 0x282832, hemiI: 1.1,
        sun: 0xd2daff, sunI: 1.4, exposure: 1.08,
        fogNear: 90, fogFar: 330, tileGlow: 0x0d0d12,
        skyTop: 0x090a10, rim: 0x8a7244, rimI: 0.7,
        envI: 0.28, ao: 0.34, star: 0.85,
    },
};

/* 确定性伪随机 */
function mulberry32(seed) {
    return () => {
        seed |= 0; seed = (seed + 0x6d2b79f5) | 0;
        let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
        t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
}

const easeOutCubic = (t) => 1 - Math.pow(1 - t, 3);
const easeOutBack = (t) => { const c = 1.4; return 1 + (c + 1) * Math.pow(t - 1, 3) + c * Math.pow(t - 1, 2); };
const smoothstep = (t) => t * t * (3 - 2 * t);

/** 六角螺旋槽位（跳过中心） */
function hexSpiralSlots(count, spacing) {
    const out = [];
    const DIRS = [[1, 0], [1, -1], [0, -1], [-1, 0], [-1, 1], [0, 1]];
    for (let ring = 1; out.length < count; ring++) {
        let q = -ring, r = ring;
        for (let side = 0; side < 6 && out.length < count; side++) {
            for (let step = 0; step < ring && out.length < count; step++) {
                out.push({
                    x: spacing * Math.sqrt(3) * (q + r / 2),
                    z: spacing * 1.5 * r,
                });
                q += DIRS[side][0]; r += DIRS[side][1];
            }
        }
    }
    return out;
}

export class World {
    constructor(canvas) {
        this.canvas = canvas;
        this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true });
        this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
        this.renderer.toneMappingExposure = 0.98;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(THEMES.light.bg);
        this.scene.fog = new THREE.Fog(THEMES.light.bg, 70, 250);

        this.camera = new THREE.PerspectiveCamera(38, 1, 0.1, 700);

        // ---- 光照 ----
        this.hemi = new THREE.HemisphereLight(0xf6f7f4, 0xd8d8d0, 1.05);
        this.scene.add(this.hemi);
        const sun = new THREE.DirectionalLight(0xfff3df, 1.3);
        sun.castShadow = true;
        sun.shadow.mapSize.set(2048, 2048);
        const sc = sun.shadow.camera;
        sc.left = -80; sc.right = 80; sc.top = 80; sc.bottom = -80; sc.far = 280;
        sun.shadow.bias = -0.0004;
        sun.shadow.normalBias = 0.03;
        this.sun = sun;
        this.sunTarget = new THREE.Object3D();
        this.scene.add(sun, this.sunTarget);
        sun.target = this.sunTarget;
        this.focusCenter = new THREE.Vector3();
        this.focusSmooth = new THREE.Vector3();

        // 轮缘补光（背侧勾边，无阴影）
        this.rim = new THREE.DirectionalLight(0xd5dfeb, 0.45);
        this.rim.position.set(-60, 70, -90);
        this.scene.add(this.rim);

        // ---- 地面 ----
        this.matGround = new THREE.MeshStandardMaterial({ color: THEMES.light.ground, roughness: 1 });
        const ground = new THREE.Mesh(new THREE.CircleGeometry(1000, 64), this.matGround);
        ground.rotation.x = -Math.PI / 2;
        ground.receiveShadow = true;
        this.scene.add(ground);
        this.themeName = "light";

        // ---- 共享材质 / 几何缓存 ----
        // 三档微差瓷面（亮度 ±3.8%）打破大面积同色的塑料感
        this.tileMults = [1.0, 0.962, 1.038];
        this.matTiles = this.tileMults.map((m) => new THREE.MeshStandardMaterial({
            color: new THREE.Color(THEMES.light.tile).multiplyScalar(m), roughness: 0.55,
        }));
        this.matTile = this.matTiles[0];
        this._geoCache = new Map();
        this._tileSeq = 0;

        // ---- 状态容器 ----
        this.hiveGroups = new Map();
        this.channelTiles = new Map();
        this.dmTiles = new Map();
        this.tweens = [];
        this.streams = [];
        this.currentHiveId = null;
        this.activeTileId = null;
        this.activeRing = null;
        this.hoverId = null;

        // ---- 蜜蜂（在线成员萤点） ----
        this.bees = [];
        const beeGeo = new THREE.IcosahedronGeometry(0.16, 1);
        const beeMat = new THREE.MeshBasicMaterial({ color: 0xeda400 });
        for (let i = 0; i < 14; i++) {
            const b = new THREE.Mesh(beeGeo, beeMat);
            b.visible = false;
            b.userData = { r: 5 + (i % 5) * 1.15, h: 4.6 + (i % 4) * 1.1, speed: 0.25 + (i % 3) * 0.1, phase: i * 1.7 };
            this.bees.push(b);
            this.scene.add(b);
        }

        // ---- 拾取 ----
        this.raycaster = new THREE.Raycaster();
        this._pickables = [];
        this._v = new THREE.Vector3();

        this._buildAOTex();
        this._buildSky();
        this._buildEnv();
        this._buildStars();
        this._buildMotes();
        this.resize();
    }

    /* ============ 几何 ============ */

    _hexGeo(radius, height) {
        const key = `h${radius}:${height}`;
        if (this._geoCache.has(key)) return this._geoCache.get(key);
        const shape = new THREE.Shape();
        for (let i = 0; i < 6; i++) {
            const a = (Math.PI / 3) * i;
            const x = Math.cos(a) * radius, y = Math.sin(a) * radius;
            i === 0 ? shape.moveTo(x, y) : shape.lineTo(x, y);
        }
        shape.closePath();
        const g = new THREE.ExtrudeGeometry(shape, {
            depth: height, bevelEnabled: true,
            bevelThickness: 0.16, bevelSize: 0.16, bevelSegments: 2, curveSegments: 1,
        });
        g.rotateX(-Math.PI / 2);
        g.translate(0, 0.16, 0);
        this._geoCache.set(key, g);
        return g;
    }

    _roundGeo(radius, height) {
        const key = `r${radius}:${height}`;
        if (this._geoCache.has(key)) return this._geoCache.get(key);
        const g = new THREE.CylinderGeometry(radius, radius, height, 40);
        g.translate(0, height / 2, 0);
        this._geoCache.set(key, g);
        return g;
    }

    _tile(geo, mat = this.matTiles[this._tileSeq++ % this.matTiles.length]) {
        const m = new THREE.Mesh(geo, mat);
        m.castShadow = true;
        m.receiveShadow = true;
        m.userData = {
            baseY: 0, raise: 0, hover: 0, bobUntil: 0, bobStart: 0,
            bobPhase: Math.random() * 6, act: 0, actVis: 0, riseK: 1,
        };
        return m;
    }

    _tileOf(id) { return this.channelTiles.get(id) ?? this.dmTiles.get(id); }

    /* ============ 大气层 / 质感 ============ */

    _buildSky() {
        const t = THEMES[this.themeName];
        this.skyUni = { top: { value: new THREE.Color(t.skyTop) }, bottom: { value: new THREE.Color(t.bg) } };
        const mat = new THREE.ShaderMaterial({
            side: THREE.BackSide, depthWrite: false, fog: false, uniforms: this.skyUni,
            vertexShader: "varying vec3 vP; void main(){ vP = position; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }",
            fragmentShader: [
                "varying vec3 vP; uniform vec3 top; uniform vec3 bottom;",
                "void main(){",
                "  float h = clamp(normalize(vP).y * 1.7, 0.0, 1.0);",
                "  gl_FragColor = vec4(mix(bottom, top, h), 1.0);",
                "  #include <tonemapping_fragment>",
                "  #include <colorspace_fragment>",
                "}",
            ].join("\n"),
        });
        this.sky = new THREE.Mesh(new THREE.SphereGeometry(600, 24, 12), mat);
        this.sky.renderOrder = -2;
        this.skyRig = new THREE.Group();
        this.skyRig.add(this.sky);
        this.scene.add(this.skyRig);
    }

    _buildEnv() {
        const pm = new THREE.PMREMGenerator(this.renderer);
        const env = new THREE.Scene();
        env.background = new THREE.Color(0xf4f4f0);
        const box = (w, h, d, x, y, z, c) => {
            const m = new THREE.Mesh(new THREE.BoxGeometry(w, h, d), new THREE.MeshBasicMaterial({ color: c }));
            m.position.set(x, y, z); env.add(m);
        };
        box(40, 1, 40, 0, 16, 0, 0xffffff);
        box(1, 14, 26, -18, 6, 0, 0xe9e9e2);
        box(1, 14, 26, 18, 6, -4, 0xd8d8d0);
        box(40, 1, 40, 0, -2, 0, 0xc9c9c0);
        this.scene.environment = pm.fromScene(env, 0.06).texture;
        this._envMats = [...this.matTiles, this.matGround];
        for (const m of this._envMats) m.envMapIntensity = THEMES[this.themeName].envI;
        pm.dispose();
    }

    _buildStars() {
        const rnd = mulberry32(2025);
        const pts = [];
        for (let i = 0; i < 360; i++) {
            const a = rnd() * Math.PI * 2, elv = 0.1 + rnd() * 1.35, r = 420 + rnd() * 130;
            pts.push(Math.cos(a) * Math.cos(elv) * r, Math.sin(elv) * r, Math.sin(a) * Math.cos(elv) * r);
        }
        const geo = new THREE.BufferGeometry();
        geo.setAttribute("position", new THREE.Float32BufferAttribute(pts, 3));
        this.stars = new THREE.Points(geo, new THREE.PointsMaterial({
            color: 0xcfd6e8, size: 1.3, transparent: true, opacity: 0,
            sizeAttenuation: false, fog: false, depthWrite: false,
        }));
        this.stars.renderOrder = -1;
        this.skyRig.add(this.stars);
    }

    _buildAOTex() {
        const c = document.createElement("canvas");
        c.width = c.height = 128;
        const g = c.getContext("2d");
        const grad = g.createRadialGradient(64, 64, 8, 64, 64, 64);
        grad.addColorStop(0, "rgba(0,0,0,0.9)");
        grad.addColorStop(0.55, "rgba(0,0,0,0.3)");
        grad.addColorStop(1, "rgba(0,0,0,0)");
        g.fillStyle = grad; g.fillRect(0, 0, 128, 128);
        this.matAO = new THREE.MeshBasicMaterial({
            map: new THREE.CanvasTexture(c), transparent: true,
            depthWrite: false, opacity: THEMES[this.themeName].ao,
        });
        this._aoGeo = new THREE.PlaneGeometry(1, 1);
        this._aoGeo.rotateX(-Math.PI / 2);
    }

    _addAO(parent, x, z, size, y = 0.015) {
        const m = new THREE.Mesh(this._aoGeo, this.matAO);
        m.scale.set(size, 1, size);
        m.position.set(x, y, z);
        m.renderOrder = 1;
        parent.add(m);
    }

    _buildMotes() {
        const rnd = mulberry32(31);
        const pts = [];
        for (let i = 0; i < 220; i++) pts.push((rnd() - 0.5) * 400 + 85, 2 + rnd() * 24, (rnd() - 0.5) * 400 - 55);
        const geo = new THREE.BufferGeometry();
        geo.setAttribute("position", new THREE.Float32BufferAttribute(pts, 3));
        this.motes = new THREE.Points(geo, new THREE.PointsMaterial({
            color: THEMES.light.motes, size: 0.26, transparent: true, opacity: 0.45,
            depthWrite: false, sizeAttenuation: true,
        }));
        this.scene.add(this.motes);
    }

    _buildPool(radius) {
        const pts = [];
        for (let x = -radius; x <= radius; x += 0.95) {
            for (let z = -radius; z <= radius; z += 0.95) {
                if (x * x + z * z > radius * radius) continue;
                pts.push(x + (Math.random() - 0.5) * 0.4, 0, z + (Math.random() - 0.5) * 0.4);
            }
        }
        const geo = new THREE.BufferGeometry();
        geo.setAttribute("position", new THREE.Float32BufferAttribute(pts, 3));
        return new THREE.Points(geo, new THREE.PointsMaterial({
            color: HONEY, size: 0.34, transparent: true, opacity: 0,
            depthWrite: false, blending: THREE.AdditiveBlending, sizeAttenuation: true,
        }));
    }

    _buildRunway(hiveCount) {
        if (this.runway) this.scene.remove(this.runway);
        const pts = [];
        const a = World.hivePos(0), b = World.hivePos(Math.max(hiveCount - 1, 1));
        const len = a.distanceTo(b) + 40;
        const dir = b.clone().sub(a).normalize();
        if (!Number.isFinite(dir.x)) dir.set(0.832, 0, -0.555);
        for (let s = -20; s < len; s += 2.4) pts.push(a.x + dir.x * s, 0.04, a.z + dir.z * s);
        const h = this.homeCenter();
        const hd = h.clone().sub(a).normalize();
        for (let s = 8; s < a.distanceTo(h) - 6; s += 2.4) pts.push(a.x + hd.x * s, 0.04, a.z + hd.z * s);
        const geo = new THREE.BufferGeometry();
        geo.setAttribute("position", new THREE.Float32BufferAttribute(pts, 3));
        this.runway = new THREE.Points(geo, new THREE.PointsMaterial({
            color: THEMES[this.themeName].runway, size: 0.22, transparent: true, opacity: 0.5, sizeAttenuation: true,
        }));
        this.scene.add(this.runway);
    }

    /* ============ 蜂巢林荫道 ============ */

    static hivePos(i) { return new THREE.Vector3(i * 54, 0, i * -36); }

    /** 立起所有蜂巢的中央纪念碑（光秃六棱柱，高度随成员数生长） */
    buildHives(hives) {
        hives.forEach((h, i) => {
            if (this.hiveGroups.has(h.id)) return;
            const center = World.hivePos(i);
            const group = new THREE.Group();
            group.position.copy(center);
            this.scene.add(group);

            const T = THEMES[this.themeName];
            const icon = new THREE.Color(h.iconColor || "#9a9a90");

            // 中央纪念碑：干净六棱柱
            const monument = this._tile(this._hexGeo(2.6, 3.0), this.matTiles[0]);
            monument.userData.kind = "hive";
            monument.userData.hiveId = h.id;
            monument.userData.riseK = 1;
            group.add(monument);
            this._riseTile(monument, i * 0.1 + 0.05);
            this._addAO(group, 0, 0, 9.5);

            // 身份色细环（贴地，标记领地，随主题与 iconColor 调和）
            const ringMat = new THREE.MeshBasicMaterial({
                color: new THREE.Color(T.line).lerp(icon, 0.5),
                transparent: true, opacity: 0.7, side: THREE.DoubleSide,
            });
            const ring = new THREE.Mesh(new THREE.RingGeometry(4.4, 4.62, 6), ringMat);
            ring.rotation.x = -Math.PI / 2;
            ring.position.y = 0.02;
            group.add(ring);

            // 琥珀脉冲点阵池
            const pool = this._buildPool(4.2);
            pool.position.y = 0.05;
            group.add(pool);

            // 远范围隐形拾取代理
            this._proxyGeo = this._proxyGeo ?? new THREE.CylinderGeometry(10, 10, 7, 8);
            this._proxyMat = this._proxyMat ?? new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false });
            const proxy = new THREE.Mesh(this._proxyGeo, this._proxyMat);
            proxy.position.y = 1;
            proxy.userData = { kind: "hive", hiveId: h.id, proxy: true };
            group.add(proxy);

            this.hiveGroups.set(h.id, {
                group, center, monument, pool, proxy, icon, ringMat,
                anchor: center.clone().setY(5.6),
                energy: 0, ambient: 0, channelsBuilt: false, index: i,
                towerScale: 1, towerTarget: 1,
            });
        });
        this._buildRunway(hives.length);
        this._refreshPickables();
    }

    /** 首次进入蜂巢时立起其频道柱群（光秃六棱柱） */
    ensureChannels(hiveId, channels) {
        const hg = this.hiveGroups.get(hiveId);
        if (!hg || hg.channelsBuilt) return;
        hg.channelsBuilt = true;
        const slots = hexSpiralSlots(channels.length, 3.2);
        const heights = [0.7, 0.88, 1.06, 1.24, 1.42];
        channels.forEach((c, i) => {
            const h = heights[Number(c.id) % heights.length];
            const tile = this._tile(this._hexGeo(2.0, h));
            tile.position.set(slots[i].x, 0, slots[i].z);
            tile.userData.kind = "channel";
            tile.userData.channelId = c.id;
            tile.userData.geoH = h;
            tile.userData.height = h;
            tile.userData.anchor = hg.center.clone().add(tile.position).setY(h + 2.0);
            hg.group.add(tile);
            this.channelTiles.set(c.id, tile);
            this._riseTile(tile, 0.15 + i * 0.07);
            this._addAO(hg.group, slots[i].x, slots[i].z, 6.0);
        });
        this._refreshPickables();
    }

    rebuildChannels(hiveId, channels) {
        const hg = this.hiveGroups.get(hiveId);
        if (!hg) return;
        for (const [cid, tile] of [...this.channelTiles]) {
            if (tile.parent === hg.group) { hg.group.remove(tile); this.channelTiles.delete(cid); }
        }
        hg.channelsBuilt = false;
        this.ensureChannels(hiveId, channels);
        for (const tile of this.channelTiles.values()) tile.userData.riseK = 1;
    }

    /* ============ 私域群落（圆柱 = 柔和私密形状语言） ============ */

    homeCenter() { return new THREE.Vector3(-58, 0, 44); }

    buildHome() {
        if (this.homeGroup) return;
        const g = new THREE.Group();
        g.position.copy(this.homeCenter());
        this.scene.add(g);

        const plinth = this._tile(this._roundGeo(3.0, 1.5), this.matTiles[0]);
        plinth.userData.kind = "home";
        g.add(plinth);
        this._rise(plinth, 0.1);
        this._addAO(g, 0, 0, 9);

        // 悬浮光环（私域的柔和地标）
        const halo = new THREE.Mesh(new THREE.TorusGeometry(3.8, 0.07, 10, 48), this.matTiles[0]);
        halo.rotation.x = Math.PI / 2;
        halo.position.y = 3.0;
        halo.castShadow = true;
        g.add(halo);
        this._rise(halo, 0.3);
        this.homeHalo = halo;

        const proxy = new THREE.Mesh(this._proxyGeo ?? new THREE.CylinderGeometry(10, 10, 7, 8), this._proxyMat ?? new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false }));
        proxy.position.y = 1;
        proxy.userData = { kind: "home", hiveId: "home", proxy: true };
        g.add(proxy);
        this.homeProxy = proxy;

        const pool = this._buildPool(4.0);
        pool.position.y = 0.05;
        g.add(pool);
        this.homeGroup = g;
        this.homePool = pool;
        this.homeEnergy = 0;
        this.homePlinth = plinth;
        this._refreshPickables();
    }

    ensureDmTiles(dms) {
        this.buildHome();
        const ring = Math.min(dms.length, 10);
        for (let i = 0; i < ring; i++) {
            const dm = dms[i];
            if (this.dmTiles.has(dm.channelId)) continue;
            const a = (Math.PI * 2 * i) / Math.max(ring, 5) - Math.PI / 3;
            const hh = 0.8 + (i % 3) * 0.25;
            const tile = this._tile(this._roundGeo(1.6, hh));
            tile.position.set(Math.cos(a) * 7.8, 0, Math.sin(a) * 7.8);
            tile.userData.kind = "dm";
            tile.userData.channelId = dm.channelId;
            tile.userData.geoH = hh;
            tile.userData.height = hh;
            tile.userData.anchor = this.homeCenter().add(tile.position).setY(hh + 2.0);
            this.homeGroup.add(tile);
            this.dmTiles.set(dm.channelId, tile);
            this._riseTile(tile, 0.1 + i * 0.06);
            this._addAO(this.homeGroup, tile.position.x, tile.position.z, 5.2);
        }
        this._refreshPickables();
    }

    /* ============ 动效 ============ */

    _rise(mesh, delay) {
        mesh.scale.y = 0.001;
        this.tweens.push({ t: -delay, dur: 0.9, update: (k) => { mesh.scale.y = Math.max(easeOutBack(k), 0.001); } });
    }

    _riseTile(mesh, delay) {
        mesh.userData.riseK = 0.001;
        this.tweens.push({ t: -delay, dur: 0.9, update: (k) => { mesh.userData.riseK = Math.max(easeOutBack(k), 0.001); } });
    }

    _tweenRaise(tile, to) {
        const from = tile.userData.raise;
        this.tweens.push({ t: 0, dur: 0.6, update: (k) => { tile.userData.raise = from + (to - from) * easeOutCubic(k); } });
    }

    setHover(channelId) {
        if (this.hoverId === channelId) return;
        const prev = this._tileOf(this.hoverId);
        if (prev) this._tweenHover(prev, 0);
        this.hoverId = channelId;
        const tile = this._tileOf(channelId);
        if (tile) this._tweenHover(tile, 0.22);
    }

    _tweenHover(tile, to) {
        const from = tile.userData.hover;
        this.tweens.push({ t: 0, dur: 0.35, update: (k) => { tile.userData.hover = from + (to - from) * easeOutCubic(k); } });
    }

    bob(channelId, now) {
        const tile = this._tileOf(channelId);
        if (!tile) return;
        const u = tile.userData;
        if (u.bobUntil <= now) u.bobStart = now;
        u.bobUntil = now + 2.8;
    }

    /** 跨巢光流：从 A 流向 B 的琥珀彗尾 */
    streamTo(from, to, dur = 2.0) {
        const N = 64;
        const geo = new THREE.BufferGeometry();
        geo.setAttribute("position", new THREE.Float32BufferAttribute(new Float32Array(N * 3), 3));
        const pts = new THREE.Points(geo, new THREE.PointsMaterial({
            color: HONEY, size: 0.5, transparent: true, opacity: 0.9,
            depthWrite: false, blending: THREE.AdditiveBlending, sizeAttenuation: true,
        }));
        pts.frustumCulled = false;
        this.scene.add(pts);
        this.streams.push({ pts, from: from.clone(), to: to.clone(), t: 0, dur, N });
    }

    bumpChannelActivity(channelId, amt = 1) {
        const u = this._tileOf(channelId)?.userData;
        if (u) u.act = Math.min(u.act + amt * 0.12, 1);
    }

    seedChannelActivity(channelId, recentCount) {
        const u = this._tileOf(channelId)?.userData;
        if (u) u.act = Math.max(u.act, Math.min(recentCount / 40, 1));
    }

    setHiveMembers(hiveId, count) {
        const hg = this.hiveGroups.get(hiveId);
        if (hg) hg.towerTarget = 0.85 + Math.min(count, 30) * 0.045;
    }

    setHiveAmbient(hiveId, k) {
        const hg = this.hiveGroups.get(hiveId);
        if (hg) hg.ambient = Math.min(Math.max(k, 0), 1) * 0.10;
    }

    /** 选中频道：柱体抬升 + 琥珀六角光环 */
    setActiveChannel(channelId) {
        if (this.activeTileId === channelId) return;
        const prev = this._tileOf(this.activeTileId);
        if (prev) this._tweenRaise(prev, 0);
        this.activeTileId = channelId;
        if (!this.activeRing) {
            const rg = new THREE.RingGeometry(1.9, 2.35, 6);
            rg.rotateX(-Math.PI / 2);
            this.activeRing = new THREE.Mesh(rg, new THREE.MeshBasicMaterial({
                color: HONEY, transparent: true, opacity: 0,
                blending: THREE.AdditiveBlending, depthWrite: false, side: THREE.DoubleSide,
            }));
            this.scene.add(this.activeRing);
        }
        const tile = this._tileOf(channelId);
        if (!tile) { this.activeRing.visible = false; return; }
        this._tweenRaise(tile, 0.6);
        this.activeRing.visible = true;
        this.ringT0 = this._now ?? 0;
    }

    pulse(hiveId, strength = 1) {
        if (hiveId === "home") { this.homeEnergy = Math.min((this.homeEnergy ?? 0) + strength * 0.55, 1.6); return; }
        const hg = this.hiveGroups.get(hiveId);
        if (hg) hg.energy = Math.min(hg.energy + strength * 0.55, 1.6);
    }

    setBees(n) { this.bees.forEach((b, i) => { b.visible = i < Math.min(n, this.bees.length); }); }

    setFocus(center) { this.focusCenter.copy(center); }

    /* ============ 主题切换 ============ */

    setTheme(name, instant = false) {
        const t = THEMES[name] ?? THEMES.light;
        this.themeName = name;
        const pairs = [
            [this.scene.background, t.bg],
            [this.scene.fog.color, t.bg],
            [this.matGround.color, t.ground],
            [this.hemi.color, t.hemiSky],
            [this.hemi.groundColor, t.hemiGround],
            [this.sun.color, t.sun],
            [this.rim.color, t.rim],
            [this.skyUni.top.value, t.skyTop],
            [this.skyUni.bottom.value, t.bg],
        ];
        this.matTiles.forEach((m, i) => {
            pairs.push([m.color, new THREE.Color(t.tile).multiplyScalar(this.tileMults[i])]);
            pairs.push([m.emissive, t.tileGlow]);
        });
        if (this.runway) pairs.push([this.runway.material.color, t.runway]);
        if (this.motes) pairs.push([this.motes.material.color, t.motes]);
        for (const hg of this.hiveGroups.values()) {
            pairs.push([hg.ringMat.color, new THREE.Color(t.line).lerp(hg.icon, 0.5)]);
        }

        const applyScalars = (e, s0) => {
            this.hemi.intensity = s0.hemiI + (t.hemiI - s0.hemiI) * e;
            this.sun.intensity = s0.sunI + (t.sunI - s0.sunI) * e;
            this.rim.intensity = s0.rimI + (t.rimI - s0.rimI) * e;
            this.renderer.toneMappingExposure = s0.exp + (t.exposure - s0.exp) * e;
            this.scene.fog.near = s0.fogN + (t.fogNear - s0.fogN) * e;
            this.scene.fog.far = s0.fogF + (t.fogFar - s0.fogF) * e;
            this.matAO.opacity = s0.ao + (t.ao - s0.ao) * e;
            this.stars.material.opacity = s0.star + (t.star - s0.star) * e;
            const envI = s0.envI + (t.envI - s0.envI) * e;
            for (const m of this._envMats) m.envMapIntensity = envI;
        };
        const s0 = {
            hemiI: this.hemi.intensity, sunI: this.sun.intensity, rimI: this.rim.intensity,
            exp: this.renderer.toneMappingExposure, fogN: this.scene.fog.near, fogF: this.scene.fog.far,
            ao: this.matAO.opacity, star: this.stars.material.opacity, envI: this._envMats[0].envMapIntensity,
        };

        if (instant) { for (const [c, x] of pairs) c.set(x); applyScalars(1, s0); return; }
        const from = pairs.map(([c]) => c.clone());
        const to = pairs.map(([, x]) => new THREE.Color(x));
        this.tweens.push({
            t: 0, dur: 0.9,
            update: (k) => { const e = smoothstep(k); pairs.forEach(([c], i) => c.copy(from[i]).lerp(to[i], e)); applyScalars(e, s0); },
        });
    }

    /* ============ 查询 ============ */

    hiveCenter(hiveId) { return this.hiveGroups.get(hiveId)?.center ?? new THREE.Vector3(); }
    hiveAnchor(hiveId) { return this.hiveGroups.get(hiveId)?.anchor ?? null; }
    channelAnchor(channelId) { return this._tileOf(channelId)?.userData.anchor ?? null; }

    channelWorldPos(channelId) {
        const tile = this._tileOf(channelId);
        if (!tile) return null;
        return tile.parent.position.clone().add(tile.position);
    }

    project(v3, out) {
        this._v.copy(v3).project(this.camera);
        out.x = (this._v.x * 0.5 + 0.5) * this.canvas.clientWidth;
        out.y = (-this._v.y * 0.5 + 0.5) * this.canvas.clientHeight;
        out.visible = this._v.z < 1;
        out.dist = this.camera.position.distanceTo(v3);
        return out;
    }

    _refreshPickables() {
        this._pickables = [];
        for (const t of this.channelTiles.values()) this._pickables.push(t);
        for (const t of this.dmTiles.values()) this._pickables.push(t);
        for (const hg of this.hiveGroups.values()) this._pickables.push(hg.monument, hg.proxy);
        if (this.homePlinth) this._pickables.push(this.homePlinth);
        if (this.homeProxy) this._pickables.push(this.homeProxy);
    }

    pick(nx, ny) {
        this.raycaster.setFromCamera({ x: nx, y: ny }, this.camera);
        const hits = this.raycaster.intersectObjects(this._pickables, false);
        for (const h of hits) {
            const u = h.object.userData;
            if (u.proxy && u.hiveId === this.currentHiveId) continue;
            return u;
        }
        return null;
    }

    /* ============ 主循环 ============ */

    resize() {
        const w = this.canvas.clientWidth || innerWidth;
        const h = this.canvas.clientHeight || innerHeight;
        this.renderer.setSize(w, h, false);
        this.camera.aspect = w / h;
        this.camera.updateProjectionMatrix();
    }

    _tickTile(tile, now, dt) {
        const u = tile.userData;
        if (u.act > 0.0005) u.act *= Math.exp(-dt / 600);
        u.actVis += (u.act - u.actVis) * (1 - Math.exp(-dt * 2.5));
        tile.scale.y = Math.max(u.riseK * (1 + u.actVis * 0.6), 0.001);
        if (u.geoH) {
            u.height = u.geoH * tile.scale.y;
            if (u.anchor) u.anchor.y = u.baseY + u.height + 2.0;
        }
        let bobbing = 0;
        if (u.bobUntil > now) {
            const amp = Math.max(Math.min((now - u.bobStart) / 0.4, (u.bobUntil - now) / 0.4, 1), 0);
            bobbing = (Math.sin(now * 6 + u.bobPhase) * 0.1 + 0.1) * amp;
        }
        tile.position.y = u.baseY + u.raise + u.hover + bobbing;
    }

    update(dt, now) {
        // 补间
        for (let i = this.tweens.length - 1; i >= 0; i--) {
            const tw = this.tweens[i];
            tw.t += dt;
            if (tw.t < 0) continue;
            const k = Math.min(tw.t / tw.dur, 1);
            tw.update(k);
            if (k >= 1) this.tweens.splice(i, 1);
        }

        if (this.skyRig) this.skyRig.position.copy(this.camera.position);

        // 焦点平滑随行：阳光阴影 / 蜜蜂
        this._now = now;
        this.focusSmooth.lerp(this.focusCenter, 1 - Math.exp(-dt * 2.5));
        this.sun.position.set(this.focusSmooth.x + 40, 64, this.focusSmooth.z + 32);
        this.sunTarget.position.copy(this.focusSmooth);

        // 柱体抬升 / 浮动 / 活跃度生长
        for (const tile of this.channelTiles.values()) this._tickTile(tile, now, dt);
        for (const tile of this.dmTiles.values()) this._tickTile(tile, now, dt);

        // 琥珀光环跟随活跃柱体（淡入）
        if (this.activeRing?.visible) {
            const tile = this._tileOf(this.activeTileId);
            if (tile) {
                tile.getWorldPosition(this.activeRing.position);
                this.activeRing.position.y = tile.position.y + (tile.userData.height ?? 0.9) + 0.22;
                const fadeIn = Math.min((now - (this.ringT0 ?? 0)) / 0.6, 1);
                this.activeRing.material.opacity = (0.45 + Math.sin(now * 2.4) * 0.18) * fadeIn;
                const s = 1 + Math.sin(now * 2.4) * 0.03;
                this.activeRing.scale.set(s, 1, s);
            }
        }

        // 蜂巢主塔随成员数生长 + 脉冲/常驻微光
        const decay = Math.exp(-dt * 1.4);
        const grow = 1 - Math.exp(-dt * 2);
        for (const hg of this.hiveGroups.values()) {
            hg.towerScale += (hg.towerTarget - hg.towerScale) * grow;
            const tu = hg.monument.userData;
            hg.monument.scale.y = Math.max(tu.riseK * hg.towerScale, 0.001);
            hg.anchor.y = 3.0 * hg.monument.scale.y + 2.6;
            hg.energy *= decay;
            hg.pool.material.opacity = Math.min(hg.energy + hg.ambient, 0.85);
        }
        if (this.homePool) {
            this.homeEnergy *= decay;
            this.homePool.material.opacity = Math.min(this.homeEnergy, 0.85);
        }
        if (this.homeHalo) {
            this.homeHalo.position.y = 3.0 + Math.sin(now * 0.8) * 0.15;
            this.homeHalo.rotation.z = Math.sin(now * 0.3) * 0.06;
        }

        // 跨巢光流推进
        for (let i = this.streams.length - 1; i >= 0; i--) {
            const s = this.streams[i];
            s.t += dt;
            const head = s.t / s.dur;
            const pos = s.pts.geometry.attributes.position;
            for (let k = 0; k < s.N; k++) {
                const p = head - k * 0.012;
                if (p < 0 || p > 1) { pos.setXYZ(k, 0, -50, 0); continue; }
                const e = smoothstep(p);
                pos.setXYZ(k, s.from.x + (s.to.x - s.from.x) * e,
                    0.5 + Math.sin(Math.PI * e) * 2.6 + Math.sin(k * 1.7) * 0.12,
                    s.from.z + (s.to.z - s.from.z) * e);
            }
            pos.needsUpdate = true;
            s.pts.material.opacity = head < 0.85 ? 0.9 : Math.max(0, 0.9 * (1 - (head - 0.85) / 0.35));
            if (head > 1.2) { this.scene.remove(s.pts); s.pts.geometry.dispose(); s.pts.material.dispose(); this.streams.splice(i, 1); }
        }

        // 浮尘缓慢游移
        if (this.motes) this.motes.position.set(Math.sin(now * 0.05) * 3, Math.sin(now * 0.07) * 1.2, Math.cos(now * 0.04) * 3);

        // 蜜蜂盘旋
        for (const b of this.bees) {
            if (!b.visible) continue;
            const u = b.userData;
            const a = now * u.speed + u.phase;
            b.position.set(this.focusSmooth.x + Math.cos(a) * u.r, u.h + Math.sin(now * 1.3 + u.phase) * 0.5, this.focusSmooth.z + Math.sin(a) * u.r);
        }

        this.renderer.render(this.scene, this.camera);
    }
}
