// Rampart OAuth 2.0 PKCE Client Helper
//
// This module handles the full Authorization Code + PKCE flow
// against a Rampart IAM server.

const RAMPART_URL = "http://localhost:8080";
const CLIENT_ID = "sample-react-app";
const REDIRECT_URI = "http://localhost:3002/callback";

// --- PKCE helpers (Web Crypto API) ---

function base64UrlEncode(buffer) {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const b of bytes) {
    binary += String.fromCharCode(b);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

export async function generateCodeChallenge(verifier) {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return base64UrlEncode(digest);
}

// --- OAuth flow ---

export async function login() {
  const verifier = generateCodeVerifier();
  const challenge = await generateCodeChallenge(verifier);
  const state = generateCodeVerifier(); // random state for CSRF protection

  // Store verifier and state so we can use them in the callback
  sessionStorage.setItem("pkce_code_verifier", verifier);
  sessionStorage.setItem("oauth_state", state);

  const params = new URLSearchParams({
    response_type: "code",
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    code_challenge: challenge,
    code_challenge_method: "S256",
    scope: "openid profile email",
    state,
  });

  window.location.href = `${RAMPART_URL}/oauth/authorize?${params.toString()}`;
}

export async function handleCallback() {
  const params = new URLSearchParams(window.location.search);
  const code = params.get("code");
  const error = params.get("error");

  if (error) {
    throw new Error(`OAuth error: ${error} - ${params.get("error_description") || ""}`);
  }

  if (!code) {
    throw new Error("No authorization code found in callback URL");
  }

  // Validate state to prevent CSRF
  const returnedState = params.get("state");
  const savedState = sessionStorage.getItem("oauth_state");
  if (!returnedState || returnedState !== savedState) {
    throw new Error("State mismatch — possible CSRF attack");
  }

  const verifier = sessionStorage.getItem("pkce_code_verifier");
  if (!verifier) {
    throw new Error("No PKCE code verifier found. Did the login flow start correctly?");
  }

  const response = await fetch(`${RAMPART_URL}/oauth/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: REDIRECT_URI,
      client_id: CLIENT_ID,
      code_verifier: verifier,
    }),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Token exchange failed (${response.status}): ${text}`);
  }

  const tokens = await response.json();

  localStorage.setItem("access_token", tokens.access_token);
  if (tokens.refresh_token) {
    localStorage.setItem("refresh_token", tokens.refresh_token);
  }
  if (tokens.id_token) {
    localStorage.setItem("id_token", tokens.id_token);
  }

  sessionStorage.removeItem("pkce_code_verifier");
  sessionStorage.removeItem("oauth_state");

  return tokens;
}

export function getAccessToken() {
  return localStorage.getItem("access_token");
}

export function logout() {
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
  localStorage.removeItem("id_token");
}

// --- Social Login ---

export async function socialLogin(provider) {
  const verifier = generateCodeVerifier();
  const challenge = await generateCodeChallenge(verifier);
  const state = generateCodeVerifier();

  sessionStorage.setItem("pkce_code_verifier", verifier);
  sessionStorage.setItem("oauth_state", state);

  const params = new URLSearchParams({
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: "openid profile email",
    state,
    code_challenge: challenge,
    nonce: generateCodeVerifier(),
  });

  window.location.href = `${RAMPART_URL}/oauth/social/${provider}?${params.toString()}`;
}

// --- Password Reset ---

export async function forgotPassword(email) {
  const response = await fetch(`${RAMPART_URL}/forgot-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  return response.json();
}

export async function resetPassword(token, newPassword) {
  const response = await fetch(`${RAMPART_URL}/reset-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token, new_password: newPassword }),
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error_description || "Reset failed");
  }
  return response.json();
}

// --- MFA ---

export async function verifyMFA(mfaToken, code) {
  const response = await fetch(`${RAMPART_URL}/mfa/totp/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mfa_token: mfaToken, code }),
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error_description || "MFA verification failed");
  }
  const tokens = await response.json();
  localStorage.setItem("access_token", tokens.access_token);
  if (tokens.refresh_token) {
    localStorage.setItem("refresh_token", tokens.refresh_token);
  }
  return tokens;
}

// --- Direct Login (for MFA flow) ---

export async function directLogin(identifier, password) {
  const response = await fetch(`${RAMPART_URL}/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ identifier, password }),
  });
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error_description || "Login failed");
  }
  // If MFA required, return the mfa_token
  if (data.mfa_required) {
    return { mfaRequired: true, mfaToken: data.mfa_token };
  }
  // Normal login — store tokens
  localStorage.setItem("access_token", data.access_token);
  if (data.refresh_token) {
    localStorage.setItem("refresh_token", data.refresh_token);
  }
  return { mfaRequired: false };
}

// --- JWT helpers (client-side decode, no signature verification) ---

function decodeBase64Url(str) {
  let base64 = str.replace(/-/g, "+").replace(/_/g, "/");
  // Pad with '=' to make length a multiple of 4
  while (base64.length % 4 !== 0) {
    base64 += "=";
  }
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return new TextDecoder().decode(bytes);
}

export function parseJwt(token) {
  if (!token) return null;
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = decodeBase64Url(parts[1]);
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

export function getUserInfo() {
  const token = getAccessToken();
  return parseJwt(token);
}

export function isAuthenticated() {
  const token = getAccessToken();
  if (!token) return false;
  const claims = parseJwt(token);
  if (!claims) return false;
  // Check expiration
  if (claims.exp && claims.exp * 1000 < Date.now()) {
    logout();
    return false;
  }
  return true;
}
