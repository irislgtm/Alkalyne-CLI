#!/usr/bin/env node
const { createWriteStream, existsSync, chmodSync, unlinkSync, renameSync } = require("fs");
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
const releaseName = `alkalyne-${os}-${cpu}${ext}`;
const canonicalName = `alkalyne${ext}`;
const url = `https://github.com/irislgtm/Alkalyne-CLI/releases/download/v${version}/${releaseName}`;
const dest = path.join(__dirname, canonicalName);
const tempDest = path.join(__dirname, `${releaseName}.download`);

if (existsSync(dest)) {
  process.exit(0);
}

console.log(`alkalyne: downloading ${releaseName}...`);

function cleanupTemp() {
  if (existsSync(tempDest)) {
    unlinkSync(tempDest);
  }
}

function finalizeInstall() {
  if (existsSync(dest)) {
    unlinkSync(dest);
  }
  renameSync(tempDest, dest);
  if (os !== "windows") chmodSync(dest, 0o755);
  console.log("alkalyne: installed");
}

function failInstall(message, error) {
  cleanupTemp();
  console.error(message);
  if (error) {
    console.error(`alkalyne: ${error.message}`);
  }
  process.exit(1);
}

function download(downloadURL, redirectsLeft = 5) {
  get(downloadURL, (res) => {
    if ((res.statusCode === 301 || res.statusCode === 302) && redirectsLeft > 0) {
      if (!res.headers.location) {
        failInstall("alkalyne: download failed (missing redirect location)");
      }
      download(res.headers.location, redirectsLeft - 1);
      return;
    }

    if (res.statusCode !== 200) {
      failInstall(`alkalyne: download failed (HTTP ${res.statusCode})\n  ${downloadURL}`);
    }

    const file = createWriteStream(tempDest);
    res.pipe(file);

    file.on("finish", () => {
      file.close(() => {
        try {
          finalizeInstall();
        } catch (err) {
          failInstall("alkalyne: failed to finalize install", err);
        }
      });
    });

    file.on("error", (err) => {
      failInstall("alkalyne: failed while writing binary", err);
    });
  }).on("error", (err) => {
    failInstall("alkalyne: download error", err);
  });
}

cleanupTemp();
download(url);
