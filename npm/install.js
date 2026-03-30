const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const https = require("https");

const REPO = "sunneydev/snapshot";
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
    https.get(url, { headers: { "User-Agent": "snapshot-backup" } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return follow(res.headers.location).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`download failed: ${res.statusCode}`));
      }
      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("end", () => resolve(Buffer.concat(chunks)));
      res.on("error", reject);
    }).on("error", reject);
  });
}

async function main() {
  const { platform, arch } = getPlatform();
  const ext = platform === "linux" ? "tar.gz" : "zip";
  const name = `snapshot_${platform}_${arch}.${ext}`;
  const url = `https://github.com/${REPO}/releases/latest/download/${name}`;

  console.log(`downloading ${name}...`);

  const data = await follow(url);
  const tmp = path.join(__dirname, name);
  fs.writeFileSync(tmp, data);
  fs.mkdirSync(BIN_DIR, { recursive: true });

  if (ext === "tar.gz") {
    execSync(`tar xzf "${tmp}" -C "${BIN_DIR}"`, { stdio: "inherit" });
  } else {
    execSync(`unzip -o "${tmp}" -d "${BIN_DIR}"`, { stdio: "inherit" });
  }

  fs.unlinkSync(tmp);

  const binary = path.join(BIN_DIR, platform === "windows" ? "snapshot.exe" : "snapshot");
  if (fs.existsSync(binary)) {
    fs.chmodSync(binary, 0o755);
  }

  console.log("snapshot installed successfully");
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
