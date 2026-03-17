import { test, expect } from "@playwright/test";

// Game server must be running: LOCAL_DEV=true PORT=7777 go run ./game-server
const GAME_PORT = "7777";

test.describe("Raid Test Page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/raid-test");
  });

  test("renders page with all controls", async ({ page }) => {
    await expect(page.getByText("RAID BATTLE DEBUG")).toBeVisible();
    await expect(page.getByText("Host", { exact: true })).toBeVisible();
    await expect(page.getByText("Port", { exact: true })).toBeVisible();
    await expect(page.getByText("Cert Hash", { exact: false })).toBeVisible();
    await expect(page.getByText("WebTransport", { exact: true })).toBeVisible();
    await expect(page.getByText("WebSocket", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "CONNECT", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "DISCONNECT", exact: true })).toBeVisible();
  });

  test("CONNECT is enabled, DISCONNECT is disabled initially", async ({ page }) => {
    await expect(page.getByRole("button", { name: "CONNECT", exact: true })).toBeEnabled();
    await expect(page.getByRole("button", { name: "DISCONNECT", exact: true })).toBeDisabled();
  });

  test("TAP ATTACK and SPECIAL are disabled when disconnected", async ({ page }) => {
    await expect(page.getByRole("button", { name: "TAP ATTACK" })).toBeDisabled();
    await expect(page.getByRole("button", { name: "SPECIAL" })).toBeDisabled();
  });

  test("protocol radio switches between WT and WS", async ({ page }) => {
    const wtRadio = page.getByRole("radio", { name: "WebTransport" });
    const wsRadio = page.getByRole("radio", { name: "WebSocket" });

    await expect(wtRadio).toBeChecked();
    await expect(wsRadio).not.toBeChecked();

    await wsRadio.click();
    await expect(wsRadio).toBeChecked();
    await expect(wtRadio).not.toBeChecked();
  });

  test("user ID is a valid UUID", async ({ page }) => {
    const userIdText = await page.locator(".select-all").textContent();
    expect(userIdText).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/,
    );
  });

  test("tap count displays 0 initially", async ({ page }) => {
    await expect(page.getByText("Taps:")).toBeVisible();
    const tapsSection = page.locator("text=Taps:").locator("..");
    await expect(tapsSection).toContainText("0");
  });
});

test.describe("Raid Test Page - WebSocket E2E", () => {
  test("WS: connect → join → tap → verify HP updates", async ({ page }) => {
    await page.goto("/raid-test");

    // Switch to WebSocket
    await page.getByRole("radio", { name: "WebSocket" }).click();

    // Fill connection form
    await page.getByRole("textbox", { name: "Host" }).fill("localhost");
    const portInput = page.locator('input[placeholder="7003"]');
    await portInput.fill(GAME_PORT);

    // Connect
    await page.getByRole("button", { name: "CONNECT", exact: true }).click();

    // Wait for connected state
    await expect(page.getByText("connected")).toBeVisible({ timeout: 5000 });

    // JOIN button should appear
    const joinBtn = page.getByRole("button", { name: "JOIN" });
    await expect(joinBtn).toBeVisible();
    await joinBtn.click();

    // Wait for boss HP to appear
    await expect(page.locator("text=/\\d+ \\/ 50,000/")).toBeVisible({ timeout: 5000 });

    // TAP ATTACK should be enabled
    const tapBtn = page.getByRole("button", { name: "TAP ATTACK" });
    await expect(tapBtn).toBeEnabled();

    // Tap 3 times
    await tapBtn.click();
    await tapBtn.click();
    await tapBtn.click();

    // Verify tap count incremented
    const tapsSection = page.locator("text=Taps:").locator("..");
    await expect(tapsSection).toContainText("3");

    // Verify boss HP decreased (should be less than 50,000 now)
    await page.waitForTimeout(500);
    const hpText = await page.locator("text=/[\\d,]+ \\/ 50,000/").textContent();
    expect(hpText).toBeTruthy();
    const currentHp = Number.parseInt(hpText!.split("/")[0].replace(/[, ]/g, ""));
    expect(currentHp).toBeLessThan(50000);

    // Disconnect
    await page.getByRole("button", { name: "DISCONNECT", exact: true }).click();
    await expect(page.getByText("disconnected")).toBeVisible({ timeout: 3000 });
  });

  test("WS: tap count is independent per protocol", async ({ page }) => {
    await page.goto("/raid-test");

    // Switch to WebSocket, connect, join
    await page.getByRole("radio", { name: "WebSocket" }).click();
    await page.getByRole("textbox", { name: "Host" }).fill("localhost");
    await page.locator('input[placeholder="7003"]').fill(GAME_PORT);
    await page.getByRole("button", { name: "CONNECT", exact: true }).click();
    await expect(page.getByText("connected")).toBeVisible({ timeout: 5000 });
    await page.getByRole("button", { name: "JOIN" }).click();
    await expect(page.locator("text=/\\d+ \\/ 50,000/")).toBeVisible({ timeout: 5000 });

    // Tap 5 times via WS
    const tapBtn = page.getByRole("button", { name: "TAP ATTACK" });
    for (let i = 0; i < 5; i++) {
      await tapBtn.click();
    }

    // Verify WS tap count = 5
    const tapsSection = page.locator("text=Taps:").locator("..");
    await expect(tapsSection).toContainText("5");

    // Switch protocol selector to WT (without reconnecting)
    await page.getByRole("radio", { name: "WebTransport" }).click();

    // WT tap count should be 0 (independent)
    await expect(tapsSection).toContainText("0");

    // Switch back to WS
    await page.getByRole("radio", { name: "WebSocket" }).click();

    // WS tap count should still be 5
    await expect(tapsSection).toContainText("5");
  });

  test("WS: message log shows sent and received messages", async ({ page }) => {
    await page.goto("/raid-test");

    await page.getByRole("radio", { name: "WebSocket" }).click();
    await page.getByRole("textbox", { name: "Host" }).fill("localhost");
    await page.locator('input[placeholder="7003"]').fill(GAME_PORT);
    await page.getByRole("button", { name: "CONNECT", exact: true }).click();
    await expect(page.getByText("connected")).toBeVisible({ timeout: 5000 });

    // Join
    await page.getByRole("button", { name: "JOIN" }).click();
    await page.waitForTimeout(500);

    // Check message log has entries
    const logSection = page.locator(".max-h-80");
    await expect(logSection).not.toContainText("No messages yet");

    // Should contain join message (sent)
    await expect(logSection).toContainText('"t":"join"');
    // Should contain joined response (received)
    await expect(logSection).toContainText('"t":"joined"');
  });
});
