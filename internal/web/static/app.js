/**
 * PanGo — Lightweight web control
 * Form handling, POST /run and SSE console stream
 */

(function () {
  const form = document.getElementById('capture-form');
  const launchBtn = document.getElementById('launch-btn');
  const cancelBtn = document.getElementById('cancel-btn');
  const consoleEl = document.getElementById('console');
  const statusBadge = document.getElementById('status-badge');

  let evtSource = null;
  let isRunning = false;

  async function loadFormDefaults() {
    try {
      const res = await fetch('/config');
      if (res.ok) {
        const cfg = await res.json();
        form.horizontal_angle_deg.value = cfg.horizontal_angle_deg ?? 180;
        form.vertical_angle_deg.value = cfg.vertical_angle_deg ?? 30;
        form.focal_length_mm.value = cfg.focal_length_mm ?? 35;
      }
    } catch (_) {
      form.horizontal_angle_deg.value = 180;
      form.vertical_angle_deg.value = 30;
      form.focal_length_mm.value = 35;
    }
  }

  function setStatus(status, label) {
    statusBadge.className = 'status-badge status-' + status;
    statusBadge.textContent = label;
    isRunning = status === 'running';
    launchBtn.disabled = isRunning;
    cancelBtn.disabled = !isRunning;
  }

  function appendConsole(msg, level) {
    const line = document.createElement('div');
    line.className = 'console-line';
    if (level) line.dataset.level = level;
    line.textContent = typeof msg === 'string' ? msg : msg.msg || msg;
    consoleEl.appendChild(line);
    consoleEl.scrollTop = consoleEl.scrollHeight;
  }

  function connectSSE() {
    if (evtSource) evtSource.close();
    evtSource = new EventSource('/status/stream');
    evtSource.onmessage = function (e) {
      try {
        const data = JSON.parse(e.data);
        appendConsole(data.msg || data, data.level || 'info');
        if (/\b(complete|cancelled|failed)\b/i.test(data.msg || '')) {
          setStatus('idle', 'Idle');
        }
      } catch {
        appendConsole(e.data);
      }
    };
    evtSource.onerror = function () {
      evtSource.close();
      evtSource = null;
    };
  }

  form.addEventListener('submit', async function (e) {
    e.preventDefault();
    if (isRunning) return;

    const payload = {
      horizontal_angle_deg: parseFloat(form.horizontal_angle_deg.value),
      vertical_angle_deg: parseFloat(form.vertical_angle_deg.value),
      focal_length_mm: parseFloat(form.focal_length_mm.value)
    };

    setStatus('running', 'Running…');

    try {
      const res = await fetch('/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (res.status === 409) {
        appendConsole('Capture already in progress.', 'error');
        setStatus('error', 'Busy');
        return;
      }

      if (!res.ok) {
        const err = await res.text();
        appendConsole('Error: ' + (err || res.status), 'error');
        setStatus('error', 'Error');
        return;
      }

      appendConsole('Capture launched.', 'info');
    } catch (err) {
      appendConsole('Network error: ' + err.message, 'error');
      setStatus('error', 'Error');
    }
  });

  cancelBtn.addEventListener('click', async function () {
    if (!isRunning) return;

    try {
      const res = await fetch('/cancel', { method: 'POST' });

      if (res.ok) {
        appendConsole('Cancellation requested...', 'warning');
        cancelBtn.disabled = true;
      } else if (res.status === 409) {
        appendConsole('No capture in progress.', 'info');
      } else {
        const err = await res.text();
        appendConsole('Cancel failed: ' + (err || res.status), 'error');
      }
    } catch (err) {
      appendConsole('Network error: ' + err.message, 'error');
    }
  });

  loadFormDefaults();
  connectSSE();
})();
