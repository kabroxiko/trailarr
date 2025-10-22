/* eslint-env node */
import fs from "fs";
import path from "path";
import sharp from "sharp";
// no external ico packer - we'll build ICO manually

// Convert web/public/logo.svg -> web/dist/favicon.ico (do NOT write to public)
const publicDir = path.resolve(
  globalThis.process && globalThis.process.cwd ? globalThis.process.cwd() : ".",
  "public",
);
const svgPath = path.join(publicDir, "logo.svg");
// write to dist/favicon.ico (Vite default output dir)
const distDir = path.resolve(
  globalThis.process && globalThis.process.cwd ? globalThis.process.cwd() : ".",
  "dist",
);
const outPath = path.join(distDir, "favicon.ico");

async function run() {
  if (!fs.existsSync(svgPath)) {
    console.error("logo.svg not found at", svgPath);
    if (globalThis.process && typeof globalThis.process.exit === "function")
      globalThis.process.exit(1);
  }

  // ensure dist directory exists; if not, fail with helpful message
  if (!fs.existsSync(distDir)) {
    console.error(
      "dist directory not found — run `vite build` before running this script",
    );
    if (globalThis.process && typeof globalThis.process.exit === "function")
      globalThis.process.exit(2);
  }

  try {
    const svg = fs.readFileSync(svgPath);
    // create PNGs at common favicon sizes and get buffers
    const sizes = [16, 32, 48, 64, 128, 256];
    const pngBuffers = [];
    for (const size of sizes) {
      const buf = await sharp(svg)
        .resize(size, size, { fit: "contain" })
        .png()
        .toBuffer();
      pngBuffers.push({ size, buf });
      // Also write PNG for each size
      const pngOutPath = path.join(distDir, `favicon-${size}x${size}.png`);
      fs.writeFileSync(pngOutPath, buf);
    }

    // Build ICO: header + dir entries + image data
    const count = pngBuffers.length;
    // header: 6 bytes
    const header = globalThis.Buffer.alloc(6);
    header.writeUInt16LE(0, 0); // reserved
    header.writeUInt16LE(1, 2); // type 1 = ICO
    header.writeUInt16LE(count, 4);

    const dirEntries = globalThis.Buffer.alloc(16 * count);
    let offset = 6 + dirEntries.length;
    const chunks = [header, dirEntries];

    for (let i = 0; i < count; i++) {
      const { size, buf } = pngBuffers[i];
      const entryOffset = 16 * i;
      // width/height: 1 byte each, 0 means 256
      dirEntries.writeUInt8(size === 256 ? 0 : size, entryOffset);
      dirEntries.writeUInt8(size === 256 ? 0 : size, entryOffset + 1);
      dirEntries.writeUInt8(0, entryOffset + 2); // color palette
      dirEntries.writeUInt8(0, entryOffset + 3); // reserved
      dirEntries.writeUInt16LE(1, entryOffset + 4); // color planes
      dirEntries.writeUInt16LE(32, entryOffset + 6); // bits per pixel
      dirEntries.writeUInt32LE(buf.length, entryOffset + 8); // size of image data
      dirEntries.writeUInt32LE(offset, entryOffset + 12); // offset of image data

      offset += buf.length;
      chunks.push(buf);
    }

    const icoBuf = globalThis.Buffer.concat(chunks);
    fs.writeFileSync(outPath, icoBuf);
    console.log("Wrote", outPath);
  } catch (err) {
    console.error("Failed to create favicon:", err);
    if (globalThis.process && typeof globalThis.process.exit === "function")
      globalThis.process.exit(2);
  }
}

run();
