import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  retries: 0,
  use: {
    baseURL: process.env.RAMPART_URL || "http://localhost:8080",
    headless: true,
    screenshot: "only-on-failure",
  },
  reporter: [["html", { open: "never" }], ["list"]],
  projects: [
    {
      name: "chromium",
      use: { browserName: "chromium" },
    },
  ],
});
