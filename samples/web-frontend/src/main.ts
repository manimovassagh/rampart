import { RampartClient } from "@rampart/web";

const RAMPART_ISSUER = "http://localhost:8080";

const client = new RampartClient({
  issuer: RAMPART_ISSUER,
  onTokenChange(tokens) {
    if (tokens) {
      localStorage.setItem("rampart_tokens", JSON.stringify(tokens));
    } else {
      localStorage.removeItem("rampart_tokens");
    }
  },
});

// Restore tokens from localStorage on page load
const saved = localStorage.getItem("rampart_tokens");
if (saved) {
  try {
    client.setTokens(JSON.parse(saved));
  } catch {
    localStorage.removeItem("rampart_tokens");
  }
}

// DOM elements
const loginSection = document.getElementById("login-section")!;
const authSection = document.getElementById("auth-section")!;
const identifierInput = document.getElementById("identifier") as HTMLInputElement;
const passwordInput = document.getElementById("password") as HTMLInputElement;
const loginBtn = document.getElementById("login-btn")!;
const loginError = document.getElementById("login-error")!;
const userInfo = document.getElementById("user-info")!;
const profileBtn = document.getElementById("profile-btn")!;
const logoutBtn = document.getElementById("logout-btn")!;
const apiResult = document.getElementById("api-result")!;
const issuerUrl = document.getElementById("issuer-url")!;

issuerUrl.textContent = RAMPART_ISSUER;

function showLogin() {
  loginSection.classList.remove("hidden");
  authSection.classList.add("hidden");
  apiResult.classList.add("hidden");
  loginError.classList.add("hidden");
}

function showAuth(user: { email: string; preferred_username?: string; username?: string; id: string; org_id: string }) {
  loginSection.classList.add("hidden");
  authSection.classList.remove("hidden");
  userInfo.innerHTML = `
    <dl>
      <dt>User</dt><dd>${user.preferred_username ?? user.username ?? "—"}</dd>
      <dt>Email</dt><dd>${user.email}</dd>
      <dt>User ID</dt><dd>${user.id}</dd>
      <dt>Org ID</dt><dd>${user.org_id}</dd>
    </dl>
  `;
}

// Login
loginBtn.addEventListener("click", async () => {
  loginError.classList.add("hidden");
  try {
    const res = await client.login({
      identifier: identifierInput.value,
      password: passwordInput.value,
    });
    showAuth(res.user);
  } catch (err: unknown) {
    const rampartErr = err as { error_description?: string };
    loginError.textContent = rampartErr.error_description ?? "Login failed";
    loginError.classList.remove("hidden");
  }
});

// Fetch protected API (through Express backend proxy)
profileBtn.addEventListener("click", async () => {
  try {
    const res = await client.authFetch("/api/profile");
    const data = await res.json();
    apiResult.textContent = JSON.stringify(data, null, 2);
    apiResult.classList.remove("hidden");
  } catch (err) {
    apiResult.textContent = `Error: ${err}`;
    apiResult.classList.remove("hidden");
  }
});

// Logout
logoutBtn.addEventListener("click", async () => {
  await client.logout();
  showLogin();
});

// On page load, check if we have tokens and try to show profile
if (client.isAuthenticated()) {
  client
    .getUser()
    .then((user) => showAuth(user))
    .catch(() => {
      client.setTokens(null);
      showLogin();
    });
} else {
  showLogin();
}
