// Tiny static server with SPA fallback for the exported web build (no dependency).
const http = require("http");
const fs = require("fs");
const path = require("path");

const types = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".png": "image/png",
  ".ico": "image/x-icon",
  ".ttf": "font/ttf"
};

function serve(root, port) {
  const server = http.createServer((req, res) => {
    const urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
    let file = path.join(root, urlPath);
    if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
      file = path.join(root, "index.html");
    }
    res.writeHead(200, { "Content-Type": types[path.extname(file)] || "application/octet-stream" });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(port, () => resolve(server)));
}

module.exports = { serve };

if (require.main === module) {
  const root = path.resolve(process.argv[2] || path.join(__dirname, "..", "dist"));
  const port = Number(process.argv[3] || 8081);
  serve(root, port).then(() => console.log(`serving ${root} on http://localhost:${port}`));
}
