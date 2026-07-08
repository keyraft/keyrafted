(() => {
  const TOKEN_KEY = "keyraft_token";
  const $ = (s, el = document) => el.querySelector(s);
  const $$ = (s, el = document) => [...el.querySelectorAll(s)];

  const state = {
    token: sessionStorage.getItem(TOKEN_KEY) || "",
    me: null,
    namespaces: [],
    currentNs: null,
    keys: [],
    selectedKey: null,
    detailEntry: null,
    secretRevealed: false,
    panel: "secrets",
    watching: false,
    watchAbort: null,
    health: null,
    viewingVersion: null,
    auditPage: 0,
    auditPageSize: 25,
  };

  // ── API ──────────────────────────────────────────────────────────
  function kvBase(ns, key, suffix = "") {
    const base = `/v1/kv/${ns}${key ? `/${encodeURIComponent(key)}` : ""}`;
    return suffix ? `${base}/${suffix}` : base;
  }

  function nsApiPath(ns) {
    return `/v1/namespaces/${ns}`;
  }

  async function api(path, opts = {}) {
    const headers = { ...(opts.headers || {}) };
    if (state.token) headers.Authorization = `Bearer ${state.token}`;
    if (opts.body && !(opts.body instanceof FormData)) {
      headers["Content-Type"] = "application/json";
      opts.body = typeof opts.body === "string" ? opts.body : JSON.stringify(opts.body);
    }
    const res = await fetch(path, { ...opts, headers });
    const text = await res.text();
    let data = null;
    try { data = text ? JSON.parse(text) : null; } catch { data = { error: text }; }
    if (!res.ok) {
      const err = new Error((data && data.error) || text || res.statusText);
      err.status = res.status;
      throw err;
    }
    return data;
  }

  function esc(s) {
    return String(s ?? "")
      .replace(/&/g, "&amp;").replace(/</g, "&lt;")
      .replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }

  function fmtTime(d) {
    return new Date(d).toLocaleString();
  }

  function toast(msg) {
    const el = $("#toast");
    el.textContent = msg;
    el.classList.remove("hidden");
    clearTimeout(toast._t);
    toast._t = setTimeout(() => el.classList.add("hidden"), 2800);
  }

  function btn(label, cls, onClick) {
    const b = document.createElement("button");
    b.className = `btn ${cls}`;
    b.textContent = label;
    b.type = "button";
    b.addEventListener("click", (e) => { e.stopPropagation(); onClick(); });
    return b;
  }

  // ── Auth ─────────────────────────────────────────────────────────
  function showLogin(err = "") {
    document.documentElement.classList.remove("has-session");
    $("#login-view").classList.remove("hidden");
    $("#main-view").classList.add("hidden");
    $("#login-error").textContent = err;
  }

  function showMain() {
    document.documentElement.classList.add("has-session");
    $("#login-view").classList.add("hidden");
    $("#main-view").classList.remove("hidden");
    const role = state.me?.role || (state.me?.is_root ? "root" : "scoped");
    const name = state.me?.metadata?.name || state.me?.id?.slice(0, 8);
    $("#session-meta").textContent = `${name} · ${role}`;
    $$(".admin-only").forEach((el) => el.classList.toggle("hidden", !state.me?.can_manage_tokens));
    $$(".audit-only").forEach((el) => el.classList.toggle("hidden", !state.me?.can_view_audit));
    $$(".roles-only").forEach((el) => el.classList.toggle("hidden", !state.me?.can_manage_roles));
    $$(".nav-group").forEach((group) => {
      const visible = [...group.querySelectorAll(".nav-item")].some((el) => !el.classList.contains("hidden"));
      group.classList.toggle("hidden", !visible);
    });
    pollHealth();
    setInterval(pollHealth, 30000);
  }

  async function login(token) {
    state.token = token.trim();
    if (!state.token) throw new Error("Token is required");
    state.me = await api("/v1/auth/me");
    sessionStorage.setItem(TOKEN_KEY, state.token);
    showMain();
    try {
      await applyRoute(false);
    } catch (err) {
      toast(err.message);
      showPanel("secrets");
      await loadNamespaces().catch(() => {});
    }
  }

  function logout() {
    stopWatch();
    state.token = "";
    state.me = null;
    state.currentNs = null;
    state.selectedKey = null;
    sessionStorage.removeItem(TOKEN_KEY);
    document.body.classList.remove("nav-open");
    const overlay = $("#nav-overlay");
    if (overlay) overlay.hidden = true;
    showLogin();
  }

  async function pollHealth() {
    try {
      const h = await fetch("/v1/health").then((r) => r.json());
      state.health = h;
      const el = $("#health-badge");
      el.textContent = `healthy · v${h.version || "?"}`;
      el.className = "health-badge ok";
    } catch {
      $("#health-badge").textContent = "unreachable";
      $("#health-badge").className = "health-badge err";
    }
  }

  function updateBreadcrumbs() {
    const parts = [];
    if (state.panel === "secrets") {
      parts.push("KV");
      if (state.currentNs) parts.push(state.currentNs);
      if (state.selectedKey) parts.push(state.selectedKey);
    } else {
      const labels = { tokens: "Access / Tokens", roles: "Policies / Roles", audit: "Audit Log", watch: "Live Watch" };
      parts.push(labels[state.panel] || state.panel);
    }
    $("#breadcrumbs").innerHTML = parts.map((p, i) =>
      i === 0 ? `<strong>${esc(p)}</strong>` : `<span>/</span><strong>${esc(p)}</strong>`
    ).join("");
  }

  // ── Namespace list (flat) ─────────────────────────────────────────
  function renderNamespaceTree() {
    const filter = ($("#ns-search").value || "").toLowerCase().trim();
    const names = state.namespaces.map((n) => n.name).sort();
    const tree = $("#ns-tree");
    tree.innerHTML = "";
    $("#ns-empty").classList.toggle("hidden", names.length > 0);
    if (!names.length) return;

    const ul = document.createElement("ul");
    ul.className = "ns-node";
    for (const name of names) {
      if (filter && !name.toLowerCase().includes(filter)) continue;

      const li = document.createElement("li");
      const row = document.createElement("div");
      row.className = "ns-row";
      row.dataset.nsPath = name;

      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "ns-item" + (state.currentNs === name ? " active" : "");
      btn.title = name;
      btn.innerHTML = `<svg class="folder-icon"><use href="#icon-folder"/></svg><span>${esc(name)}</span>`;
      btn.addEventListener("click", () => selectNamespace(name));
      row.addEventListener("contextmenu", (e) => {
        e.preventDefault();
        e.stopPropagation();
        showNsContextMenu(e.clientX, e.clientY, name);
      });
      row.appendChild(btn);
      li.appendChild(row);
      ul.appendChild(li);
    }
    if (ul.childNodes.length) tree.appendChild(ul);
  }

  async function loadNamespaces({ preserveUrl = false } = {}) {
    const data = await api("/v1/namespaces");
    state.namespaces = data.namespaces || [];
    renderNamespaceTree();
    populateWatchSelect();

    const route = parseRoute();
    const { ns: urlNs, key: urlKey } = route.panel === "secrets"
      ? resolveKvRest(route.kvRest, state.namespaces)
      : { ns: null, key: null };

    if (urlNs && state.namespaces.some((n) => n.name === urlNs)) {
      await selectNamespace(urlNs, { push: false, sync: !preserveUrl });
      if (urlKey) {
        try { await openDetail(urlKey, { push: false, sync: !preserveUrl }); }
        catch { toast(`Key not found: ${urlKey}`); syncUrl(false); }
      }
      return;
    }

    if (state.currentNs && state.namespaces.some((n) => n.name === state.currentNs)) {
      await selectNamespace(state.currentNs, { push: false });
      return;
    }

    if (state.namespaces.length) {
      await selectNamespace(state.namespaces[0].name, { push: false });
      return;
    }

    state.currentNs = null;
    state.selectedKey = null;
    state.keys = [];
    renderKeys();
    syncUrl(false);
  }

  async function selectNamespace(name, { push = true, sync = true } = {}) {
    state.currentNs = name;
    state.selectedKey = null;
    state.detailEntry = null;
    $("#keys-view").classList.remove("hidden");
    $("#detail-view").classList.add("hidden");
    renderNamespaceTree();
    $("#current-path").textContent = name;
    $("#key-filter").disabled = false;
    updateBreadcrumbs();
    if (sync) syncUrl(push);
    await loadKeys(name);
  }

  async function loadKeys(ns) {
    try {
      const data = await api(kvBase(ns));
      // List responses have keys[]; a mistaken key-get has key/value at top level
      if (data && Array.isArray(data.keys)) {
        state.keys = data.keys;
      } else if (data && data.key !== undefined) {
        state.keys = [data];
      } else {
        state.keys = [];
      }
      renderKeys();
    } catch (err) {
      state.keys = [];
      renderKeys();
      throw err;
    }
  }

  function renderKeys() {
    const filter = ($("#key-filter").value || "").toLowerCase();
    const rows = state.keys.filter((k) => !filter || k.key.toLowerCase().includes(filter));
    const tbody = $("#keys-tbody");
    tbody.innerHTML = "";
    const empty = $("#keys-empty");
    empty.classList.toggle("hidden", rows.length > 0 || !state.currentNs);
    if (!state.currentNs) { empty.textContent = "Select a namespace to list secrets."; return; }
    if (!rows.length) { empty.textContent = filter ? "No keys match filter." : "No secrets at this path."; return; }

    for (const entry of rows) {
      const tr = document.createElement("tr");
      tr.className = "row-clickable";
      tr.innerHTML = `
        <td class="mono">${esc(entry.key)}</td>
        <td><span class="badge ${esc(entry.type)}">${esc(entry.type)}</span></td>
        <td>v${entry.version}</td>
        <td>${fmtTime(entry.updated_at)}</td>
        <td class="actions-cell"></td>`;
      tr.addEventListener("click", () => openDetail(entry.key).catch((e) => toast(e.message)));
      const actions = tr.querySelector(".actions-cell");
      actions.addEventListener("click", (e) => e.stopPropagation());
      actions.append(
        btn("Edit", "btn-ghost btn-sm", () => openSecretModal(entry)),
        btn("Delete", "btn-danger btn-sm", () => confirmDelete(entry))
      );
      tbody.appendChild(tr);
    }
  }

  // ── Detail view ──────────────────────────────────────────────────
  async function openDetail(key, { push = true, sync = true } = {}) {
    state.selectedKey = key;
    state.secretRevealed = false;
    state.viewingVersion = null;
    const entry = await api(kvBase(state.currentNs, key));
    if (!entry || entry.key === undefined) {
      throw new Error("Unexpected response when loading secret");
    }
    state.detailEntry = entry;
    $("#keys-view").classList.add("hidden");
    $("#detail-view").classList.remove("hidden");
    $("#detail-path").textContent = `${state.currentNs}/${key}`;
    const ver = entry.version != null ? `v${entry.version}` : "—";
    const typ = entry.type || "config";
    const updated = entry.updated_at ? fmtTime(entry.updated_at) : "—";
    $("#detail-meta").textContent = `${ver} · ${typ} · Updated ${updated}`;
    renderDetailValue();
    renderMetadata();
    await loadVersions();
    switchTab("current");
    updateBreadcrumbs();
    if (sync) syncUrl(push);
  }

  function closeDetail() {
    state.selectedKey = null;
    state.detailEntry = null;
    $("#detail-view").classList.add("hidden");
    $("#keys-view").classList.remove("hidden");
    updateBreadcrumbs();
    syncUrl(true);
  }

  function renderDetailValue() {
    const e = state.detailEntry;
    if (!e) return;
    const isSecret = e.type === "secret";
    const val = isSecret && !state.secretRevealed ? "••••••••••••••••" : e.value;
    $("#detail-value").textContent = val;
    const toggle = $("#toggle-secret");
    toggle.classList.toggle("hidden", !isSecret);
    toggle.textContent = state.secretRevealed ? "Hide" : "Reveal";
  }

  function renderMetadata() {
    const meta = state.detailEntry?.metadata || {};
    const keys = Object.keys(meta);
    const grid = $("#metadata-display");
    grid.innerHTML = "";
    $("#metadata-empty").classList.toggle("hidden", keys.length > 0);
    for (const k of keys) {
      const row = document.createElement("div");
      row.className = "meta-row";
      row.innerHTML = `<div class="meta-key">${esc(k)}</div><div class="mono">${esc(meta[k])}</div>`;
      grid.appendChild(row);
    }
  }

  function showHistoricalVersion(ver) {
    state.viewingVersion = ver.version;
    state.detailEntry = {
      ...state.detailEntry,
      value: ver.value,
      version: ver.version,
      type: ver.type,
      updated_at: ver.timestamp,
      metadata: ver.metadata || {},
    };
    state.secretRevealed = ver.type !== "secret";
    switchTab("current");
    renderDetailValue();
    renderMetadata();
    $("#detail-meta").textContent = `v${ver.version} · ${ver.type} · ${fmtTime(ver.timestamp)} · historical`;
    highlightVersionRow(ver.version);
  }

  function highlightVersionRow(version) {
    $$("#versions-tbody tr").forEach((tr) => {
      tr.classList.toggle("version-active", tr.dataset.version === String(version));
    });
  }

  async function loadVersions() {
    const tbody = $("#versions-tbody");
    tbody.innerHTML = "";
    try {
      const data = await api(kvBase(state.currentNs, state.selectedKey, "versions"));
      const versions = (data.versions || []).sort((a, b) => b.version - a.version);
      const latestVer = versions[0]?.version;
      for (const v of versions) {
        const masked = v.type === "secret" ? "••••••••" : v.value;
        const tr = document.createElement("tr");
        tr.dataset.version = String(v.version);
        if (state.viewingVersion === v.version) tr.classList.add("version-active");
        tr.innerHTML = `
          <td>v${v.version}${v.version === latestVer ? ' <span class="badge config">latest</span>' : ""}</td>
          <td><span class="badge ${esc(v.type)}">${esc(v.type)}</span></td>
          <td>${fmtTime(v.timestamp)}</td>
          <td class="mono" style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(masked)}</td>
          <td></td>`;
        tr.querySelector("td:last-child").appendChild(
          btn("View", "btn-ghost btn-sm", async () => {
            try {
              const ver = await api(`${kvBase(state.currentNs, state.selectedKey)}?version=${v.version}`);
              showHistoricalVersion(ver);
            } catch (err) {
              toast(err.message);
            }
          })
        );
        tbody.appendChild(tr);
      }
    } catch (err) {
      tbody.innerHTML = `<tr><td colspan="5" class="empty-msg">${esc(err.message)}</td></tr>`;
    }
  }

  function switchTab(name) {
    $$(".tab").forEach((t) => t.classList.toggle("active", t.dataset.tab === name));
    $$(".tab-panel").forEach((p) => p.classList.add("hidden"));
    $(`#tab-${name}`).classList.remove("hidden");
  }

  // ── Secret modal ─────────────────────────────────────────────────
  function openSecretModal(entry) {
    const form = $("#secret-form");
    $("#secret-form-error").textContent = "";
    $("#secret-modal-title").textContent = entry ? "Edit secret" : "Create secret";
    form.namespace.value = entry?.namespace || state.currentNs || "";
    form.key.value = entry?.key || "";
    form.key.readOnly = !!entry;
    form.namespace.readOnly = !!entry;
    form.type.value = entry?.type || "config";
    form.value.value = entry?.value || "";
    form.metadata.value = entry?.metadata ? JSON.stringify(entry.metadata, null, 2) : "";
    $("#secret-modal").showModal();
  }

  async function saveSecret(e) {
    e.preventDefault();
    const form = e.target;
    $("#secret-form-error").textContent = "";
    const namespace = form.namespace.value.trim();
    const key = form.key.value.trim();
    const body = { value: form.value.value, type: form.type.value };
    if (form.metadata.value.trim()) {
      try { body.metadata = JSON.parse(form.metadata.value); }
      catch { $("#secret-form-error").textContent = "Invalid metadata JSON"; return; }
    }
    try {
      await api(kvBase(namespace, key), { method: "PUT", body });
      $("#secret-modal").close();
      toast("Secret saved");
      await loadNamespaces();
      await selectNamespace(namespace);
      if (state.selectedKey === key || !state.selectedKey) await openDetail(key);
    } catch (err) {
      $("#secret-form-error").textContent = err.message;
    }
  }

  function confirmDelete(entry) {
    const ns = entry.namespace || state.currentNs;
    const key = entry.key;
    askConfirm(`Permanently delete secret at ${ns}/${key}?`, async () => {
      await api(kvBase(ns, key), { method: "DELETE" });
      toast("Secret deleted");
      closeDetail();
      await loadNamespaces();
      if (state.currentNs) await loadKeys(state.currentNs);
    });
  }

  let ctxNs = null;
  function showNsContextMenu(x, y, ns) {
    ctxNs = ns;
    const menu = $("#ns-ctx-menu");
    menu.style.left = `${x}px`;
    menu.style.top = `${y}px`;
    menu.classList.remove("hidden");
  }

  function hideNsContextMenu() {
    ctxNs = null;
    $("#ns-ctx-menu").classList.add("hidden");
  }

  function confirmDeleteNamespace(ns) {
    hideNsContextMenu();
    askConfirm(
      `Delete namespace "${ns}" and all secrets in it? This cannot be undone.`,
      async () => {
        await api(nsApiPath(ns), { method: "DELETE" });
        toast("Namespace deleted");
        if (state.currentNs === ns) {
          state.currentNs = null;
          state.selectedKey = null;
          state.detailEntry = null;
          $("#detail-view").classList.add("hidden");
          $("#keys-view").classList.remove("hidden");
        }
        await loadNamespaces();
        syncUrl(false);
      }
    );
  }

  // ── Tokens ───────────────────────────────────────────────────────
  async function loadTokens() {
    if (!state.me?.can_manage_tokens) return;
    const data = await api("/v1/auth/tokens");
    const tokens = data.tokens || [];
    const rootCount = tokens.filter(isRootTokenRow).length;
    const tbody = $("#tokens-tbody");
    tbody.innerHTML = "";
    $("#tokens-empty").classList.toggle("hidden", tokens.length > 0);
    for (const t of tokens) {
      const tr = document.createElement("tr");
      tr.className = "no-hover";
      const name = t.metadata?.name || "—";
      const lastRoot = isRootTokenRow(t) && rootCount <= 1;
      tr.innerHTML = `
        <td>${esc(name)}</td>
        <td><span class="badge config">${esc(t.role || "scoped")}</span></td>
        <td class="mono">${esc(t.id.slice(0, 12))}…</td>
        <td>${fmtTime(t.created_at)}</td>
        <td>${t.expires_at ? fmtTime(t.expires_at) : "never"}</td>
        <td class="actions-cell"></td>`;
      const revokeBtn = btn("Revoke", "btn-danger btn-sm", () => confirmRevoke(t));
      if (lastRoot) {
        revokeBtn.disabled = true;
        revokeBtn.title = "Cannot revoke the last root token";
      }
      tr.lastElementChild.appendChild(revokeBtn);
      tbody.appendChild(tr);
    }
  }

  function isRootTokenRow(t) {
    return t.metadata?.type === "root" || t.role === "admin" || (!t.role && !(t.scopes || []).length);
  }

  function openTokenModal() {
    $("#token-form").reset();
    $("#token-form-error").textContent = "";
    $("#token-created").classList.add("hidden");
    $("#token-submit").textContent = "Create";
    $("#token-role-fields").classList.remove("hidden");
    $("#token-scope-fields").classList.add("hidden");
    $("#token-modal").showModal();
  }

  async function createToken(e) {
    e.preventDefault();
    if (!$("#token-created").classList.contains("hidden")) {
      $("#token-modal").close();
      return;
    }
    const form = e.target;
    $("#token-form-error").textContent = "";
    const mode = form.mode.value;
    const body = { metadata: {} };
    if (form.name?.value?.trim()) body.metadata.name = form.name.value.trim();
    if (form.expires_in.value) body.expires_in = Number(form.expires_in.value);

    if (mode === "role") {
      body.role = form.role.value;
    } else {
      body.scopes = [{
        namespace: form.scope_ns.value.trim() || "*",
        read: form.scope_read.checked,
        write: form.scope_write.checked,
      }];
    }

    try {
      const tok = await api("/v1/auth/token", { method: "POST", body });
      $("#token-created-value").textContent = tok.token;
      $("#token-created").classList.remove("hidden");
      $("#token-submit").textContent = "Done";
      toast("Token created");
      await loadTokens();
    } catch (err) {
      $("#token-form-error").textContent = err.message;
    }
  }

  function confirmRevoke(t) {
    askConfirm(`Revoke token "${t.metadata?.name || t.id}"?`, async () => {
      await api(`/v1/auth/token/${encodeURIComponent(t.token)}`, { method: "DELETE" });
      toast("Token revoked");
      await loadTokens();
    });
  }

  // ── Roles ────────────────────────────────────────────────────────
  async function loadRoles() {
    if (!state.me?.can_manage_roles) return;
    const data = await api("/v1/roles");
    const grid = $("#roles-grid");
    grid.innerHTML = "";
    for (const role of data.roles || []) {
      const card = document.createElement("div");
      card.className = "role-card";
      card.innerHTML = `
        <h3><svg width="16" height="16"><use href="#icon-shield"/></svg> ${esc(role.name)}</h3>
        <p>${esc(role.description || "")}</p>
        <div class="perm-list">${(role.permissions || []).map((p) => `<span class="perm-tag">${esc(p)}</span>`).join("")}</div>`;
      grid.appendChild(card);
    }
  }

  // ── Audit ────────────────────────────────────────────────────────
  async function loadAudit(resetPage = false) {
    if (!state.me?.can_view_audit) return;
    if (resetPage) state.auditPage = 0;

    const ns = $("#audit-ns-filter").value.trim();
    const offset = state.auditPage * state.auditPageSize;
    const params = new URLSearchParams({
      limit: String(state.auditPageSize),
      offset: String(offset),
    });
    if (ns) params.set("namespace", ns);

    const data = await api(`/v1/audit?${params}`);
    const logs = data.logs || [];
    const total = data.total ?? 0;
    const tbody = $("#audit-tbody");
    tbody.innerHTML = "";
    $("#audit-empty").classList.toggle("hidden", logs.length > 0);

    for (const log of logs) {
      const tr = document.createElement("tr");
      tr.className = "no-hover";
      tr.innerHTML = `
        <td>${fmtTime(log.timestamp)}</td>
        <td><span class="badge config">${esc(log.action)}</span></td>
        <td class="mono">${esc(log.token_id?.slice(0, 8) || "")}…</td>
        <td class="mono">${esc(log.namespace)}</td>
        <td class="mono">${esc(log.key || "—")}</td>
        <td><span class="badge ${log.success ? "ok" : "fail"}">${log.success ? "success" : "failed"}</span></td>`;
      tbody.appendChild(tr);
    }

    const pag = $("#audit-pagination");
    pag.classList.toggle("hidden", total === 0);
    const from = total === 0 ? 0 : offset + 1;
    const to = offset + logs.length;
    $("#audit-page-info").textContent = `Showing ${from}–${to} of ${total}`;
    $("#audit-prev").disabled = state.auditPage === 0;
    $("#audit-next").disabled = !data.has_more;
  }

  // ── Watch ────────────────────────────────────────────────────────
  function populateWatchSelect() {
    const sel = $("#watch-ns");
    const cur = sel.value;
    sel.innerHTML = '<option value="">Select namespace…</option>';
    for (const ns of state.namespaces) {
      const opt = document.createElement("option");
      opt.value = ns.name;
      opt.textContent = ns.name;
      sel.appendChild(opt);
    }
    if (cur) sel.value = cur;
  }

  function appendWatchEvent(ev, timeout = false) {
    const box = $("#watch-events");
    const row = document.createElement("div");
    if (timeout) {
      row.className = "watch-event";
      row.innerHTML = `<span class="time">${new Date().toLocaleTimeString()}</span><span class="action" style="color:var(--text-dim)">— timeout, reconnecting…</span>`;
    } else {
      row.className = `watch-event ${esc(ev.action)}`;
      row.innerHTML = `
        <span class="time">${new Date(ev.timestamp || Date.now()).toLocaleTimeString()}</span>
        <span><span class="action">${esc(ev.action)}</span> <strong>${esc(ev.key)}</strong> in <code>${esc(ev.namespace)}</code></span>`;
    }
    box.prepend(row);
    while (box.children.length > 100) box.lastChild.remove();
  }

  async function watchLoop(ns, signal) {
    while (state.watching && !signal.aborted) {
      try {
        const data = await api(`/v1/watch/${ns}?timeout=25s`, { signal });
        if (data.timeout) {
          appendWatchEvent(null, true);
        } else if (data.action) {
          appendWatchEvent(data);
          if (state.currentNs === ns) await loadKeys(ns);
        }
      } catch (err) {
        if (signal.aborted) break;
        $("#watch-status").textContent = `Error: ${err.message}`;
        $("#watch-status").classList.remove("live");
        state.watching = false;
        $("#watch-toggle").textContent = "Start watching";
        break;
      }
    }
  }

  function startWatch() {
    const ns = $("#watch-ns").value;
    if (!ns) { toast("Select a namespace"); return; }
    stopWatch();
    state.watching = true;
    state.watchAbort = new AbortController();
    $("#watch-toggle").textContent = "Stop watching";
    $("#watch-status").textContent = `Watching ${ns}…`;
    $("#watch-status").classList.add("live");
    watchLoop(ns, state.watchAbort.signal);
  }

  function stopWatch() {
    state.watching = false;
    if (state.watchAbort) { state.watchAbort.abort(); state.watchAbort = null; }
    $("#watch-toggle").textContent = "Start watching";
    $("#watch-status").classList.remove("live");
  }

  // ── Confirm ──────────────────────────────────────────────────────
  let confirmAction = null;
  function askConfirm(msg, action) {
    confirmAction = action;
    $("#confirm-msg").textContent = msg;
    $("#confirm-modal").showModal();
  }

  // ── Navigation (path URLs survive refresh) ───────────────────────
  const PANEL_PATH = {
    secrets: "/ui/kv",
    tokens: "/ui/tokens",
    roles: "/ui/roles",
    audit: "/ui/audit",
    watch: "/ui/watch",
  };
  const PATH_PANEL = Object.fromEntries(Object.entries(PANEL_PATH).map(([k, v]) => [v, k]));

  function parseRoute() {
    const path = (location.pathname.replace(/\/$/, "") || "/") || "/";
    if (path === "/" || path === "/ui") return { panel: "secrets", kvRest: "" };
    if (path === "/ui/kv" || path.startsWith("/ui/kv/")) {
      return {
        panel: "secrets",
        kvRest: path === "/ui/kv" ? "" : path.slice("/ui/kv/".length),
      };
    }
    return { panel: PATH_PANEL[path] || "secrets", kvRest: "" };
  }

  // Match longest known namespace; leftover single segment becomes the key
  function resolveKvRest(rest, namespaces) {
    if (!rest) return { ns: null, key: null };
    const decoded = rest.split("/").map((s) => {
      try { return decodeURIComponent(s); } catch { return s; }
    }).join("/");
    const names = (namespaces || []).map((n) => n.name).sort((a, b) => b.length - a.length);
    for (const name of names) {
      if (decoded === name) return { ns: name, key: null };
      if (decoded.startsWith(name + "/")) {
        const key = decoded.slice(name.length + 1);
        if (key && !key.includes("/")) return { ns: name, key };
      }
    }
    return { ns: decoded, key: null };
  }

  function buildUrl(panel = state.panel, ns = state.currentNs, key = state.selectedKey) {
    if (panel !== "secrets") return PANEL_PATH[panel] || "/ui/kv";
    let url = "/ui/kv";
    if (ns) url += "/" + ns.split("/").map(encodeURIComponent).join("/");
    if (key) url += "/" + encodeURIComponent(key);
    return url;
  }

  function syncUrl(push = true) {
    const next = buildUrl();
    if (location.pathname === next) return;
    if (push) history.pushState({ panel: state.panel }, "", next);
    else history.replaceState({ panel: state.panel }, "", next);
  }

  function canOpenPanel(name) {
    if (name === "tokens" && !state.me?.can_manage_tokens) return false;
    if (name === "roles" && !state.me?.can_manage_roles) return false;
    if (name === "audit" && !state.me?.can_view_audit) return false;
    return true;
  }

  function showPanel(name) {
    if (name !== "watch") stopWatch();
    state.panel = name;
    $$(".nav-item").forEach((b) => b.classList.toggle("active", b.dataset.nav === name));
    $$(".panel").forEach((p) => p.classList.add("hidden"));
    $(`#panel-${name}`).classList.remove("hidden");
    updateBreadcrumbs();
  }

  async function ensureNamespacesLoaded() {
    if (state.namespaces.length) {
      populateWatchSelect();
      return;
    }
    const data = await api("/v1/namespaces");
    state.namespaces = data.namespaces || [];
    populateWatchSelect();
  }

  async function applyRoute(push = false) {
    let { panel } = parseRoute();
    if (!canOpenPanel(panel)) {
      panel = "secrets";
    }
    showPanel(panel);
    if (panel === "secrets") {
      await loadNamespaces({ preserveUrl: true });
      if (!push) syncUrl(false);
    } else {
      syncUrl(push);
      if (panel === "tokens") await loadTokens();
      if (panel === "roles") await loadRoles();
      if (panel === "audit") await loadAudit(true);
      if (panel === "watch") await ensureNamespacesLoaded();
    }
  }

  async function setPanel(name, push = true) {
    if (!canOpenPanel(name)) name = "secrets";
    if (name === "secrets") {
      showPanel("secrets");
      if (push) {
        const next = buildUrl("secrets", state.currentNs, null);
        if (location.pathname !== next) history.pushState({ panel: "secrets" }, "", next);
      }
      await loadNamespaces();
      return;
    }
    showPanel(name);
    syncUrl(push);
    if (name === "tokens") await loadTokens();
    if (name === "roles") await loadRoles();
    if (name === "audit") await loadAudit(true);
    if (name === "watch") await ensureNamespacesLoaded();
  }

  // ── Bind ─────────────────────────────────────────────────────────
  function bind() {
    $("#login-btn").addEventListener("click", async () => {
      try { await login($("#token-input").value); }
      catch (err) { showLogin(err.message || "Invalid token"); }
    });
    $("#token-input").addEventListener("keydown", (e) => {
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) $("#login-btn").click();
    });
    $("#logout-btn").addEventListener("click", logout);
    const navToggle = $("#nav-toggle");
    const navOverlay = $("#nav-overlay");
    function closeNav() {
      document.body.classList.remove("nav-open");
      if (navToggle) navToggle.setAttribute("aria-expanded", "false");
      if (navOverlay) navOverlay.hidden = true;
    }
    function openNav() {
      document.body.classList.add("nav-open");
      if (navToggle) navToggle.setAttribute("aria-expanded", "true");
      if (navOverlay) navOverlay.hidden = false;
    }
    navToggle?.addEventListener("click", () => {
      if (document.body.classList.contains("nav-open")) closeNav();
      else openNav();
    });
    navOverlay?.addEventListener("click", closeNav);
    $$(".nav-item").forEach((b) => b.addEventListener("click", () => {
      setPanel(b.dataset.nav);
      closeNav();
    }));
    window.addEventListener("resize", () => {
      if (window.innerWidth > 900) closeNav();
    });
    window.addEventListener("popstate", () => {
      if (!state.token) return;
      applyRoute(false).catch((e) => toast(e.message));
    });

    $("#refresh-secrets").addEventListener("click", () => loadNamespaces().catch((e) => toast(e.message)));
    $("#create-secret-btn").addEventListener("click", () => openSecretModal(null));
    $("#ns-search").addEventListener("input", renderNamespaceTree);
    $("#ns-ctx-delete").addEventListener("click", () => { if (ctxNs) confirmDeleteNamespace(ctxNs); });
    $("#ns-ctx-menu").addEventListener("click", (e) => e.stopPropagation());
    document.addEventListener("click", hideNsContextMenu);
    document.addEventListener("contextmenu", (e) => {
      if (e.target.closest("[data-ns-path]")) return;
      if (!$("#ns-ctx-menu").contains(e.target)) hideNsContextMenu();
    });
    document.addEventListener("keydown", (e) => {
      if (e.key !== "Escape") return;
      closeNav();
      hideNsContextMenu();
    });
    $("#key-filter").addEventListener("input", renderKeys);
    $("#secret-form").addEventListener("submit", saveSecret);

    $("#detail-back").addEventListener("click", closeDetail);
    $("#detail-edit").addEventListener("click", () => openSecretModal(state.detailEntry));
    $("#detail-delete").addEventListener("click", () => confirmDelete(state.detailEntry));
    $("#toggle-secret").addEventListener("click", () => { state.secretRevealed = !state.secretRevealed; renderDetailValue(); });
    $("#copy-value").addEventListener("click", async () => {
      await navigator.clipboard.writeText(state.detailEntry?.value || "");
      toast("Copied to clipboard");
    });
    $$(".tab").forEach((t) => t.addEventListener("click", async () => {
      if (t.dataset.tab === "current" && state.viewingVersion != null && state.selectedKey) {
        await openDetail(state.selectedKey);
        return;
      }
      switchTab(t.dataset.tab);
    }));

    $("#new-token-btn").addEventListener("click", openTokenModal);
    $("#token-form").addEventListener("submit", createToken);
    $$('input[name="mode"]').forEach((r) => r.addEventListener("change", () => {
      const scope = $("input[name='mode']:checked").value === "scope";
      $("#token-role-fields").classList.toggle("hidden", scope);
      $("#token-scope-fields").classList.toggle("hidden", !scope);
    }));
    $("#copy-token").addEventListener("click", async () => {
      await navigator.clipboard.writeText($("#token-created-value").textContent);
      toast("Copied");
    });

    $("#refresh-audit").addEventListener("click", () => loadAudit(true).catch((e) => toast(e.message)));
    $("#audit-ns-filter").addEventListener("keydown", (e) => { if (e.key === "Enter") loadAudit(true); });
    $("#audit-prev").addEventListener("click", () => {
      if (state.auditPage > 0) {
        state.auditPage--;
        loadAudit().catch((e) => toast(e.message));
      }
    });
    $("#audit-next").addEventListener("click", () => {
      state.auditPage++;
      loadAudit().catch((e) => toast(e.message));
    });

    $("#watch-toggle").addEventListener("click", () => state.watching ? stopWatch() : startWatch());
    $("#watch-clear").addEventListener("click", () => { $("#watch-events").innerHTML = ""; });

    $("#confirm-form").addEventListener("submit", async (e) => {
      e.preventDefault();
      const action = confirmAction;
      confirmAction = null;
      $("#confirm-modal").close();
      if (action) try { await action(); } catch (err) { toast(err.message); }
    });
    $$("[data-close]").forEach((b) => b.addEventListener("click", () => b.closest("dialog").close()));
  }

  async function boot() {
    bind();
    if (!state.token) { showLogin(); return; }
    try { await login(state.token); }
    catch { logout(); showLogin("Session expired — sign in again"); }
  }

  boot();
})();
