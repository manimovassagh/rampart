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

// --- DOM elements ---

// Unauthenticated test
const unauthSection = document.getElementById("unauth-section")!;
const unauthProfileBtn = document.getElementById("unauth-profile-btn")!;
const unauthClaimsBtn = document.getElementById("unauth-claims-btn")!;
const unauthResponse = document.getElementById("unauth-response")!;
const unauthEndpointLabel = document.getElementById("unauth-endpoint-label")!;
const unauthResult = document.getElementById("unauth-result")!;

// Login
const loginSection = document.getElementById("login-section")!;
const identifierInput = document.getElementById("identifier") as HTMLInputElement;
const passwordInput = document.getElementById("password") as HTMLInputElement;
const loginBtn = document.getElementById("login-btn")!;
const loginError = document.getElementById("login-error")!;
const showSignupBtn = document.getElementById("show-signup-btn")!;

// Signup
const signupSection = document.getElementById("signup-section")!;
const regUsername = document.getElementById("reg-username") as HTMLInputElement;
const regEmail = document.getElementById("reg-email") as HTMLInputElement;
const regPassword = document.getElementById("reg-password") as HTMLInputElement;
const regGivenName = document.getElementById("reg-given-name") as HTMLInputElement;
const regFamilyName = document.getElementById("reg-family-name") as HTMLInputElement;
const signupBtn = document.getElementById("signup-btn")!;
const showLoginBtn = document.getElementById("show-login-btn")!;
const signupError = document.getElementById("signup-error")!;
const signupFieldErrors = document.getElementById("signup-field-errors")!;
const signupSuccess = document.getElementById("signup-success")!;

// Authenticated
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

issuerUrl.textContent = RAMPART_ISSUER;

// --- View switching ---

function showLogin() {
  loginSection.classList.remove("hidden");
  unauthSection.classList.remove("hidden");
  signupSection.classList.add("hidden");
  authSection.classList.add("hidden");
  loginError.classList.add("hidden");
}

function showSignup() {
  signupSection.classList.remove("hidden");
  unauthSection.classList.remove("hidden");
  loginSection.classList.add("hidden");
  authSection.classList.add("hidden");
  signupError.classList.add("hidden");
  signupFieldErrors.classList.add("hidden");
  signupSuccess.classList.add("hidden");
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
  signupSection.classList.add("hidden");
  unauthSection.classList.add("hidden");
  authSection.classList.remove("hidden");
  apiResponseSection.classList.add("hidden");

  const verified = user.email_verified
    ? '<span class="badge badge-green">verified</span>'
    : '<span class="badge badge-yellow">unverified</span>';

  const name = [user.given_name, user.family_name].filter(Boolean).join(" ");

  userInfo.innerHTML = `
    <dl>
      <dt>User</dt><dd>${user.preferred_username ?? user.username ?? "—"}</dd>
      ${name ? `<dt>Name</dt><dd>${name}</dd>` : ""}
      <dt>Email</dt><dd>${user.email} ${verified}</dd>
      <dt>User ID</dt><dd>${user.id}</dd>
      <dt>Org ID</dt><dd>${user.org_id}</dd>
    </dl>
  `;
}

function showApiResponse(endpoint: string, data: unknown, status: number) {
  endpointLabel.innerHTML = `<code>${endpoint}</code> — ${status}`;
  apiResult.textContent = JSON.stringify(data, null, 2);
  apiResponseSection.classList.remove("hidden");
}

// --- Unauthenticated endpoint tests (no token) ---

function showUnauthResponse(endpoint: string, data: unknown, status: number) {
  unauthEndpointLabel.innerHTML = `<code>${endpoint}</code> — <span style="color:#f87171">${status} Unauthorized</span>`;
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

// --- Toggle login / signup ---

showSignupBtn.addEventListener("click", showSignup);
showLoginBtn.addEventListener("click", showLogin);

// --- Signup ---

signupBtn.addEventListener("click", async () => {
  signupError.classList.add("hidden");
  signupFieldErrors.classList.add("hidden");
  signupSuccess.classList.add("hidden");

  try {
    const user = await client.register({
      username: regUsername.value,
      email: regEmail.value,
      password: regPassword.value,
      given_name: regGivenName.value || undefined,
      family_name: regFamilyName.value || undefined,
    });

    signupSuccess.textContent = `Account created for ${user.email}. You can now login.`;
    signupSuccess.classList.remove("hidden");

    // Auto-switch to login after 1.5s
    setTimeout(() => {
      identifierInput.value = regUsername.value;
      passwordInput.value = "";
      showLogin();
    }, 1500);
  } catch (err: unknown) {
    const rampartErr = err as {
      error_description?: string;
      fields?: { field: string; message: string }[];
    };

    if (rampartErr.fields?.length) {
      signupFieldErrors.innerHTML = rampartErr.fields
        .map((f) => `<li><strong>${f.field}:</strong> ${f.message}</li>`)
        .join("");
      signupFieldErrors.classList.remove("hidden");
    } else {
      signupError.textContent =
        rampartErr.error_description ?? "Registration failed";
      signupError.classList.remove("hidden");
    }
  }
});

// --- Login ---

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

// --- Init ---

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
