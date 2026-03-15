import { RampartClient } from "@rampart-auth/web";

const RAMPART_ISSUER = "http://localhost:8080";
const CLIENT_ID = "sample-react-app"; // reuse the same OAuth client

const client = new RampartClient({
  issuer: RAMPART_ISSUER,
  clientId: CLIENT_ID,
  redirectUri: "http://localhost:3000/callback",
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

// --- DOM elements ---

const loginSection = document.getElementById("login-section")!;
const loginBtn = document.getElementById("login-btn")!;
const unauthSection = document.getElementById("unauth-section")!;
const unauthProfileBtn = document.getElementById("unauth-profile-btn")!;
const unauthClaimsBtn = document.getElementById("unauth-claims-btn")!;
const unauthResponse = document.getElementById("unauth-response")!;
const unauthEndpointLabel = document.getElementById("unauth-endpoint-label")!;
const unauthResult = document.getElementById("unauth-result")!;

const authSection = document.getElementById("auth-section")!;
const userInfo = document.getElementById("user-info")!;
const profileBtn = document.getElementById("profile-btn")!;
const claimsBtn = document.getElementById("claims-btn")!;
const meBtn = document.getElementById("me-btn")!;
const logoutBtn = document.getElementById("logout-btn")!;
const apiResponseSection = document.getElementById("api-response-section")!;
const endpointLabel = document.getElementById("endpoint-label")!;
const apiResult = document.getElementById("api-result")!;
const issuerUrl = document.getElementById("issuer-url")!;
const callbackStatus = document.getElementById("callback-status")!;

issuerUrl.textContent = RAMPART_ISSUER;

// --- View switching ---

function showLogin() {
  loginSection.classList.remove("hidden");
  unauthSection.classList.remove("hidden");
  authSection.classList.add("hidden");
  callbackStatus.classList.add("hidden");
}

function showAuth(user: {
  email: string;
  preferred_username?: string;
  username?: string;
  id: string;
  org_id: string;
  email_verified?: boolean;
  given_name?: string;
  family_name?: string;
}) {
  loginSection.classList.add("hidden");
  unauthSection.classList.add("hidden");
  authSection.classList.remove("hidden");
  apiResponseSection.classList.add("hidden");
  callbackStatus.classList.add("hidden");

  const name = [user.given_name, user.family_name].filter(Boolean).join(" ");

  // Build user info using safe DOM manipulation (no innerHTML to prevent XSS)
  userInfo.textContent = "";
  const dl = document.createElement("dl");

  const addRow = (label: string, value: string) => {
    const dt = document.createElement("dt");
    dt.textContent = label;
    const dd = document.createElement("dd");
    dd.textContent = value;
    dl.appendChild(dt);
    dl.appendChild(dd);
  };

  addRow("User", user.preferred_username ?? user.username ?? "\u2014");
  if (name) addRow("Name", name);

  // Email row with verified badge
  const emailDt = document.createElement("dt");
  emailDt.textContent = "Email";
  const emailDd = document.createElement("dd");
  emailDd.textContent = user.email + " ";
  const badge = document.createElement("span");
  badge.className = user.email_verified ? "badge badge-green" : "badge badge-yellow";
  badge.textContent = user.email_verified ? "verified" : "unverified";
  emailDd.appendChild(badge);
  dl.appendChild(emailDt);
  dl.appendChild(emailDd);

  addRow("User ID", user.id);
  addRow("Org ID", user.org_id);

  userInfo.appendChild(dl);
}

function showApiResponse(endpoint: string, data: unknown, status: number) {
  // Build endpoint label using safe DOM manipulation (no innerHTML to prevent XSS)
  endpointLabel.textContent = "";
  const code = document.createElement("code");
  code.textContent = endpoint;
  const statusSpan = document.createElement("span");
  statusSpan.className = status >= 200 && status < 300 ? "status-success" : "status-error";
  statusSpan.textContent = String(status);
  endpointLabel.appendChild(code);
  endpointLabel.appendChild(document.createTextNode(" \u2014 "));
  endpointLabel.appendChild(statusSpan);
  apiResult.textContent = JSON.stringify(data, null, 2);
  apiResponseSection.classList.remove("hidden");
}

// --- Unauthenticated endpoint tests ---

function showUnauthResponse(endpoint: string, data: unknown, status: number) {
  // Build endpoint label using safe DOM manipulation (no innerHTML to prevent XSS)
  unauthEndpointLabel.textContent = "";
  const unauthCode = document.createElement("code");
  unauthCode.textContent = endpoint;
  const unauthStatus = document.createElement("span");
  unauthStatus.className = "status-error";
  unauthStatus.textContent = `${status} Unauthorized`;
  unauthEndpointLabel.appendChild(unauthCode);
  unauthEndpointLabel.appendChild(document.createTextNode(" \u2014 "));
  unauthEndpointLabel.appendChild(unauthStatus);
  unauthResult.textContent = JSON.stringify(data, null, 2);
  unauthResponse.classList.remove("hidden");
}

unauthProfileBtn.addEventListener("click", async () => {
  const res = await fetch("/api/profile");
  const data = await res.json();
  showUnauthResponse("GET /api/profile (no token)", data, res.status);
});

unauthClaimsBtn.addEventListener("click", async () => {
  const res = await fetch("/api/claims");
  const data = await res.json();
  showUnauthResponse("GET /api/claims (no token)", data, res.status);
});

// --- Login (OAuth PKCE redirect) ---

loginBtn.addEventListener("click", async () => {
  await client.loginWithRedirect();
});

// --- Protected endpoints ---

profileBtn.addEventListener("click", async () => {
  const res = await client.authFetch("/api/profile");
  const data = await res.json();
  showApiResponse("GET /api/profile", data, res.status);
});

claimsBtn.addEventListener("click", async () => {
  const res = await client.authFetch("/api/claims");
  const data = await res.json();
  showApiResponse("GET /api/claims", data, res.status);
});

meBtn.addEventListener("click", async () => {
  const res = await client.authFetch(`${RAMPART_ISSUER}/me`);
  const data = await res.json();
  showApiResponse("GET /me (Rampart)", data, res.status);
});

// --- Logout ---

logoutBtn.addEventListener("click", async () => {
  await client.logout();
  showLogin();
});

// --- Init: handle callback or restore session ---

async function init() {
  const url = new URL(window.location.href);

  // Check if this is an OAuth callback
  if (url.pathname === "/callback" && url.searchParams.has("code")) {
    callbackStatus.textContent = "Exchanging authorization code...";
    callbackStatus.classList.remove("hidden");
    loginSection.classList.add("hidden");
    unauthSection.classList.add("hidden");

    try {
      await client.handleCallback();
      // Redirect to home after successful token exchange
      window.history.replaceState({}, "", "/");
      const user = await client.getUser();
      showAuth(user);
    } catch (err) {
      callbackStatus.textContent = `Login failed: ${(err as { error_description?: string }).error_description ?? "unknown error"}`;
      callbackStatus.style.color = "#f87171";
    }
    return;
  }

  // Try to restore existing session
  if (client.isAuthenticated()) {
    try {
      const user = await client.getUser();
      showAuth(user);
    } catch {
      client.setTokens(null);
      showLogin();
    }
  } else {
    showLogin();
  }
}

init();
