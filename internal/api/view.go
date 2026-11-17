package api

import (
	_ "embed"
	"html/template"
	"net/http"
)

// cytoscapeJS is cytoscape.js 3.30.2 (MIT, The Cytoscape Consortium; see
// static/cytoscape-LICENSE), taken from the npm registry tarball whose
// SHA-512 matched the registry's published integrity value. It is served
// from this binary so the page loads nothing from other servers.
//
//go:embed static/cytoscape-3.30.2.min.js
var cytoscapeJS []byte

var viewPage = template.Must(template.New("view").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Premise {{.}}</title>
<style>
 body{margin:0;font:14px system-ui,sans-serif;color:#1d2433;background:#fff}
 header{padding:10px 16px;border-bottom:1px solid #d9dee8}
 #status{font-weight:600} #cy{position:absolute;top:84px;bottom:0;left:0;right:0}
 .legend span{display:inline-block;margin-right:12px}
 .sw{display:inline-block;width:10px;height:10px;border-radius:2px;margin-right:4px;vertical-align:middle}
</style></head><body>
<header><div id="statement">Loading…</div>
<div><span id="status"></span> <span class="legend">
 <span><i class="sw" style="background:#2f6fde"></i>data</span>
 <span><i class="sw" style="background:#16a085"></i>method</span>
 <span><i class="sw" style="background:#c0841a"></i>material</span>
 <span><i class="sw" style="background:#7a8699"></i>term</span></span></div></header>
<div id="cy"></div>
<script src="/static/cytoscape.min.js"></script>
<script>
const id = {{.}};
const colors = {data:"#2f6fde", method:"#16a085", material:"#c0841a"};
fetch("/api/premises/" + encodeURIComponent(id)).then(r => r.json()).then(p => {
  if (p.error) { document.getElementById("statement").textContent = p.error; return; }
  document.getElementById("statement").textContent = p.statement;
  document.getElementById("status").textContent = "status: " + p.status;
  const els = [{data:{id:"p", label:p.statement, kind:"premise"}}];
  const add = (list, rel) => (list || []).forEach((e, i) => {
    const n = rel + i;
    els.push({data:{id:n, label:e.kind + ": " + e.reference, kind:e.kind}});
    els.push({data:{id:n + "e", source:n, target:"p", label:rel}});
  });
  add(p.support, "supports");
  add(p.challenge, "challenges");
  (p.terms || []).forEach((t, i) => {
    els.push({data:{id:"t"+i, label:t.surfaceForm + (t.ontologyURI ? "\n" + t.ontologyURI : ""), kind:"term"}});
    els.push({data:{id:"t"+i+"e", source:"p", target:"t"+i, label:"term"}});
  });
  cytoscape({container: document.getElementById("cy"), elements: els,
    layout: {name: "breadthfirst", roots: ["p"], spacingFactor: 1.2},
    style: [
      {selector:"node", style:{label:"data(label)", "text-wrap":"wrap", "text-overflow-wrap":"anywhere", "text-max-width":"220px", "font-size":"11px", "background-color":"#7a8699", "text-valign":"bottom"}},
      {selector:"node[kind='premise']", style:{"text-overflow-wrap":"whitespace", "background-color":"#1d2433", width:40, height:40, "font-weight":"bold"}},
      {selector:"node[kind='data']", style:{"background-color":colors.data}},
      {selector:"node[kind='method']", style:{"background-color":colors.method}},
      {selector:"node[kind='material']", style:{"background-color":colors.material}},
      {selector:"edge", style:{label:"data(label)", "font-size":"10px", "curve-style":"bezier", "target-arrow-shape":"triangle", width:1.5}}
    ]});
});
</script></body></html>`))

func (s *Server) viewPremise(w http.ResponseWriter, r *http.Request, uuid string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = viewPage.Execute(w, uuid)
}

func serveCytoscape(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(cytoscapeJS)
}
