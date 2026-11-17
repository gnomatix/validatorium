package api

import "net/http"

// builderPage is the argument builder: search premises, keep chosen ones in
// a toolkit (stored in this browser only), and assemble them into an
// argument written in Argdown as docs/specs/argdown-conventions.md defines
// (statement syntax §1, premise metadata §2.2, support + and attack - §4).
// Titles are typed by the person building the argument: the specification
// asks for a 2-5 word noun phrase, and premise records carry no title.
const builderPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Argument builder</title>
<style>
 :root{--fg:#1d2433;--muted:#5b6578;--line:#d9dee8;--accent:#2f6fde;--bg:#fff}
 body{margin:0;font:15px system-ui,sans-serif;color:var(--fg);background:var(--bg)}
 main{max-width:960px;margin:0 auto;padding:16px}
 h1{font-size:20px;margin:0 0 4px} h2{font-size:16px;margin:20px 0 8px}
 p.note{color:var(--muted);margin:0 0 12px}
 input[type=text]{font:inherit;padding:6px 8px;border:1px solid var(--line);border-radius:6px}
 #q{width:min(420px,100%)} button{font:inherit;padding:5px 10px;border:1px solid var(--line);border-radius:6px;background:#f5f7fb;cursor:pointer}
 .item{border:1px solid var(--line);border-radius:8px;padding:8px 10px;margin:6px 0}
 .status{font-size:12px;color:var(--muted)} .valid{color:#13804b;font-weight:600}
 .row{display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin-top:6px}
 textarea{width:100%;min-height:200px;font:13px ui-monospace,monospace;border:1px solid var(--line);border-radius:6px;padding:8px;box-sizing:border-box}
</style></head><body><main>
<h1>Argument builder</h1>
<p class="note">Find premises, add them to your toolkit, pick the claim you want to argue for, and say which premises support or attack it.</p>
<form id="search"><input id="q" type="text" placeholder="Search premises, e.g. water" aria-label="Search premises"> <button>Search</button></form>
<div id="results"></div>
<h2>Your toolkit</h2><div id="toolkit"><p class="note">Empty. Add premises from the search results.</p></div>
<h2>Your argument (Argdown)</h2><textarea id="out" readonly aria-label="Argument in Argdown"></textarea>
</main>
<script>
const KEY = "validatorium-toolkit";
let kit = [];
try { kit = JSON.parse(localStorage.getItem(KEY) || "[]"); } catch (e) { kit = []; }
const save = () => { try { localStorage.setItem(KEY, JSON.stringify(kit)); } catch (e) {} };
const el = (tag, props, ...kids) => { const n = Object.assign(document.createElement(tag), props || {}); kids.forEach(k => n.append(k)); return n; };
const statusLabel = s => el("span", {className: "status" + (s === "valid" ? " valid" : "")}, s);

document.getElementById("search").onsubmit = async ev => {
  ev.preventDefault();
  const q = document.getElementById("q").value.trim(); if (!q) return;
  const box = document.getElementById("results"); box.textContent = "Searching…";
  const r = await fetch("/api/premises?q=" + encodeURIComponent(q)).then(r => r.json()).catch(() => ({results: []}));
  box.textContent = "";
  if (!r.results || !r.results.length) { box.append(el("p", {className: "note"}, "No premises found.")); return; }
  r.results.forEach(p => {
    const add = el("button", {type: "button"}, "Add to toolkit");
    add.onclick = () => { if (!kit.some(k => k.uuid === p.uuid)) { kit.push({uuid: p.uuid, title: "", role: "none", premise: p}); save(); render(); } };
    box.append(el("div", {className: "item"}, el("div", {}, p.statement), el("div", {className: "row"}, statusLabel(p.status), add)));
  });
};

function render() {
  const box = document.getElementById("toolkit"); box.textContent = "";
  if (!kit.length) box.append(el("p", {className: "note"}, "Empty. Add premises from the search results."));
  kit.forEach((k, i) => {
    const title = el("input", {type: "text", value: k.title, placeholder: "Title (2–5 words)", "aria-label": "Title"});
    title.oninput = () => { k.title = title.value; save(); argdown(); };
    const role = el("select", {"aria-label": "Role"});
    [["none", "not used"], ["claim", "my claim"], ["supports", "supports my claim (+)"], ["attacks", "attacks my claim (−)"]]
      .forEach(([v, t]) => role.append(el("option", {value: v, selected: k.role === v}, t)));
    role.onchange = () => { if (role.value === "claim") kit.forEach(o => { if (o.role === "claim") o.role = "none"; }); k.role = role.value; save(); render(); };
    const rm = el("button", {type: "button"}, "Remove");
    rm.onclick = () => { kit.splice(i, 1); save(); render(); };
    box.append(el("div", {className: "item"}, el("div", {}, k.premise.statement), el("div", {className: "row"}, statusLabel(k.premise.status), title, role, rm)));
  });
  argdown();
}

const titleOf = k => (k.title.trim() || "Untitled " + k.uuid.slice(0, 8)).replace(/[\[\]]/g, "");
function meta(p) {
  const support = p.support || [];
  const method = (support.find(e => e.kind === "method") || {}).reference || ((p.attestations || [])[0] || {}).method || "";
  const links = support.filter(e => e.kind !== "method").map(e => e.reference);
  return "{\n  premise_uuid: " + JSON.stringify(p.uuid) +
    ",\n  term_record_uuids: []" +
    ",\n  evidence_links: " + JSON.stringify(links) +
    ",\n  evaluation_method: " + JSON.stringify(method) + "\n}";
}
function argdown() {
  const claim = kit.find(k => k.role === "claim");
  const lines = [];
  if (claim) {
    lines.push("[" + titleOf(claim) + "]: " + claim.premise.statement + " " + meta(claim.premise));
    kit.filter(k => k.role === "supports").forEach(k => lines.push("  + [" + titleOf(k) + "]"));
    kit.filter(k => k.role === "attacks").forEach(k => lines.push("  - [" + titleOf(k) + "]"));
    kit.filter(k => k.role === "supports" || k.role === "attacks").forEach(k => lines.push("", "[" + titleOf(k) + "]: " + k.premise.statement + " " + meta(k.premise)));
  }
  document.getElementById("out").value = claim ? lines.join("\n") + "\n" : "";
}
render();
</script></body></html>`

func serveBuilder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(builderPage))
}
