const fs = require("fs");
const path = require("path");
const puppeteer = require("B:/workgit/node_modules/puppeteer");
const { execSync } = require("child_process");

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function clickByText(page, selector, text) {
  const elements = await page.$$(selector);
  for (const el of elements) {
    const content = await page.evaluate((node) => node.textContent, el);
    if (content && content.includes(text)) {
      await el.click();
      return true;
    }
  }
  return false;
}

async function run() {
  const outputDir = path.resolve(__dirname, "../docs/images");
  const framesDir = path.resolve(__dirname, "../frames");
  if (!fs.existsSync(framesDir)) fs.mkdirSync(framesDir, { recursive: true });

  const browser = await puppeteer.launch({
    headless: "new",
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  });

  const page = await browser.newPage();

  // 1. Desktop Dashboard Screenshot (1440x900)
  console.log("Capturing Dashboard...");
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });
  await page.goto("http://localhost:8085/studio", { waitUntil: "domcontentloaded" });
  await sleep(1500);
  await page.screenshot({ path: path.join(outputDir, "notiflux_dashboard.png") });
  console.log("✓ notiflux_dashboard.png captured");

  // 2. Workflows & Code Snippets Modal Screenshot
  console.log("Capturing Workflows & Code Snippets...");
  await clickByText(page, "button", "Workflows");
  await sleep(800);
  await clickByText(page, "button", "Code Snippets");
  await sleep(600);
  await page.screenshot({ path: path.join(outputDir, "notiflux_workflows.png") });
  console.log("✓ notiflux_workflows.png captured");

  // Close modal
  await clickByText(page, "button", "Close");
  await sleep(400);

  // 3. Delivery Logs Screenshot with Engagement Tracking
  console.log("Capturing Delivery Logs...");
  await clickByText(page, "button", "Delivery Logs");
  await sleep(800);
  // Expand first row
  const firstRow = await page.$("tbody tr");
  if (firstRow) await firstRow.click();
  await sleep(500);
  await page.screenshot({ path: path.join(outputDir, "notiflux_logs.png") });
  console.log("✓ notiflux_logs.png captured");

  // 4. Mobile Responsiveness Screenshot (390x844 iPhone size)
  console.log("Capturing Mobile Responsive View...");
  await page.setViewport({ width: 390, height: 844, isMobile: true, deviceScaleFactor: 2 });
  await page.goto("http://localhost:8085/studio", { waitUntil: "domcontentloaded" });
  await sleep(1000);

  // Open mobile drawer
  const menuBtn = await page.$("button[aria-label='Open navigation menu']");
  if (menuBtn) await menuBtn.click();
  await sleep(600);
  await page.screenshot({ path: path.join(outputDir, "notiflux_mobile.png") });
  console.log("✓ notiflux_mobile.png captured");

  // 5. Interactive Demo GIF frames (1200x750)
  console.log("Recording Demo Sequence for Animated GIF...");
  await page.setViewport({ width: 1200, height: 750, deviceScaleFactor: 1 });
  await page.goto("http://localhost:8085/studio", { waitUntil: "domcontentloaded" });
  await sleep(1200);

  let frameIdx = 0;
  const captureFrame = async (count = 1) => {
    for (let i = 0; i < count; i++) {
      const p = path.join(framesDir, `frame_${String(frameIdx).padStart(3, "0")}.png`);
      await page.screenshot({ path: p });
      frameIdx++;
      await sleep(80);
    }
  };

  // Hold initial view
  await captureFrame(8);

  // Type in subject
  const titleInput = await page.$("input[value='Security Alert']");
  if (titleInput) {
    await titleInput.click({ clickCount: 3 });
    await titleInput.type("Order #9921 Dispatched via Express");
    await captureFrame(6);
  }

  // Type in body
  const bodyInput = await page.$("textarea");
  if (bodyInput) {
    await bodyInput.click({ clickCount: 3 });
    await bodyInput.type("Package in transit. Guaranteed delivery tomorrow morning.");
    await captureFrame(6);
  }

  // Fire Trigger
  await clickByText(page, "button", "Fire Trigger");
  await sleep(400);
  await captureFrame(10);

  // Open the live notification bell in header
  await page.evaluate(() => {
    const bell = document.querySelector("notiflux-bell");
    if (bell && bell.shadowRoot) {
      const btn = bell.shadowRoot.querySelector("#nf-bell-btn");
      if (btn) btn.click();
    }
  });
  await sleep(500);
  await captureFrame(15);

  // Mark all as read
  await page.evaluate(() => {
    const bell = document.querySelector("notiflux-bell");
    if (bell && bell.shadowRoot) {
      const markAll = bell.shadowRoot.querySelector("#nf-mark-all");
      if (markAll) markAll.click();
    }
  });
  await sleep(500);
  await captureFrame(10);

  await browser.close();
  console.log(`✓ Captured ${frameIdx} frames. Assembling GIF with ffmpeg...`);

  const gifPath = path.join(outputDir, "notiflux_demo.gif");
  const ffmpegCmd = `ffmpeg -y -framerate 10 -i "${path.join(framesDir, "frame_%03d.png")}" -vf "fps=10,scale=960:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer" "${gifPath}"`;
  execSync(ffmpegCmd, { stdio: "inherit" });

  // Clean frames
  fs.rmSync(framesDir, { recursive: true, force: true });
  console.log("✓ docs/images/notiflux_demo.gif generated successfully");
}

run().catch((err) => {
  console.error("Capture script error:", err);
  process.exit(1);
});
