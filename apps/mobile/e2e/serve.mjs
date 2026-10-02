// Minimal static server for the exported web build (SPA fallback to index.html).
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { extname, join, normalize } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../dist", import.meta.url));
const port = Number(process.env.PORT ?? 8081);
const types = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".json": "application/json",
  ".png": "image/png",
  ".ttf": "font/ttf",
  ".css": "text/css",
  ".ico": "image/x-icon",
};

createServer(async (req, res) => {
  const path = normalize(decodeURIComponent(new URL(req.url ?? "/", "http://x").pathname)).replace(
    /^(\.\.[/\\])+/,
    "",
  );
  try {
    const file = join(root, path);
    if (!file.startsWith(root)) throw new Error("outside root");
    const data = await readFile(file);
    res
      .writeHead(200, { "Content-Type": types[extname(file)] ?? "application/octet-stream" })
      .end(data);
  } catch {
    const html = await readFile(join(root, "index.html"));
    res.writeHead(200, { "Content-Type": "text/html" }).end(html);
  }
}).listen(port, () => console.log(`web build on http://localhost:${port}`));
