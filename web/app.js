const $ = (id) => document.getElementById(id);

const ALL = "全部";
const TRASH = "垃圾桶";
const TAB_KEY = "tooldock.tab";

let allTools = [];
let trash = [];
let tabs = [];
let currentTab = localStorage.getItem(TAB_KEY) || ALL;
let selectMode = false;
const selected = new Set();
let lastSnap = "";
let lastLayout = "";
let listEnter = true;

let csrf = "";

async function api(path, opts = {}) {
  const headers = { "Content-Type": "application/json", ...(opts.headers || {}) };
  if (csrf) headers["X-ToolDock-Token"] = csrf;
  const res = await fetch(path, { ...opts, headers });
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

function escapeHtml(s) {
  return String(s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/"/g, "&quot;");
}

function visibleTools() {
  if (currentTab === TRASH) return [];
  const q = $("q").value.trim().toLowerCase();
  return allTools.filter((t) => {
    if (currentTab !== ALL && (t.group || "") !== currentTab) return false;
    const blob = [t.name, t.id, t.group, t.description, t.url, t.git, t.git_label, t.git_web, ...(t.platform_labels || [])]
      .join(" ")
      .toLowerCase();
    return !q || blob.includes(q);
  });
}

function renderTabs() {
  const counts = { [ALL]: allTools.length };
  for (const name of tabs) counts[name] = 0;
  for (const t of allTools) {
    const g = t.group || "其他";
    counts[g] = (counts[g] || 0) + 1;
  }
  const names = [ALL, ...tabs, TRASH];
  $("tabbar").innerHTML =
    names
      .map((name) => {
        const n = name === TRASH ? trash.length : counts[name] || 0;
        const active = name === currentTab ? "active" : "";
        const del =
          name !== ALL && name !== "其他" && name !== TRASH
            ? `<span class="tab-x" data-tab-del="${escapeHtml(name)}" title="删除标签">×</span>`
            : "";
        return `<button type="button" class="tab ${active}" draggable="false" data-tab="${escapeHtml(
          name
        )}">${escapeHtml(name)} ${n}${del}</button>`;
      })
      .join("") + `<button type="button" class="tab tab-add" id="btn-tab-add">+ 新标签</button>`;

  const dl = $("tab-list");
  dl.innerHTML = tabs.map((n) => `<option value="${escapeHtml(n)}"></option>`).join("");
  const sel = $("batch-target");
  sel.innerHTML = tabs.map((n) => `<option value="${escapeHtml(n)}">${escapeHtml(n)}</option>`).join("");
}

function renderTrash() {
  const q = $("q").value.trim().toLowerCase();
  const filtered = trash.filter((t) => {
    const blob = [t.name, t.id, t.workdir, t.git, t.git_label].join(" ").toLowerCase();
    return !q || blob.includes(q);
  });
  $("empty").textContent = "垃圾桶是空的。移除的工具会停在这里，扫描和 AI 都不会再注册进来。";
  $("empty").classList.toggle("hidden", filtered.length > 0);
  $("list").innerHTML = filtered
    .map((t, i) => {
      const gitBtn = t.git
        ? `<span class="id">${escapeHtml(t.git_label || t.git)}</span>`
        : "";
      return `
    <article class="card" data-id="${escapeHtml(t.id)}" style="--i:${i}">
      <div class="row">
        <h3>${escapeHtml(t.name || t.id)}</h3>
        <span class="status">已屏蔽</span>
      </div>
      <div class="meta">
        <div class="id">${escapeHtml(t.id)}</div>
        ${gitBtn}
        ${t.workdir ? `<div>${escapeHtml(t.workdir)}</div>` : ""}
      </div>
      <div class="actions">
        <button class="btn primary" data-restore="${escapeHtml(t.id)}">恢复</button>
        <button class="btn ghost" data-unblock="${escapeHtml(t.id)}">允许再注册</button>
      </div>
    </article>`;
    })
    .join("");
}

function render() {
  if (currentTab === TRASH) {
    renderTrash();
    return;
  }
  const filtered = visibleTools();
  $("empty").textContent = "还没有工具。发给 AI，或点注册。";
  $("empty").classList.toggle("hidden", filtered.length > 0);
  $("list").innerHTML = filtered
    .map((t, i) => {
      const canOpen = t.compatible !== false;
      const stopBtn =
        t.kind !== "url" || (t.services && t.services.length)
          ? `<button class="btn danger" data-stop="${escapeHtml(t.id)}">关闭</button>`
          : "";
      const extra = [];
      if (t.depends && t.depends.length) extra.push("依赖 " + t.depends.length);
      if (t.services && t.services.length) extra.push("服务 " + t.services.length);
      const extraLine = extra.length ? `<br>${escapeHtml(extra.join(" · "))}` : "";
      const openBtn = canOpen
        ? `<button class="btn primary" data-open="${escapeHtml(t.id)}">打开</button>`
        : `<button class="btn" disabled>不可用</button>`;
      const checked = selected.has(t.id) ? "checked" : "";
      const selCls = selected.has(t.id) ? "selected" : "";
      const liveCls = t.running ? "live" : "";
      const dirBtn = t.workdir
        ? `<button class="btn" data-dir="${escapeHtml(t.id)}">目录</button>`
        : "";
      const appBtn = t.app_path
        ? `<button class="btn" data-app="${escapeHtml(t.id)}">程序</button>`
        : "";
      const gitBtn = t.git
        ? `<button class="repo" data-git="${escapeHtml(t.id)}" title="${escapeHtml(t.git)}">${escapeHtml(t.git_label || t.git)}</button>`
        : "";
      return `
    <article class="card ${t.compatible ? "" : "incompat"} ${selCls} ${liveCls}" draggable="true" data-id="${escapeHtml(t.id)}" style="--i:${i}">
      <label class="pick"><input type="checkbox" data-pick="${escapeHtml(t.id)}" ${checked} />选择</label>
      <div class="row">
        <h3>${escapeHtml(t.name)}</h3>
        <span class="status ${t.running ? "on" : ""}">${t.running ? "运行" : "待机"}</span>
      </div>
      <div class="osrow">${platformBadges(t)}</div>
      <div class="meta">
        <div class="id">${escapeHtml(t.id)}</div>
        ${gitBtn}
        ${t.description ? `<div>${escapeHtml(t.description)}</div>` : ""}
        ${extraLine}
      </div>
      <div class="actions">
        ${openBtn}
        ${stopBtn}
        ${dirBtn}
        ${appBtn}
        <button class="btn ghost" data-move="${escapeHtml(t.id)}">移到</button>
        <button class="btn ghost" data-del="${escapeHtml(t.id)}">移除</button>
      </div>
    </article>`;
    })
    .join("");
  $("sel-count").textContent = `已选 ${selected.size}`;
  const list = $("list");
  if (listEnter) {
    list.classList.add("enter");
    listEnter = false;
    const ms = 450 + Math.max(filtered.length, 1) * 35;
    window.setTimeout(() => list.classList.remove("enter"), ms);
  } else {
    list.classList.remove("enter");
  }
}

async function loadSystem() {
  try {
    const s = await api("/api/system");
    const ver = s.version || "";
    csrf = s.token || csrf;
    $("ver").textContent = ver;
    document.title = ver ? `工坞 ${ver}` : "工坞";
    $("sys").innerHTML = `
      <span>VERSION <strong>${escapeHtml(ver)}</strong></span>
      &nbsp;·&nbsp;
      <span>HOST <strong>${escapeHtml(s.os_name)} ${escapeHtml(s.arch)}</strong></span>
      &nbsp;·&nbsp;
      <span>macOS / Windows</span>`;
    $("sysline").textContent = ver
      ? `${ver} · ${s.os_name} ${s.arch}`
      : `${s.os_name} ${s.arch} · 本机停靠`;
  } catch (_) {
    $("sys").innerHTML = `<span>macOS / Windows</span>`;
  }
}

function snapshot(tools, tabNames, withRunning) {
  return JSON.stringify({
    tabs: tabNames,
    tools: (tools || []).map((t) => ({
      id: t.id,
      name: t.name,
      kind: t.kind,
      group: t.group,
      description: t.description,
      url: t.url,
      git: t.git,
      git_label: t.git_label,
      running: withRunning ? !!t.running : false,
      compatible: t.compatible,
      workdir: t.workdir,
      app_path: t.app_path,
      depends: t.depends,
      services: t.services,
      platform_labels: t.platform_labels,
    })),
    trash: (trash || []).map((t) => ({ id: t.id, name: t.name })),
  });
}

function patchRunning(tools) {
  for (const t of tools) {
    const card = document.querySelector(`article.card[data-id="${CSS.escape(t.id)}"]`);
    if (!card) continue;
    card.classList.toggle("live", !!t.running);
    const st = card.querySelector(".status");
    if (st) {
      st.classList.toggle("on", !!t.running);
      st.textContent = t.running ? "运行" : "待机";
    }
  }
  return true;
}

async function refresh() {
  try {
    const [tools, tabData, blocked] = await Promise.all([
      api("/api/tools"),
      api("/api/tabs"),
      api("/api/blocked"),
    ]);
    const nextTabs = tabData.tabs || [];
    trash = blocked.entries || [];
    const snap = snapshot(tools, nextTabs, true);
    if (snap === lastSnap) return;
    const layout = snapshot(tools, nextTabs, false);
    const onlyStatus = lastLayout !== "" && layout === lastLayout;
    allTools = tools;
    tabs = nextTabs;
    lastSnap = snap;
    lastLayout = layout;
    if (currentTab !== ALL && currentTab !== TRASH && !tabs.includes(currentTab)) {
      currentTab = ALL;
      localStorage.setItem(TAB_KEY, currentTab);
    }
    if (onlyStatus && patchRunning(tools)) return;
    renderTabs();
    render();
  } catch (err) {
    lastSnap = "";
    lastLayout = "";
    listEnter = true;
    $("list").innerHTML = `<p class="empty">${escapeHtml(err.message)}</p>`;
  }
}

function setSelectMode(on) {
  selectMode = on;
  document.body.classList.toggle("selecting", on);
  $("batchbar").classList.toggle("hidden", !on);
  $("btn-select").textContent = on ? "完成" : "选择";
  if (!on) selected.clear();
  render();
}

async function buryIds(ids) {
  if (!ids.length) return;
  for (const id of ids) {
    await api(`/api/tools/${encodeURIComponent(id)}`, { method: "DELETE" });
    selected.delete(id);
  }
  await refresh();
}

async function moveIds(ids, group) {
  if (!ids.length || !group) return;
  await api("/api/tools/move", { method: "POST", body: JSON.stringify({ ids, group }) });
  ids.forEach((id) => selected.delete(id));
  await refresh();
}

$("q").addEventListener("input", render);
$("btn-add").addEventListener("click", () => $("form").classList.remove("hidden"));
$("btn-cancel").addEventListener("click", () => $("form").classList.add("hidden"));
$("btn-select").addEventListener("click", () => setSelectMode(!selectMode));

$("btn-save").addEventListener("click", async () => {
  const exec = $("f-exec").value.trim();
  const platforms = [];
  if ($("f-mac").checked) platforms.push("darwin");
  if ($("f-win").checked) platforms.push("windows");
  const depends = $("f-depends")
    .value.split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean);
  const services = $("f-services")
    .value.split(/\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => ({ command: line.split(/\s+/), name: line.split(/\s+/)[0] }));
  const body = {
    id: $("f-id").value.trim(),
    name: $("f-name").value.trim(),
    kind: $("f-kind").value,
    group: $("f-group").value.trim() || "其他",
    workdir: $("f-workdir").value.trim(),
    command: exec ? exec.split(/\s+/) : [],
    url: $("f-url").value.trim(),
    health_url: $("f-url").value.trim(),
    health_contains: $("f-health").value.trim(),
    app_path: $("f-app").value.trim(),
    terminal: $("f-term").checked,
    platforms,
    depends,
    services,
    git: $("f-git").value.trim(),
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
  const pick = e.target.dataset.pick;
  const open = e.target.dataset.open;
  const stop = e.target.dataset.stop;
  const dir = e.target.dataset.dir;
  const app = e.target.dataset.app;
  const git = e.target.dataset.git;
  const del = e.target.dataset.del;
  const move = e.target.dataset.move;
  try {
    if (pick) {
      if (e.target.checked) selected.add(pick);
      else selected.delete(pick);
      $("sel-count").textContent = `已选 ${selected.size}`;
      e.target.closest(".card")?.classList.toggle("selected", e.target.checked);
      return;
    }
    if (open) {
      await api(`/api/tools/${encodeURIComponent(open)}/launch`, { method: "POST" });
      await refresh();
    }
    if (dir) await api(`/api/tools/${encodeURIComponent(dir)}/dir`, { method: "POST" });
    if (app) await api(`/api/tools/${encodeURIComponent(app)}/app`, { method: "POST" });
    if (git) await api(`/api/tools/${encodeURIComponent(git)}/git`, { method: "POST" });
    if (stop) {
      e.target.disabled = true;
      try {
        await api(`/api/tools/${encodeURIComponent(stop)}/stop`, { method: "POST" });
      } finally {
        e.target.disabled = false;
        await refresh();
      }
    }
    if (move) {
      const dest = prompt(`把「${move}」移到哪个标签？\n${tabs.join(" / ")}`, currentTab === ALL ? tabs[0] : currentTab);
      if (dest) await moveIds([move], dest.trim());
    }
    if (del && confirm("移到垃圾桶？以后扫描和 AI 都不会再把这个工具注册进来。")) {
      await api(`/api/tools/${encodeURIComponent(del)}`, { method: "DELETE" });
      selected.delete(del);
      await refresh();
    }
    const restore = e.target.dataset.restore;
    const unblock = e.target.dataset.unblock;
    if (restore) {
      await api(`/api/tools/${encodeURIComponent(restore)}/restore`, { method: "POST" });
      await refresh();
    }
    if (unblock && confirm("取消屏蔽后，以后可以再把这个项目注册进来。")) {
      await api(`/api/blocked/${encodeURIComponent(unblock)}`, { method: "DELETE" });
      await refresh();
    }
  } catch (err) {
    alert(err.message);
  }
});

$("list").addEventListener("dragstart", (e) => {
  const card = e.target.closest("[data-id]");
  if (!card) return;
  e.dataTransfer.setData("text/plain", card.dataset.id);
  e.dataTransfer.effectAllowed = "move";
  card.classList.add("dragging");
});
$("list").addEventListener("dragend", (e) => {
  e.target.closest("[data-id]")?.classList.remove("dragging");
});

$("tabbar").addEventListener("click", async (e) => {
  const del = e.target.dataset.tabDel;
  if (del) {
    e.stopPropagation();
    if (!confirm(`删除标签「${del}」？里面的工具会移到「其他」。`)) return;
    try {
      await api(`/api/tabs/${encodeURIComponent(del)}?move=${encodeURIComponent("其他")}`, { method: "DELETE" });
      if (currentTab === del) currentTab = ALL;
      await refresh();
    } catch (err) {
      alert(err.message);
    }
    return;
  }
  if (e.target.id === "btn-tab-add") {
    const name = prompt("新标签名称（尽量用已有的工作/财务/开发/其他）");
    if (!name) return;
    try {
      const data = await api("/api/tabs", { method: "POST", body: JSON.stringify({ name: name.trim() }) });
      currentTab = data.name || name.trim();
      localStorage.setItem(TAB_KEY, currentTab);
      await refresh();
    } catch (err) {
      alert(err.message);
    }
    return;
  }
  const tab = e.target.closest("[data-tab]");
  if (!tab) return;
  const name = tab.dataset.tab;
  if (selectMode && selected.size && name !== ALL) {
    try {
      if (name === TRASH) {
        await buryIds([...selected]);
        return;
      }
      await moveIds([...selected], name);
      return;
    } catch (err) {
      alert(err.message);
      return;
    }
  }
  currentTab = name;
  localStorage.setItem(TAB_KEY, currentTab);
  renderTabs();
  render();
});

$("tabbar").addEventListener("dblclick", async (e) => {
  const tab = e.target.closest("[data-tab]");
  if (!tab) return;
  const from = tab.dataset.tab;
  if (from === ALL || from === "其他" || from === TRASH) return;
  const to = prompt("重命名标签", from);
  if (!to || to.trim() === from) return;
  try {
    const data = await api("/api/tabs/rename", { method: "POST", body: JSON.stringify({ from, to: to.trim() }) });
    currentTab = data.to || to.trim();
    localStorage.setItem(TAB_KEY, currentTab);
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

$("tabbar").addEventListener("dragover", (e) => {
  const tab = e.target.closest("[data-tab]");
  if (!tab || tab.dataset.tab === ALL) return;
  e.preventDefault();
  tab.classList.add("drop");
});
$("tabbar").addEventListener("dragleave", (e) => {
  e.target.closest("[data-tab]")?.classList.remove("drop");
});
$("tabbar").addEventListener("drop", async (e) => {
  const tab = e.target.closest("[data-tab]");
  if (!tab || tab.dataset.tab === ALL) return;
  e.preventDefault();
  tab.classList.remove("drop");
  const id = e.dataTransfer.getData("text/plain");
  const ids = selected.size && selected.has(id) ? [...selected] : id ? [id] : [];
  try {
    if (tab.dataset.tab === TRASH) {
      await buryIds(ids);
      return;
    }
    await moveIds(ids, tab.dataset.tab);
  } catch (err) {
    alert(err.message);
  }
});

$("btn-batch-move").addEventListener("click", async () => {
  try {
    await moveIds([...selected], $("batch-target").value);
  } catch (err) {
    alert(err.message);
  }
});
$("btn-select-all").addEventListener("click", () => {
  visibleTools().forEach((t) => selected.add(t.id));
  render();
});
$("btn-select-clear").addEventListener("click", () => {
  selected.clear();
  render();
});

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (_) {
    const el = document.createElement("textarea");
    el.value = text;
    document.body.appendChild(el);
    el.select();
    const ok = document.execCommand("copy");
    el.remove();
    return ok;
  }
}

function showCopied() {
  $("ai-copied").classList.remove("hidden");
  setTimeout(() => $("ai-copied").classList.add("hidden"), 2500);
}

async function openAIModal() {
  const data = await api("/api/ai-handoff");
  $("ai-url").value = data.url;
  $("ai-prompt").value = data.prompt;
  $("ai-modal").classList.remove("hidden");
  if (await copyText(data.prompt)) showCopied();
}

$("btn-ai").addEventListener("click", async () => {
  try {
    await openAIModal();
  } catch (err) {
    alert(err.message);
  }
});
$("btn-ai-close").addEventListener("click", () => $("ai-modal").classList.add("hidden"));
$("ai-modal").addEventListener("click", (e) => {
  if (e.target === $("ai-modal")) $("ai-modal").classList.add("hidden");
});
$("btn-copy-prompt").addEventListener("click", async () => {
  if (await copyText($("ai-prompt").value)) showCopied();
});
$("btn-copy-url").addEventListener("click", async () => {
  if (await copyText($("ai-url").value)) showCopied();
});
$("btn-open-ai").addEventListener("click", () => {
  window.open($("ai-url").value, "_blank");
});
$("ai-url").addEventListener("focus", (e) => e.target.select());
$("ai-prompt").addEventListener("focus", (e) => e.target.select());

async function loadLogs() {
  const data = await api("/api/logs");
  $("log-path").textContent = data.path || "";
  $("log-text").textContent = data.text || "(还没有日志)";
  $("log-text").scrollTop = $("log-text").scrollHeight;
}

async function openLogModal() {
  $("log-modal").classList.remove("hidden");
  await loadLogs();
}

$("btn-logs").addEventListener("click", async () => {
  try {
    await openLogModal();
  } catch (err) {
    alert(err.message);
  }
});
$("btn-log-close").addEventListener("click", () => $("log-modal").classList.add("hidden"));
$("log-modal").addEventListener("click", (e) => {
  if (e.target === $("log-modal")) $("log-modal").classList.add("hidden");
});
$("btn-log-refresh").addEventListener("click", async () => {
  try {
    await loadLogs();
  } catch (err) {
    alert(err.message);
  }
});
$("btn-log-copy").addEventListener("click", async () => {
  if (await copyText($("log-text").textContent)) {
    $("log-copied").classList.remove("hidden");
    setTimeout(() => $("log-copied").classList.add("hidden"), 2000);
  }
});
$("btn-log-folder").addEventListener("click", async () => {
  try {
    await api("/api/logs/open", { method: "POST", body: "{}" });
  } catch (err) {
    alert(err.message);
  }
});

loadSystem().finally(() => {
  refresh();
  setInterval(() => {
    if (document.hidden) return;
    if (document.querySelector(".card.dragging")) return;
    refresh();
  }, 8000);
});
document.addEventListener("visibilitychange", () => {
  if (!document.hidden) refresh();
});
