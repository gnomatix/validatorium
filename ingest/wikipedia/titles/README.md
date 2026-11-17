# Title lists

`elements.txt`: English Wikipedia article titles of every Wikidata item that
is an instance of chemical element (`wdt:P31 wd:Q11344`), one per line, for
`wikiextract -titles`. `elements.wikidata.tsv` keeps the Wikidata item for
each title. Retrieved 2026-11-17 from <https://query.wikidata.org/sparql>:

```sparql
SELECT ?item ?title WHERE {
  ?item wdt:P31 wd:Q11344 .
  ?a schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> ; schema:name ?title
} ORDER BY ?title
```

164 titles: the 118 named elements plus 46 systematic names of
undiscovered elements (Unbinilium and so on), as Wikidata classes them.

`countries.txt`: English Wikipedia article titles of every Wikidata item
that is an instance of sovereign state (`wdt:P31 wd:Q3624078`) and has no
dissolved date (`wdt:P576`). `countries.wikidata.tsv` keeps the item for each
title. Retrieved 2026-11-17 from the same endpoint:

```sparql
SELECT DISTINCT ?item ?title WHERE {
  ?item wdt:P31 wd:Q3624078 .
  FILTER NOT EXISTS { ?item wdt:P576 ?end }
  ?a schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> ; schema:name ?title
} ORDER BY ?title
```

197 titles. Membership follows Wikidata's classification as retrieved; it
includes states with limited recognition that Wikidata classes this way
(for example Kosovo, Palestine, Taiwan).
