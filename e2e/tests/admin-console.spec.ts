import { test, expect } from "@playwright/test";

const adminUser = {
  username: `admin${Date.now()}`,
  email: `admin${Date.now()}@e2e.test`,
  password: "AdminP@ss123",
  given_name: "Admin",
  family_name: "E2E",
};

test.describe.serial("Admin console browser flow", () => {
  test.beforeAll(async ({ request }) => {
    await request.post("/register", { data: adminUser });
  });

  test("login page renders correctly", async ({ page }) => {
    await page.goto("/admin/login");
    await expect(page.getByRole("heading", { name: "Rampart" })).toBeVisible();
    await expect(
      page.getByRole("textbox", { name: "Username or Email" })
    ).toBeVisible();
    await expect(
      page.getByRole("textbox", { name: "Password" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Sign In" })
    ).toBeVisible();
  });

  test("login navigates to dashboard", async ({ page }) => {
    await page.goto("/admin/login");
    await page
      .getByRole("textbox", { name: "Username or Email" })
      .fill(adminUser.email);
    await page
      .getByRole("textbox", { name: "Password" })
      .fill(adminUser.password);
    await page.getByRole("button", { name: "Sign In" }).click();

    await expect(page).toHaveURL(/\/admin\/$/);
    await expect(page).toHaveTitle(/Dashboard/);
    await expect(
      page.getByRole("heading", { name: "Dashboard" })
    ).toBeVisible();
    await expect(page.getByText(adminUser.username)).toBeVisible();
  });

  test("dashboard shows stat cards", async ({ page }) => {
    await page.goto("/admin/login");
    await page
      .getByRole("textbox", { name: "Username or Email" })
      .fill(adminUser.email);
    await page
      .getByRole("textbox", { name: "Password" })
      .fill(adminUser.password);
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page).toHaveURL(/\/admin\/$/);

    const main = page.getByRole("main");
    await expect(main.getByText("Total Users")).toBeVisible();
    await expect(main.getByText("Active Sessions")).toBeVisible();
    await expect(main.getByText("Organizations")).toBeVisible();
    await expect(main.getByText("Roles")).toBeVisible();
  });

  test("navigate to Users page", async ({ page }) => {
    await page.goto("/admin/login");
    await page
      .getByRole("textbox", { name: "Username or Email" })
      .fill(adminUser.email);
    await page
      .getByRole("textbox", { name: "Password" })
      .fill(adminUser.password);
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page).toHaveURL(/\/admin\/$/);

    await page.getByRole("link", { name: "Users" }).click();
    await expect(page).toHaveURL(/\/admin\/users$/);
    await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();
    await expect(
      page.getByRole("columnheader", { name: "User" })
    ).toBeVisible();
    await expect(
      page.getByRole("columnheader", { name: "Email" })
    ).toBeVisible();
    await expect(page.getByText(adminUser.email)).toBeVisible();
  });

  test("navigate to Roles page", async ({ page }) => {
    await page.goto("/admin/login");
    await page
      .getByRole("textbox", { name: "Username or Email" })
      .fill(adminUser.email);
    await page
      .getByRole("textbox", { name: "Password" })
      .fill(adminUser.password);
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page).toHaveURL(/\/admin\/$/);

    await page.getByRole("link", { name: "Roles" }).click();
    await expect(page).toHaveURL(/\/admin\/roles$/);
    await expect(page.getByRole("heading", { name: "Roles" })).toBeVisible();
  });

  test("navigate to Organizations page", async ({ page }) => {
    await page.goto("/admin/login");
    await page
      .getByRole("textbox", { name: "Username or Email" })
      .fill(adminUser.email);
    await page
      .getByRole("textbox", { name: "Password" })
      .fill(adminUser.password);
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page).toHaveURL(/\/admin\/$/);

    await page.getByRole("link", { name: "Organizations" }).click();
    await expect(page).toHaveURL(/\/admin\/organizations$/);
    await expect(
      page.getByRole("heading", { name: "Organizations" })
    ).toBeVisible();
  });

  test("navigate to OIDC Config page", async ({ page }) => {
    await page.goto("/admin/login");
    await page
      .getByRole("textbox", { name: "Username or Email" })
      .fill(adminUser.email);
    await page
      .getByRole("textbox", { name: "Password" })
      .fill(adminUser.password);
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page).toHaveURL(/\/admin\/$/);

    await page.getByRole("link", { name: "OIDC Config" }).click();
    await expect(page).toHaveURL(/\/admin\/oidc$/);
    await expect(
      page.getByRole("heading", { name: "OIDC Configuration" })
    ).toBeVisible();
    await expect(page.getByText("Issuer")).toBeVisible();
    await expect(page.getByText("Authorization Endpoint")).toBeVisible();
    await expect(page.getByText("Token Endpoint")).toBeVisible();
    await expect(page.getByText("JWKS URI")).toBeVisible();
  });

  test("sidebar navigation links are all present", async ({ page }) => {
    await page.goto("/admin/login");
    await page
      .getByRole("textbox", { name: "Username or Email" })
      .fill(adminUser.email);
    await page
      .getByRole("textbox", { name: "Password" })
      .fill(adminUser.password);
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page).toHaveURL(/\/admin\/$/);

    const expectedLinks = [
      "Dashboard",
      "Users",
      "Roles",
      "Groups",
      "Organizations",
      "Sessions",
      "Events",
      "Clients",
      "OIDC Config",
    ];

    for (const name of expectedLinks) {
      await expect(page.getByRole("link", { name })).toBeVisible();
    }
  });
});
