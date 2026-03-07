// Rampart OAuth 2.0 PKCE Client Helper
//
// This module handles the full Authorization Code + PKCE flow
// against a Rampart IAM server.

const RAMPART_URL = "http://localhost:8080";
const CLIENT_ID = "demo-react-app";
const REDIRECT_URI = "http://localhost:5173/callback";

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

  // Store verifier so we can use it in the callback
  sessionStorage.setItem("pkce_code_verifier", verifier);

  const params = new URLSearchParams({
    response_type: "code",
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    code_challenge: challenge,
    code_challenge_method: "S256",
    scope: "openid profile email",
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
