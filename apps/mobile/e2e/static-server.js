// Minimal static server for the exported web build (single-page app fallback).
const http = require("http");
const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..", "dist");
const port = Number(process.env.WEB_PORT || 8081);
const types = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".json": "application/json",
  ".png": "image/png",
  ".css": "text/css",
  ".ico": "image/x-icon"
};

http
  .createServer((req, res) => {
    const urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
    let file = path.join(root, path.normalize(urlPath));
    if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory())
      file = path.join(root, "index.html");
    res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  })
  .listen(port, () => console.log(`web build served on :${port}`));
