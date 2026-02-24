#!/usr/bin/env node
"use strict";

const { execSync } = require("child_process");
const fs = require("fs");
const https = require("https");
const path = require("path");
const os = require("os");

const VERSION = require("./package.json").version;
const REPO = "ProgenyAlpha/reddit-lurker";
const BIN_DIR = path.join(__dirname, "bin");
const BIN_PATH = path.join(BIN_DIR, process.platform === "win32" ? "lurk.exe" : "lurk");

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  const osMap = { linux: "linux", darwin: "darwin" };
  const archMap = { x64: "amd64", arm64: "arm64" };

  const goos = osMap[platform];
  const goarch = archMap[arch];

  if (!goos || !goarch) {
    console.error(`Unsupported platform: ${platform}/${arch}`);
    process.exit(1);
  }

  return { goos, goarch };
}

function download(url) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        return download(res.headers.location).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
      }
      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("end", () => resolve(Buffer.concat(chunks)));
      res.on("error", reject);
    }).on("error", reject);
  });
}

async function extractTarGz(buffer, destDir) {
  // Write to temp file and extract with tar
  const tmp = path.join(os.tmpdir(), `lurk-${Date.now()}.tar.gz`);
  fs.writeFileSync(tmp, buffer);
  fs.mkdirSync(destDir, { recursive: true });
  execSync(`tar xzf "${tmp}" -C "${destDir}"`, { stdio: "pipe" });
  fs.unlinkSync(tmp);
}

async function main() {
  const { goos, goarch } = getPlatform();
  const archive = `lurk-${goos}-${goarch}.tar.gz`;
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${archive}`;

  console.log(`Downloading lurk v${VERSION} for ${goos}/${goarch}...`);

  try {
    const data = await download(url);
    await extractTarGz(data, BIN_DIR);

    // Make binary executable
    fs.chmodSync(BIN_PATH, 0o755);
    console.log(`Installed lurk to ${BIN_PATH}`);
  } catch (err) {
    console.error(`Failed to download lurk: ${err.message}`);
    console.error(`URL: ${url}`);
    console.error(`\nYou can build from source instead:`);
    console.error(`  git clone https://github.com/${REPO}.git`);
    console.error(`  cd reddit-lurker && go build -o lurk .`);
    process.exit(1);
  }
}

main();
