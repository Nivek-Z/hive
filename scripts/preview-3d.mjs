import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { extname, join, normalize } from "node:path";
const root = process.argv[2];
const types = { ".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8", ".woff2": "font/woff2", ".svg": "image/svg+xml" };
createServer(async (req, res) => {
  try {
    let p = decodeURIComponent(new URL(req.url, "http://x").pathname);
    if (p.endsWith("/")) p += "index.html";
    const data = await readFile(normalize(join(root, p)));
    res.writeHead(200, { "Content-Type": types[extname(p)] ?? "application/octet-stream" });
    res.end(data);
  } catch { res.writeHead(404); res.end("not found"); }
}).listen(8077, () => console.log("static up :8077"));
