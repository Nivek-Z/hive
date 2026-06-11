// REST 封装：统一携带 JWT、解包 ApiResponse、401 全局登出
const TOKEN_KEY = "hive_token";

export const getToken = () => localStorage.getItem(TOKEN_KEY);
export const setToken = (t) => t ? localStorage.setItem(TOKEN_KEY, t) : localStorage.removeItem(TOKEN_KEY);

let onUnauthorized = () => {};
export const setUnauthorizedHandler = (fn) => { onUnauthorized = fn; };

async function request(method, path, body) {
    const headers = {};
    const token = getToken();
    if (token) headers.Authorization = `Bearer ${token}`;

    let payload;
    if (body instanceof FormData) {
        payload = body; // multipart 由浏览器自动设置 Content-Type
    } else if (body !== undefined) {
        headers["Content-Type"] = "application/json";
        payload = JSON.stringify(body);
    }

    const res = await fetch(`/api${path}`, { method, headers, body: payload });
    if (res.status === 401) {
        onUnauthorized();
        throw new Error("登录已过期，请重新登录");
    }
    const json = await res.json();
    if (json.code !== 0) throw new Error(json.msg || "请求失败");
    return json.data;
}

export const api = {
    get: (p) => request("GET", p),
    post: (p, b) => request("POST", p, b ?? {}),
    put: (p, b) => request("PUT", p, b),
    del: (p) => request("DELETE", p),
    upload: (file) => {
        const fd = new FormData();
        fd.append("file", file);
        return request("POST", "/files", fd);
    },
};
