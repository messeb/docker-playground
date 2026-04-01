'use strict';

const POLL_INTERVAL_MS = 2000;

const els = {
  position:     document.getElementById('position'),
  total:        document.getElementById('total'),
  progressFill: document.getElementById('progress-fill'),
  etaBlock:     document.getElementById('eta-block'),
  etaText:      document.getElementById('eta-text'),
};

let lastPosition = null;

async function pollPosition() {
  try {
    const res = await fetch('/api/queue/position', { credentials: 'same-origin' });

    if (!res.ok) {
      scheduleNext();
      return;
    }

    const data = await res.json();

    switch (data.status) {
      case 'active':
        // We've been admitted — redirect to the protected page.
        window.location.href = data.redirect || '/';
        return;

      case 'queued':
        updateUI(data);
        break;

      case 'unknown':
      case 'no_session':
        // Session is gone — visit root to re-enter the queue.
        window.location.href = '/';
        return;

      default:
        break;
    }
  } catch (_) {
    // Network error — will retry.
  }

  scheduleNext();
}

function updateUI(data) {
  const pos   = data.position ?? 0;
  const total = data.total    ?? 0;

  // Position number with a brief scale animation when it changes.
  if (pos !== lastPosition) {
    els.position.textContent = pos;
    els.position.classList.remove('bump');
    // Force reflow so re-adding the class triggers the animation.
    void els.position.offsetWidth;
    els.position.classList.add('bump');
    lastPosition = pos;
  }

  els.total.textContent = total;

  // Progress bar: higher = closer to front (minimum 4 %).
  const pct = total > 0
    ? Math.max(4, Math.round((1 - (pos - 1) / total) * 100))
    : 4;
  els.progressFill.style.width = pct + '%';
  els.progressFill.closest('[role=progressbar]').setAttribute('aria-valuenow', pct);

  // ETA.
  const eta = data.eta_seconds;
  if (eta != null && eta >= 0) {
    els.etaText.textContent = 'Estimated wait: ' + formatETA(eta);
  } else {
    els.etaText.textContent = 'Calculating estimated wait…';
  }
}

function formatETA(seconds) {
  if (seconds === 0) return 'almost there';
  if (seconds < 60)  return `about ${seconds}s`;
  const mins = Math.ceil(seconds / 60);
  return `about ${mins} min${mins !== 1 ? 's' : ''}`;
}

function scheduleNext() {
  setTimeout(pollPosition, POLL_INTERVAL_MS);
}

// Kick off immediately.
pollPosition();

// ── Heartbeat ─────────────────────────────────────────────────────────────────
// Keep the queue position alive while the tab is open. Without this, the
// server-side reaper will remove the session after the heartbeat TTL expires.

function pingHeartbeat() {
  fetch('/api/session/heartbeat', { method: 'POST', credentials: 'same-origin' })
    .catch(() => {});
}
setInterval(pingHeartbeat, 10_000);

// Queue positions are released by the server-side heartbeat reaper, not by a
// leave beacon. This is intentional: pagehide fires on both tab-close AND
// page-refresh, so a beacon here would evict the session on every refresh.
// The reaper frees the slot within ~35 s after the last heartbeat (30 s TTL +
// up to 5 s reaper tick) — acceptable for a queue system.
//
// The target page (active users) keeps its own immediate leave beacon.
