import { execSync } from "child_process";
import { expect, test } from "@playwright/test";

const API = process.env.E2E_API_URL ?? "http://localhost:18080";
// Command granting a role, e.g. "go run ./cmd/admin" with DATABASE_URL set.
const ADMIN_CMD = process.env.E2E_ADMIN_CMD;

async function call(path: string, body: unknown, token?: string) {
  const res = await fetch(`${API}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: JSON.stringify(body)
  });
  const text = await res.text();
  return { status: res.status, body: text ? JSON.parse(text) : null };
}

test("a moderator reviews a report and suspends the account", async ({ page }) => {
  test.skip(!ADMIN_CMD, "E2E_ADMIN_CMD not set");
  const id = `${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
  const moderatorEmail = `mod_${id}@e2e.test`;
  const mod = await call("/auth/register", { email: moderatorEmail, password: "Password123" });
  const reporter = await call("/auth/register", { email: `rep_${id}@e2e.test`, password: "Password123" });
  const target = await call("/auth/register", { email: `tgt_${id}@e2e.test`, password: "Password123" });
  expect(mod.status).toBe(201);
  await fetch(`${API}/profile`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${target.body.accessToken}` },
    body: JSON.stringify({ firstName: "Bruno", birthdate: "1990-01-01", gender: "man" })
  });
  const report = await call("/reports", { userId: target.body.user.id, reason: "spam", details: `Me demande de l'argent ${id}` }, reporter.body.accessToken);
  expect(report.status).toBe(201);
  execSync(`${ADMIN_CMD} role ${moderatorEmail} moderator`, { stdio: "ignore" });

  await page.goto("/");
  await page.getByTestId("welcome-signin").click();
  await page.getByTestId("signin-email").fill(moderatorEmail);
  await page.getByTestId("signin-password").fill("Password123");
  await page.getByTestId("signin-submit").click();

  await expect(page.getByTestId("moderation-screen")).toBeVisible();
  await expect(page.getByText(`« Me demande de l'argent ${id} »`)).toBeVisible();
  await page.getByTestId("moderation-suspend").last().click();
  await expect(page.getByText("Compte suspendu", { exact: false })).toBeVisible();

  const login = await call("/auth/login", { email: `tgt_${id}@e2e.test`, password: "Password123" });
  expect(login.status).toBe(403);
});
