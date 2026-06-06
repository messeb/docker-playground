// Tiny OIDC authorization-code + PKCE client for the banking-mfa realm.
// No build step, no library — just enough to demonstrate the flow.
//
// Keycloak: http://localhost:8180/realms/banking-mfa
// Client  : banking-spa  (public, PKCE S256)
// Redirect: http://localhost:8080/callback.html

const KC = "http://localhost:8180/realms/banking-mfa";
const CLIENT_ID = "banking-spa";
const REDIRECT_URI = "http://localhost:8080/callback.html";

const OIDC = {
  async login() {
    const verifier = randomString(64);
    const challenge = await s256(verifier);
    sessionStorage.setItem("pkce_verifier", verifier);

    const state = randomString(16);
    sessionStorage.setItem("oauth_state", state);

    const url = new URL(`${KC}/protocol/openid-connect/auth`);
    url.searchParams.set("response_type", "code");
    url.searchParams.set("client_id", CLIENT_ID);
    url.searchParams.set("redirect_uri", REDIRECT_URI);
    url.searchParams.set("scope", "openid email profile");
    url.searchParams.set("state", state);
    url.searchParams.set("code_challenge", challenge);
    url.searchParams.set("code_challenge_method", "S256");
    location.href = url.toString();
  },

  async handleCallback() {
    const params = new URLSearchParams(location.search);
    const code = params.get("code");
    const state = params.get("state");
    if (!code) {
      document.getElementById("status").textContent = "Error: no code in callback URL.";
      return;
    }
    if (state !== sessionStorage.getItem("oauth_state")) {
      document.getElementById("status").textContent = "Error: state mismatch.";
      return;
    }
    const verifier = sessionStorage.getItem("pkce_verifier");
    const body = new URLSearchParams({
      grant_type: "authorization_code",
      client_id: CLIENT_ID,
      code,
      redirect_uri: REDIRECT_URI,
      code_verifier: verifier,
    });
    // Exchange the auth-code at the BFF, not directly at Keycloak. The BFF
    // forwards to Keycloak, intercepts the response, and wraps the access
    // token into JWE before returning. The SPA never sees the plain JWS.
    const res = await fetch("/api/token", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body,
    });
    if (!res.ok) {
      document.getElementById("status").textContent = `Token exchange failed: ${res.status}`;
      return;
    }
    const json = await res.json();
    sessionStorage.setItem("id_token", json.id_token || "");
    sessionStorage.setItem("refresh_token", json.refresh_token || "");
    sessionStorage.setItem("access_token", json.access_token);   // already a JWE
    location.href = "/";
  },

  // Exchange refresh_token for a new {access_token, id_token, refresh_token}
  // and rewrap into a JWE-for-BFF. Returns true on success. The new JWS
  // payload picks up whatever protocol mappers evaluate to *now* — in
  // particular, mfa_verified flips to true right after enroll/verify.
  async refresh() {
    const rt = sessionStorage.getItem("refresh_token");
    if (!rt) return false;
    // Refresh-grant also goes through the BFF token proxy; the new access
    // token comes back already wrapped as JWE.
    const res = await fetch("/api/token", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "refresh_token",
        client_id: CLIENT_ID,
        refresh_token: rt,
      }),
    });
    if (!res.ok) return false;
    const json = await res.json();
    sessionStorage.setItem("id_token", json.id_token || "");
    sessionStorage.setItem("refresh_token", json.refresh_token || rt);
    sessionStorage.setItem("access_token", json.access_token);   // already a JWE
    return true;
  },

  logout() {
    const idToken = sessionStorage.getItem("id_token");
    sessionStorage.clear();
    const url = new URL(`${KC}/protocol/openid-connect/logout`);
    url.searchParams.set("client_id", CLIENT_ID);
    url.searchParams.set("post_logout_redirect_uri", "http://localhost:8080/");
    // Only attach id_token_hint when it's still parseable + not expired.
    // After a Keycloak restart with fresh keys (volume wipe, key rotation)
    // a stale id_token would trigger "Invalid parameter: id_token_hint".
    if (idToken) {
      const payload = decodeJwtPayload(idToken);
      const now = Math.floor(Date.now() / 1000);
      if (payload && payload.exp && payload.exp - 30 > now) {
        url.searchParams.set("id_token_hint", idToken);
      }
    }
    location.href = url.toString();
  },

  token() { return sessionStorage.getItem("access_token"); },
};

