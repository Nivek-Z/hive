// M5 角色权限冒烟测试（Node 22+，零依赖）
// 核心验证：无权限403 → 创建角色 → 分配 → 权限即时生效 → 撤销 → 再次403 → 默认角色保护
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
async function login(username) {
    const r = await rest(null, "POST", "/auth/login", { username, password: "123456" });
    if (r.code !== 0) fail("登录", JSON.stringify(r));
    return { token: r.data.token, me: r.data.user };
}

const a = await login("afeng");    // 巢主
const x = await login("xiaomi");   // 普通成员
const HIVE = 1;                    // 演示蜂巢

// 0. 前置：xiaomi 无管理频道权限
const deny = await rest(x.token, "POST", `/hives/${HIVE}/channels`, { name: "测试频道", type: "TEXT" });
if (deny.code !== 403) fail("无权限应403", JSON.stringify(deny));
ok("普通成员建频道被拒（403）");

// 1. 巢主创建带「管理频道」权限的角色
const created = await rest(a.token, "POST", `/hives/${HIVE}/roles`,
    { name: "巡逻蜂", color: "#26A69A", permissions: 4 | 512 | 1024 | 2048 });
if (created.code !== 0) fail("创建角色", JSON.stringify(created));
const roleId = created.data.id;
ok("创建角色「巡逻蜂」(含管理频道权限)");

// 2. 普通成员不能管理角色
const denyRole = await rest(x.token, "POST", `/hives/${HIVE}/roles`, { name: "黑角色", permissions: 1 });
if (denyRole.code !== 403) fail("成员建角色应403", JSON.stringify(denyRole));
ok("普通成员管理角色被拒（403）");

// 3. 分配角色 → 权限即时生效
await rest(a.token, "PUT", `/hives/${HIVE}/members/${x.me.id}/roles`, { roleIds: [roleId] });
const allow = await rest(x.token, "POST", `/hives/${HIVE}/channels`, { name: "巡逻报告", type: "TEXT" });
if (allow.code !== 0) fail("分配后应可建频道", JSON.stringify(allow));
ok("分配角色后权限即时生效（可建频道）");

// 4. 详情接口包含角色列表与我的权限
const detail = await rest(x.token, "GET", `/hives/${HIVE}`);
if (!detail.data.roles.some((r) => r.name === "巡逻蜂")) fail("详情含角色", JSON.stringify(detail.data.roles));
if ((detail.data.myPermissions & 4) !== 4) fail("myPermissions 含管理频道位", String(detail.data.myPermissions));
ok("蜂巢详情返回角色列表 + 我的生效权限位");

// 5. 撤销角色 → 权限随之失效
await rest(a.token, "PUT", `/hives/${HIVE}/members/${x.me.id}/roles`, { roleIds: [] });
const deny2 = await rest(x.token, "POST", `/hives/${HIVE}/channels`, { name: "再试一次", type: "TEXT" });
if (deny2.code !== 403) fail("撤销后应403", JSON.stringify(deny2));
ok("撤销角色后权限即时失效");

// 6. 默认角色不能删除
const roles = await rest(a.token, "GET", `/hives/${HIVE}/roles`);
const def = roles.data.find((r) => r.isDefault);
const delDef = await rest(a.token, "DELETE", `/roles/${def.id}`);
if (delDef.code === 0) fail("默认角色保护", "默认角色竟被删除");
ok("默认角色（工蜂）不可删除");

// 7. 清理：删角色 + 删测试频道
await rest(a.token, "DELETE", `/roles/${roleId}`);
await rest(a.token, "DELETE", `/channels/${allow.data.id}`);
const rolesAfter = await rest(a.token, "GET", `/hives/${HIVE}/roles`);
if (rolesAfter.data.some((r) => r.id === roleId)) fail("删除角色", "角色仍存在");
ok("删除角色与测试频道（清理）");

console.log(`\nALL PASS (${passed}) - M5 角色权限冒烟测试全部通过`);
