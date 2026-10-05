// Minimal static server for the exported web build (single-page app fallback). Used by the E2E suite.
const fs = require("fs");
const http = require("http");
const path = require("path");

const root = path.resolve(process.argv[2] || "dist");
const port = Number(process.env.PORT || 8081);
const types = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".json": "application/json",
  ".css": "text/css",
  ".png": "image/png",
  ".ttf": "font/ttf",
  ".ico": "image/x-icon",
  ".svg": "image/svg+xml"
};

http
  .createServer((req, res) => {
    const urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
    let file = path.join(root, path.normalize(urlPath));
    if (!file.startsWith(root)) {
      res.writeHead(403).end();
      return;
    }
    if (!fs.existsSync(file) || fs.statSync(file).isDirectory()) file = path.join(root, "index.html");
    res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  })
  .listen(port, () => console.log(`[serve-web] ${root} on http://localhost:${port}`));
