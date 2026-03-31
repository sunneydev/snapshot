const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const https = require("https");

const SNAPSHOT_REPO = "sunneydev/snapshot";
const RESTIC_REPO = "restic/restic";
const RESTIC_VERSION = "0.18.1";
const BIN_DIR = path.join(__dirname, "bin");

const PLATFORM_MAP = { linux: "linux", darwin: "darwin", win32: "windows" };
const ARCH_MAP = { x64: "amd64", arm64: "arm64" };

function getPlatform() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];
  if (!platform || !arch) {
    console.error(`unsupported platform: ${process.platform}/${process.arch}`);
    process.exit(1);
  }
  return { platform, arch };
}

function follow(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { "User-Agent": "snapvault" } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return follow(res.headers.location).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`download failed (${url}): ${res.statusCode}`));
      }
      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("end", () => resolve(Buffer.concat(chunks)));
      res.on("error", reject);
    }).on("error", reject);
  });
}

function downloadAndExtract(url, filename) {
  return follow(url).then((data) => {
    const tmp = path.join(__dirname, filename);
    fs.writeFileSync(tmp, data);

    if (filename.endsWith(".tar.gz")) {
      execSync(`tar xzf "${tmp}" -C "${BIN_DIR}"`, { stdio: "inherit" });
    } else if (filename.endsWith(".bz2")) {
      execSync(`bunzip2 -f "${tmp}"`, { stdio: "inherit" });
      const extracted = tmp.replace(/\.bz2$/, "");
      fs.renameSync(extracted, path.join(BIN_DIR, "restic"));
    } else if (filename.endsWith(".zip")) {
      execSync(`unzip -o "${tmp}" -d "${BIN_DIR}"`, { stdio: "inherit" });
    }

    if (fs.existsSync(tmp)) fs.unlinkSync(tmp);
  });
}

async function main() {
  const { platform, arch } = getPlatform();
  fs.mkdirSync(BIN_DIR, { recursive: true });

  const snapshotExt = platform === "linux" ? "tar.gz" : "zip";
  const snapshotFile = `snapshot_${platform}_${arch}.${snapshotExt}`;
  const snapshotUrl = `https://github.com/${SNAPSHOT_REPO}/releases/latest/download/${snapshotFile}`;

  const isWindows = platform === "windows";
  const resticExt = isWindows ? "zip" : "bz2";
  const resticFile = `restic_${RESTIC_VERSION}_${platform}_${arch}.${resticExt}`;
  const resticUrl = `https://github.com/${RESTIC_REPO}/releases/download/v${RESTIC_VERSION}/${resticFile}`;

  console.log("downloading snapshot...");
  console.log("downloading restic...");

  await Promise.all([
    downloadAndExtract(snapshotUrl, snapshotFile),
    downloadAndExtract(resticUrl, resticFile),
  ]);

  for (const name of ["snapshot", "restic"]) {
    const bin = path.join(BIN_DIR, isWindows ? `${name}.exe` : name);
    if (fs.existsSync(bin)) fs.chmodSync(bin, 0o755);
  }

  console.log("snapshot + restic installed successfully");
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
