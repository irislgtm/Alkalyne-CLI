#!/usr/bin/env node
const { createWriteStream, existsSync, chmodSync } = require("fs");
const { get } = require("https");
const { platform, arch } = process;
const path = require("path");

const pkg = require("./package.json");
const version = pkg.version;

const osMap = { darwin: "darwin", linux: "linux", win32: "windows" };
const archMap = { x64: "amd64", arm64: "arm64" };

const os = osMap[platform];
const cpu = archMap[arch];

if (!os || !cpu) {
  console.error(`alkalyne: unsupported platform ${platform}/${arch}`);
  process.exit(1);
}

const ext = os === "windows" ? ".exe" : "";
const binName = `alkalyne-${os}-${cpu}${ext}`;
const url = `https://github.com/irislgtm/Alkalyne-CLI/releases/download/v${version}/${binName}`;
const dest = path.join(__dirname, binName);

if (existsSync(dest)) {
  process.exit(0);
}

console.log(`alkalyne: downloading ${binName}...`);

const file = createWriteStream(dest);
get(url, (res) => {
  if (res.statusCode === 302 || res.statusCode === 301) {
    get(res.headers.location, (res2) => {
      res2.pipe(file);
      file.on("finish", () => {
        file.close();
        if (os !== "windows") chmodSync(dest, 0o755);
        console.log("alkalyne: installed");
      });
    });
    return;
  }
  if (res.statusCode !== 200) {
    console.error(`alkalyne: download failed (HTTP ${res.statusCode})`);
    console.error(`  ${url}`);
    process.exit(1);
  }
  res.pipe(file);
  file.on("finish", () => {
    file.close();
    if (os !== "windows") chmodSync(dest, 0o755);
    console.log("alkalyne: installed");
  });
}).on("error", (err) => {
  console.error(`alkalyne: download error: ${err.message}`);
  process.exit(1);
});
