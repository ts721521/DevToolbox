const $ = (id) => document.getElementById(id);

async function api(path, opts = {}) {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  const text = await res.text();
  if (!res.ok) throw new Error(text || res.statusText);
  return text ? JSON.parse(text) : {};
}

function platformBadges(t) {
  const all = ["macOS", "Windows"];
  const have = new Set(t.platform_labels || []);
  return all
    .map((name) => `<span class="os ${have.has(name) ? "on" : "off"}">${name}</span>`)
    .join("");
}

function render(tools) {
  const q = $("q").value.trim().toLowerCase();
  const filtered = tools.filter((t) => {
    const blob = [t.name, t.id, t.group, t.description, t.url, ...(t.platform_labels || [])]
      .join(" ")
      .toLowerCase();
    return !q || blob.includes(q);
  });
  $("empty").classList.toggle("hidden", filtered.length > 0);
  $("list").innerHTML = filtered
    .map((t) => {
      const canOpen = t.compatible !== false;
      const stopBtn = t.kind === "url" ? "" : `<button class="btn danger" data-stop="${escapeHtml(t.id)}">关闭</button>`;
      const openBtn = canOpen
        ? `<button class="btn primary" data-open="${escapeHtml(t.id)}">打开</button>`
        : `<button class="btn" disabled>当前系统不可用</button>`;
      const pids = t.pids && t.pids.length ? ` · PID ${t.pids.join(",")}` : "";
      return `
    <article class="card ${t.compatible ? "" : "incompat"}">
      <div class="row">
        <h3>${escapeHtml(t.name)}</h3>
        <span><span class="dot ${t.running ? "on" : "off"}"></span>${t.running ? "运行中" : "未启动"}${pids}</span>
      </div>
      <div class="osrow">${platformBadges(t)}</div>
      <div class="meta">${escapeHtml(t.group || t.kind)} · ${escapeHtml(t.id)}
        ${t.url ? `<br>${escapeHtml(t.url)}` : ""}
        ${t.description ? `<br>${escapeHtml(t.description)}` : ""}
      </div>
      <div class="actions">
        ${openBtn}
        ${stopBtn}
        <button class="btn" data-del="${escapeHtml(t.id)}">移除</button>
      </div>
    </article>`;
    })
    .join("");
}

function escapeHtml(s) {
  return String(s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/"/g, "&quot;");
}

async function loadSystem() {
  try {
    const s = await api("/api/system");
    $("sys").innerHTML = `
      <span>本机 <strong>${escapeHtml(s.os_name)} ${escapeHtml(s.arch)}</strong></span>
      <span>工具箱支援 <strong>macOS · Windows</strong></span>
      <span>配置 ${escapeHtml(s.tools_dir)}</span>`;
    $("sysline").textContent = `本机 ${s.os_name} ${s.arch} · 支援 macOS / Windows · 打开后可一键关闭后台`;
  } catch (_) {
    $("sys").innerHTML = `<span>工具箱支援 <strong>macOS · Windows</strong></span>`;
  }
}

async function refresh() {
  try {
    const tools = await api("/api/tools");
    render(tools);
  } catch (err) {
    $("list").innerHTML = `<p class="empty">${escapeHtml(err.message)}</p>`;
  }
}

$("q").addEventListener("input", refresh);
$("btn-add").addEventListener("click", () => $("form").classList.remove("hidden"));
$("btn-cancel").addEventListener("click", () => $("form").classList.add("hidden"));

$("btn-save").addEventListener("click", async () => {
  const exec = $("f-exec").value.trim();
  const platforms = [];
  if ($("f-mac").checked) platforms.push("darwin");
  if ($("f-win").checked) platforms.push("windows");
  const body = {
    id: $("f-id").value.trim(),
    name: $("f-name").value.trim(),
    kind: $("f-kind").value,
    group: $("f-group").value.trim(),
    workdir: $("f-workdir").value.trim(),
    command: exec ? exec.split(/\s+/) : [],
    url: $("f-url").value.trim(),
    health_url: $("f-url").value.trim(),
    health_contains: $("f-health").value.trim(),
    app_path: $("f-app").value.trim(),
    terminal: $("f-term").checked,
    platforms,
  };
  try {
    await api("/api/tools", { method: "POST", body: JSON.stringify(body) });
    $("form").classList.add("hidden");
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

$("list").addEventListener("click", async (e) => {
  const open = e.target.dataset.open;
  const stop = e.target.dataset.stop;
  const del = e.target.dataset.del;
  try {
    if (open) await api(`/api/tools/${encodeURIComponent(open)}/launch`, { method: "POST" });
    if (stop) {
      e.target.disabled = true;
      await api(`/api/tools/${encodeURIComponent(stop)}/stop`, { method: "POST" });
      await refresh();
    }
    if (del && confirm("从工具箱移除这个快捷方式？不会删除工具本身。")) {
      await api(`/api/tools/${encodeURIComponent(del)}`, { method: "DELETE" });
      await refresh();
    }
  } catch (err) {
    alert(err.message);
  }
});

loadSystem();
refresh();
setInterval(refresh, 4000);
