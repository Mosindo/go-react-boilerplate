// Minimal static server for the exported web build (SPA fallback to index.html).
const http = require("http");
const fs = require("fs");
const path = require("path");

const types = {
  ".html": "text/html; charset=utf-8",
  ".js": "application/javascript",
  ".json": "application/json",
  ".ttf": "font/ttf",
  ".png": "image/png",
  ".ico": "image/x-icon",
  ".css": "text/css"
};

function startStaticServer(root, port) {
  const base = path.resolve(root);
  const server = http.createServer((req, res) => {
    const urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
    let file = path.resolve(base, "." + urlPath);
    if (!file.startsWith(base) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
      file = path.join(base, "index.html");
    }
    res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(port, () => resolve(server)));
}

module.exports = { startStaticServer };

if (require.main === module) {
  const port = Number(process.env.PORT || 8081);
  startStaticServer(process.argv[2] || "dist", port).then(() => console.log(`serving on http://localhost:${port}`));
}