function randomString(len) {
  const a = new Uint8Array(len);
  crypto.getRandomValues(a);
  return base64url(a);
}

async function s256(verifier) {
  const buf = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  return base64url(new Uint8Array(buf));
}

function base64url(bytes) {
  let str = btoa(String.fromCharCode(...bytes));
  return str.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function decodeJwtPayload(jws) {
  try {
    const part = jws.split(".")[1];
    const pad = "=".repeat((4 - part.length % 4) % 4);
    const json = atob(part.replace(/-/g, "+").replace(/_/g, "/") + pad);
    return JSON.parse(json);
  } catch { return null; }
}

window.OIDC = OIDC;

// ── UI wiring (only fires on the index page) ────────────────────────────────
window.addEventListener("DOMContentLoaded", () => {
  const loginBtn = document.getElementById("loginBtn");
  if (!loginBtn) return; // we're on callback.html

  const token = OIDC.token();
  const loggedOut = document.getElementById("loggedOut");
  const loggedIn  = document.getElementById("loggedIn");

  if (!token) {
    loggedOut.hidden = false;
    loggedIn.hidden  = true;
    loginBtn.addEventListener("click", () => OIDC.login());
    return;
  }

  loggedOut.hidden = true;
  loggedIn.hidden  = false;

  // The access_token is a JWE — only the wrapper can decrypt it. For UI
  // display we read the id_token (JWS) which Keycloak emits alongside it.
  const payload = decodeJwtPayload(sessionStorage.getItem("id_token") || "") || {};
  const hasMFA  = !!payload.mfa_verified;
  const email   = payload.email || "";

  document.getElementById("who").textContent =
    payload.preferred_username || payload.sub || "(unknown)";
  renderMFAState(hasMFA, payload.mfa_method);

  document.getElementById("logoutBtn").addEventListener("click", () => OIDC.logout());

  document.getElementById("enableMfaBtn").addEventListener("click", () => {
    window.openMFADialog && window.openMFADialog(email, null);
  });

  document.querySelectorAll("button[data-call]").forEach(b => {
    b.addEventListener("click", () => apiCall(b.dataset.call, "GET"));
  });

  // Write buttons. When MFA is off they're styled red AND clicking opens the
  // enrollment dialog first; the original request gets replayed on success.
  document.querySelectorAll("button[data-post]").forEach(b => {
    if (!hasMFA) b.classList.add("danger");
    b.addEventListener("click", () => {
      const path = b.dataset.post, body = b.dataset.body;
      if (!hasMFA) {
        window.openMFADialog && window.openMFADialog(email, { path, method: "POST", body });
        return;
      }
      apiCall(path, "POST", body);
    });
  });

  wireMFADialog();
});

function renderMFAState(hasMFA, method) {
  const badge  = document.getElementById("mfaBadge");
  const prompt = document.getElementById("mfaPromptCard");
  const note   = document.getElementById("writeMfaNote");
  if (hasMFA) {
    badge.className = "badge on";
    badge.textContent = "🔒 MFA · " + (method || "email");
    prompt.hidden = true;
    note.hidden = true;
  } else {
    badge.className = "badge off";
    badge.textContent = "🔓 MFA off";
    prompt.hidden = false;
    note.hidden = false;
  }
}

function wireMFADialog() {
  const dlg = document.getElementById("mfaDialog");
  if (!dlg) return;

  const stepEmail = document.getElementById("mfaStepEmail");
  const stepCode  = document.getElementById("mfaStepCode");
  const emailIn   = document.getElementById("mfaEmail");
  const codeIn    = document.getElementById("mfaCode");
  const msg       = document.getElementById("mfaDialogMsg");
  const msg2      = document.getElementById("mfaDialogMsg2");
  const sentTo    = document.getElementById("mfaSentTo");
  const sendBtn   = document.getElementById("mfaSendCode");
  const verifyBtn = document.getElementById("mfaVerify");
  const cancelBtn = document.getElementById("mfaCancel");
  const backBtn   = document.getElementById("mfaBack");

  // Stash the (method, path, body) that triggered the modal so we can replay
  // it transparently once verification succeeds.
  let pendingReplay = null;

  function showStep(which) {
    stepEmail.hidden = (which !== "email");
    stepCode.hidden  = (which !== "code");
    msg.textContent = "";
    msg2.textContent = "In dev the code arrives at Mailpit.";
  }

  cancelBtn.addEventListener("click", () => dlg.close());
  backBtn.addEventListener("click", () => showStep("email"));

  sendBtn.addEventListener("click", async () => {
    const email = emailIn.value.trim();
    if (!email) { msg.textContent = "Email is required."; return; }
    msg.textContent = "Sending code…";
    sendBtn.disabled = true;
    try {
      const res = await fetch("/api/v1/mfa/enroll/start", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": "Bearer " + OIDC.token(),
        },
        body: JSON.stringify({ email }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        msg.textContent = "Failed: " + (json.error || res.status);
        return;
      }
      sentTo.textContent = email;
      showStep("code");
      setTimeout(() => codeIn.focus(), 0);
    } finally {
      sendBtn.disabled = false;
    }
  });

  verifyBtn.addEventListener("click", async () => {
    const code = (codeIn.value || "").trim();
    if (!/^[0-9]{6}$/.test(code)) { msg2.textContent = "Enter the 6-digit code."; return; }
    msg2.textContent = "Verifying…";
    verifyBtn.disabled = true;
    try {
      const res = await fetch("/api/v1/mfa/enroll/verify", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": "Bearer " + OIDC.token(),
        },
        body: JSON.stringify({ code }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        msg2.textContent = "Failed: " + (json.error || res.status);
        return;
      }
      msg2.innerHTML = "✅ Verified. Refreshing token…";
      // Exchange refresh_token for a fresh JWS — Keycloak's protocol mappers
      // re-evaluate now and emit mfa_verified=true / mfa_method=email since
      // the API just flipped the user attribute and cleared the KC cache.
      const refreshed = await OIDC.refresh();
      dlg.close();
      const id = decodeJwtPayload(sessionStorage.getItem("id_token") || "") || {};
      // Update the badge + un-redden the write buttons.
      renderMFAState(!!id.mfa_verified, id.mfa_method);
      document.querySelectorAll("button.write-btn.danger")
        .forEach(b => b.classList.remove("danger"));
      if (pendingReplay) {
        const { path, method, body } = pendingReplay;
        pendingReplay = null;
        apiCall(path, method, body);
      }
    } finally {
      verifyBtn.disabled = false;
    }
  });

  window.openMFADialog = function(currentEmail, replay) {
    pendingReplay = replay || null;
    showStep("email");
    emailIn.value = currentEmail || "";
    codeIn.value  = "";
    dlg.showModal();
  };
}

async function apiCall(path, method, body) {
  const out = document.getElementById("out");
  out.textContent = "…";
  const res = await fetch(path, {
    method,
    headers: {
      "Authorization": "Bearer " + OIDC.token(),
      ...(body ? { "Content-Type": "application/json" } : {}),
    },
    body: body || undefined,
  });
  const text = await res.text();
  let parsed = null;
  try { parsed = JSON.parse(text); } catch {}
  out.textContent = `HTTP ${res.status}\n` + (parsed ? JSON.stringify(parsed, null, 2) : text);

  // When the API rejects a non-MFA caller, offer the in-place enrollment.
  // The modal replays this exact request on success so the user stays logged in.
  if (res.status === 403 && parsed && /mfa required/i.test(parsed.error || "")) {
    const id = decodeJwtPayload(sessionStorage.getItem("id_token") || "") || {};
    window.openMFADialog && window.openMFADialog(id.email || "", { path, method, body });
  }
}
