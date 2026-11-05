# Canonical encoding for cross-language CIDs: DAG-CBOR, RFC 8785 JCS, DAG-JSON, Go struct JSON (design revisit item 1)

**Status: UNVERIFIED.** Extracted on 2026-11-05 from a deep-research run that was stopped
during its fetch phase, before adversarial verification and synthesis. Each line is a
claim as extracted from one fetched source by a reader agent; none has been checked
against a second source. Treat as leads to confirm, not as findings.

Fetched sources with extracted claims: 0; claims: 65

## Claims

- JCS canonicalisation sorts object property names by comparing them as arrays of UTF-16 code units treated as unsigned integers, independent of locale, so any language that can reproduce UTF-16 code-unit ordering (not byte-wise UTF-8 or locale collation) will produce the same key order.  
  Fetch label: fetch:rfc-editor.org
- JCS requires JSON numbers to be serialised using the ECMAScript (ECMA-262 Section 7.1.12.1, with Note 2) IEEE 754 double-precision number-to-string algorithm, and a compliant implementation must terminate with an error on NaN or Infinity; integers outside the IEEE 754 safe-integer range (-9007199254740991 to 9007199254740991) are not reliably representable and should be carried as JSON strings.  
  Fetch label: fetch:rfc-editor.org
- JCS does not apply Unicode normalisation (e.g. NFC/NFD) to strings; two strings that are canonically equivalent in Unicode but differ in code points will canonicalise to different bytes, and lone surrogates must cause the implementation to abort with an error.  
  Fetch label: fetch:rfc-editor.org
- JCS input must conform to I-JSON (RFC 7493), which means JSON objects must not contain duplicate property names and numbers must be expressible as IEEE 754 doubles; applications are required to verify I-JSON conformance and abort if any step fails.  
  Fetch label: fetch:rfc-editor.org
- RFC 8785 is an Informational (not Standards Track) RFC, and its Appendix G lists verified-compatible open-source implementations in JavaScript, Java, Go, .NET/C#, and Python, indicating cross-language library availability at publication time (2020).  
  Fetch label: fetch:rfc-editor.org
- JCS (the scheme implemented by this repository) is specified in RFC 8785 and serialises JSON primitives using rules compatible with ECMAScript's JSON.stringify(), so a non-Go writer that follows ES6 number/string serialisation can reproduce the canonical bytes.  
  Fetch label: fetch:github.com
- JCS defines a deterministic key order by lexicographically sorting JSON Object properties recursively, while leaving JSON Array element order unchanged.  
  Fetch label: fetch:github.com
- JCS restricts the canonicalisable input to the I-JSON subset (RFC 7493), which implies the usual I-JSON constraints on numbers (IEEE-754 double range) and strings (valid Unicode), so records outside that subset cannot be canonicalised by JCS.  
  Fetch label: fetch:github.com
- The project provides or links JCS implementations in at least nine languages: Rust, JavaScript, Java, Go, .NET/C#, Python, Elixir, Ruby and PHP, with Go, .NET and Python hosted in the repository itself, indicating broad cross-language availability for byte-identical encoding.  
  Fetch label: fetch:github.com
- The repository ships a shared `testdata` directory and an online browser test page, giving cross-language implementers conformance vectors to verify byte-identical output.  
  Fetch label: fetch:github.com
- RFC 8785 (JCS) section 3.2.2.3 requires numbers to be parsed as IEEE-754 doubles, so an integer literal such as 2^53+1 is rounded to 2^53 before serialization; a strict JCS canonicalizer therefore cannot preserve integers above 2^53 and two distinct inputs can canonicalize to identical bytes (and thus identical hashes/CIDs).  
  Fetch label: fetch:pkg.go.dev
- Truestamp's jcs package deliberately deviates from RFC 8785 by emitting integer literals exactly as parsed (arbitrary precision) rather than rounding through a double, which means documents containing such integers cannot be verified by a strict RFC 8785 implementation; Canonicalize therefore returns the offending literals as a portability report.  
  Fetch label: fetch:pkg.go.dev
- RFC 8785 Appendix B recommends (SHOULD) that producers keep integers within JavaScript's Number.MAX_SAFE_INTEGER (2^53 - 1); the package encodes this as MaxSafeInteger for producers and a separate MaxExactInteger (2^53) for verifiers, with the one-value gap kept intentional and guarded by a test (TestThresholdsDifferByOne).  
  Fetch label: fetch:pkg.go.dev
- For inputs without oversized integers, the package delegates key ordering (sorted by UTF-16 code units), string escaping, ES6 float formatting, and negative-zero handling to the Go library github.com/gowebpki/jcs, and its output is byte-identical to that library.  
  Fetch label: fetch:pkg.go.dev
- The gowebpki/jcs Go implementation silently rewrites some malformed JSON documents into different valid ones, so Truestamp adds a strict RFC 8259 well-formedness check before canonicalizing; also, in Go the integer literal is lost unless the input is decoded with json.Decoder.UseNumber, and large values written as floats (e.g. 1e21) are never flagged as oversized integers.  
  Fetch label: fetch:pkg.go.dev
- gowebpki/jcs is a Go library implementing RFC 8785 JSON Canonicalization Scheme, exposing a single Transform([]byte) ([]byte, error) API that re-serializes arbitrary UTF-8 JSON into canonical form; it constrains input to the I-JSON (RFC 7493) subset and serializes primitives compatibly with ECMAScript JSON.stringify().  
  Fetch label: fetch:github.com
- The library sorts object keys by UTF-16 code units (converting each key to UTF-16 and comparing element by element, shorter prefix first) and rejects duplicate keys with a 'Duplicate key' error, so documents with duplicate keys cannot be canonicalized.  
  Fetch label: fetch:github.com
- All numbers are parsed with strconv.ParseFloat at 64 bits and re-serialized via ES6 Number-to-string rules (shortest round-trip repr, fixed notation for magnitudes in [1e-6, 1e21), exponential otherwise, exponent leading zero stripped, -0 emitted as 0); NaN/Infinity return an error. Consequently integers beyond 2^53 lose precision and out-of-range values error ('Number out of range').  
  Fetch label: fetch:github.com
- String output escapes only ", \, backspace, form feed, newline, carriage return, tab with short escapes, other controls below 0x20 as lowercase \u00xx, and writes all other characters including non-ASCII as raw UTF-8 with no Unicode normalization (NFC/NFD) applied; canonical output must be UTF-8 for cross-platform byte identity.  
  Fetch label: fetch:github.com
- The project has a small issue history (3 closed issues) including conformance bugs: issue #5 (closed Sep 2026) reported that reversed surrogate pairs became U+FFFD and invalid UTF-8 passed through, violating MUST requirements in RFC 8785 sections 3.2.2.2 and 3.2.4; issue #2 reported that bare JSON null could not be transformed (closed Oct 2023). The repo is a maintained fork of cyberphone/json-canonicalization (Anders Rundgren), requires go 1.21, and has ~29 stars.  
  Fetch label: fetch:github.com
- The Deno standard library package @std/json (version 1.1.0) ships a canonicalize() function that implements RFC 8785 JCS, producing a deterministic JSON string intended for hashing or signing, with the signature canonicalize(value: JsonValue): string.  
  Fetch label: fetch:jsr.io
- @std/json canonicalize() sorts object keys lexicographically by UTF-16 code units (so digits sort before uppercase before lowercase), and emits no whitespace between tokens; e.g. { a: 1, A: 2, "1": 3 } serializes to {"1":3,"A":2,"a":1}.  
  Fetch label: fetch:jsr.io
- Number serialization in @std/json canonicalize() follows ECMAScript Number-to-string rules per RFC 8785 Section 3.2.2.3, so 10.0 becomes "10", 1e21 becomes "1e+21", 0.0000001 becomes "1e-7", and -0 becomes "0" (negative zero is not preserved).  
  Fetch label: fetch:jsr.io
- canonicalize() throws a TypeError for non-finite numbers (Infinity, -Infinity, NaN) and for circular references, rather than emitting null or a placeholder as JSON.stringify would.  
  Fetch label: fetch:jsr.io
- The function requires JSON-compatible input and does not invoke toJSON(); values such as Date must be converted to their JSON representation by the caller first, and the documentation does not specify BigInt handling or string escaping rules.  
  Fetch label: fetch:jsr.io
- dsnet/golib/jsoncs is a Go package that implements RFC 8785 JCS, giving Go writers an alternative to the Go module in the RFC's reference implementations; it is licensed BSD-3-Clause and the indexed version v1.0.1 was published 20 January 2021.  
  Fetch label: fetch:pkg.go.dev
- The package's canonicalizer requires its input to already conform to RFC 7493 (I-JSON), meaning JCS canonical output is only defined for I-JSON-compliant documents (e.g. no NaN/Infinity, numbers within IEEE-754 double range) — a constraint any cross-language writer must respect to get byte-identical output.  
  Fetch label: fetch:pkg.go.dev
- Correct JCS number serialization depends on the host language's shortest-round-trip float formatter: on Go 1.12 and earlier, strconv float formatting had a rounding bug (golang.org/issue/29491) that makes jsoncs produce non-canonical (and therefore non-byte-identical) output in rare cases.  
  Fetch label: fetch:pkg.go.dev
- The package derives ES6 Number::toString output (RFC 8785 section 3.2.2.3) from Go's strconv.FormatFloat 'e' format rather than a native implementation, because Go's standard library does not expose the n/k/s decomposition and its output format is not guaranteed stable — a fragility point for determinism across Go versions.  
  Fetch label: fetch:pkg.go.dev
- The float formatter follows ECMA-262 6th edition section 7.1.12.1 exactly (fixed notation for 0 < n <= 21 and -6 < n <= 0, exponent notation otherwise, -0 rendered as "0"), but deviates from RFC 8785 by also emitting "NaN" and "Infinity" strings, which are not valid I-JSON and would not round-trip in other JCS implementations.  
  Fetch label: fetch:pkg.go.dev
- DAG-CBOR mandates deterministic map key ordering: keys are sorted byte-wise lexically including their major-type-3 header and length, which means shorter keys sort before longer ones (length-first, then bytewise), unlike JCS which sorts by UTF-16 code units.  
  Fetch label: fetch:ipld.io
- DAG-CBOR requires all floats to be encoded as 64-bit doubles and forbids NaN, Infinity and -Infinity entirely; this is a strict float rule that a cross-language writer must follow to get byte-identical output.  
  Fetch label: fetch:ipld.io
- DAG-CBOR strictness rules require shortest-possible integer and length encodings, permit only CBOR tag 42 (for CIDs, with the 0x00 identity multibase prefix byte that must not be omitted), and require exactly one top-level object per block; these constraints are what make encoding canonical.  
  Fetch label: fetch:ipld.io
- The spec explicitly allows decoders to relax strictness (map key order, integer/length size, float precision) for legacy data, and notes that go-ipld-prime and go-ipld-cbor do not enforce strictness on decode, so a CID computed from re-encoded data may differ from the original block's CID unless strict encoding is used end-to-end.  
  Fetch label: fetch:ipld.io
- JavaScript DAG-CBOR implementations cannot distinguish integer-valued floats from integers and must use BigInt for integers beyond Number.MAX_SAFE_INTEGER, which is a documented cross-language round-trip pitfall for byte-identical encodings.  
  Fetch label: fetch:ipld.io
- go-ipld-prime's dagcbor Encode sorts map keys by default using RFC 7049 (length-first) ordering, not RFC 8949 bytewise ordering, and exposes a MapSortMode option in EncodeOptions to change this.  
  Fetch label: fetch:pkg.go.dev
- go-ipld-prime's dagcbor Encode only emits 64-bit floats, so cross-language writers must also encode all floats as CBOR float64 to get byte-identical output.  
  Fetch label: fetch:pkg.go.dev
- The go-ipld-prime dagcbor decoder enforces strictness on integer/header minimality, NaN/Infinity and duplicate keys, but does NOT enforce sorted map keys or 64-bit floats on decode, so round-tripping non-canonical input through Go will silently re-encode to different bytes (and a different CID).  
  Fetch label: fetch:pkg.go.dev
- In go-ipld-prime's DAG-CBOR codec, CBOR tag 42 is the only tag accepted and must contain a valid CID; links are controlled by the AllowLinks option on both encode and decode.  
  Fetch label: fetch:pkg.go.dev
- Indefinite-length CBOR items are always rejected by go-ipld-prime's dagcbor decoder, even with RelaxedDecode, which otherwise tolerates unsorted and duplicate map keys and non-64-bit floats.  
  Fetch label: fetch:pkg.go.dev
- RFC 8949's core deterministic encoding (section 4.2.1) requires map keys to be sorted by bytewise lexicographic order of their deterministic encodings, which differs from the length-first ordering of RFC 7049 'Canonical CBOR' that IPLD DAG-CBOR inherits; two encoders following different RFC orderings will produce different bytes (and thus different CIDs) for the same map once key lengths differ.  
  Fetch label: fetch:rfc-editor.org
- RFC 8949 section 4.2.3 defines 'Length-First Map Key Ordering' as an alternative kept for compatibility with RFC 7049's Canonical CBOR: shorter encoded keys sort before longer ones, with equal-length keys sorted bytewise lexicographically. This is the ordering DAG-CBOR specifies, so a generic RFC 8949 'deterministic' CBOR library will not by default produce DAG-CBOR-compatible key order.  
  Fetch label: fetch:rfc-editor.org
- Deterministic CBOR requires integer, length and tag arguments to be encoded in their shortest possible form (preferred serialization), so an encoder that emits e.g. a 2-byte argument for a value under 24 breaks byte-identity.  
  Fetch label: fetch:rfc-editor.org
- RFC 8949 deterministic encoding requires floats to use the shortest float width (binary16/32/64) that preserves the value, e.g. 1.5 encodes as binary16; DAG-CBOR deliberately deviates by mandating 64-bit floats always, so a core-deterministic CBOR encoder and a DAG-CBOR encoder produce different bytes for most floats.  
  Fetch label: fetch:rfc-editor.org
- Deterministic CBOR forbids indefinite-length (streaming) items, requiring definite-length encodings for strings, arrays and maps; NaN handling is left to the application protocol (section 4.2.2) which must pick a single representation, typically 0xf97e00, meaning NaN/float edge cases are a protocol-level pitfall rather than resolved by the base spec.  
  Fetch label: fetch:rfc-editor.org
- As of the May 2022 spec change, the IPLD DAG-CBOR spec permits decoders to relax strictness by default, meaning a DAG-CBOR decoder may accept non-canonical input (e.g. unsorted map keys) and the strictness requirements bind encoders, not decoders; an opt-in strict mode is only optional.  
  Fetch label: fetch:github.com
- The DAG-CBOR spec enumerates five specific rules that decoders may relax: map key ordering, minimal integer encoding, minimal length descriptors for major types 2-5, minimal tag 42 encoding (0xd82a), and float precision (single/half allowed on decode). Consequently a successful decode does not prove the bytes were canonical, so CID-equality across writers cannot be verified by decode acceptance alone.  
  Fetch label: fetch:github.com
- The DAG-JSON spec was changed at the same time so that its canonicalization rules are a MUST for encoders only, with decoders told not to enforce strictness by default (optional opt-in), mirroring DAG-CBOR.  
  Fetch label: fetch:github.com
- The Go cbor-gen library (used by Filecoin) is documented in the DAG-CBOR spec as a non-conforming encoder with respect to map key ordering: it sorts keys with Go's sort.Strings() (plain lexicographic) rather than the DAG-CBOR length-first ordering, and PR discussion states the Filecoin genesis block has this wrong — a concrete cross-implementation CID-divergence pitfall.  
  Fetch label: fetch:github.com
- The JavaScript @ipld/dag-cbor decoder, as of May 2022, enforces only partial strictness on decode (rejecting non-minimal integer encodings) while not enforcing map key ordering or float size; the spec text notes its legacy predecessor and the Java and Go cbor-gen implementations enforce no decode strictness at all.  
  Fetch label: fetch:github.com
- The IPLD DAG-CBOR spec was, as of late 2019, imprecise about how parsers should handle CBOR tags other than tag 42 (CID); the issue proposed ignoring foreign tags and writing only tag 42 when encoding.  
  Fetch label: fetch:github.com
- IPLD maintainers (warpfork) argued DAG-CBOR decoders should reject foreign tags by default rather than silently ignore them, with lenient parsing only as an opt-in option.  
  Fetch label: fetch:github.com
- Plain CBOR permits multiple byte encodings of the same integer (e.g. 255 as 18FF, 1900FF, 1A000000FF, 1B00000000000000FF), so DAG-CBOR needs a 'strict' mode enforcing shortest-form integers to guarantee one encoding per value.  
  Fetch label: fetch:github.com
- The stated design goal of DAG-CBOR strictness is that bit-for-bit identical content must map to exactly one CID, i.e. identical logical content should not be expressible under multiple cryptographic identifiers.  
  Fetch label: fetch:github.com
- Deterministic map key ordering was identified as part of the same strictness/canonical-encoding requirement set, and the resulting spec tightening was done via ipld/specs PR #236 (commits titled 'DagCBOR: tighten up spec, additional strictness requirements') in January-March 2020.  
  Fetch label: fetch:github.com
- The DAG-JSON spec mandates a canonical serialization: encoders MUST sort map keys by their UTF-8 byte representation and MUST strip all whitespace, but decoders are NOT required to enforce this (strict mode is only optional), so byte-identical encodings are guaranteed only on the encode side.  
  Fetch label: fetch:ipld.io
- DAG-JSON distinguishes integers from floats purely by the presence of a decimal point, requires floats without a fractional part to be encoded WITH a decimal point to round-trip, allows exponent notation, and does not support Infinity/NaN; the spec recommends avoiding floats in content-addressed data.  
  Fetch label: fetch:ipld.io
- DAG-JSON permits big integers at the format level, but JavaScript's native JSON parser cannot handle integers above 2^53-1, which is a documented cross-language integer-limit pitfall for JS writers/readers.  
  Fetch label: fetch:ipld.io
- DAG-JSON reserves maps whose first key is "/" for Links (CIDs encoded as {"/": <base32 CIDv1 string>}) and Bytes ({"/": {"bytes": <unpadded base64>}}), and there is no escape mechanism, so ordinary JSON-shaped records that legitimately contain a "/" key cannot be represented and must be rejected as invalid DAG-JSON.  
  Fetch label: fetch:ipld.io
- Conforming DAG-JSON implementations exist for JavaScript (@ipld/dag-json) and Go (go-ipld-prime), but the legacy JS library ipld-dag-json is non-conforming: it does not enforce the reserved-namespace rules and encodes bytes with the old multibase 'm' prefix, which would produce different bytes (and thus different CIDs) than conforming encoders.  
  Fetch label: fetch:ipld.io
- The DAG-CBOR spec requires every map's keys to be sorted length-first by the byte representation of the string keys (then byte-wise lexically), so any conformant encoder in any language must produce the same key order, which is a prerequisite for byte-identical encodings and identical CIDs.  
  Fetch label: fetch:github.com
- DAG-CBOR mandates that floating-point values are always encoded in 64-bit double-precision form and that NaN, Infinity and -Infinity are rejected, removing float-width ambiguity from the canonical encoding; integers must also use the shortest possible CBOR encoding.  
  Fetch label: fetch:github.com
- DAG-CBOR permits only CBOR tag 42 (for CIDs) and no other tags, and CIDs are encoded as a byte string with the multibase identity prefix 0x00 prepended; indefinite-length items are not allowed. This fixes how links are serialised across languages.  
  Fetch label: fetch:github.com
- Reference implementations do not fully enforce the strictness rules on decode: go-ipld-prime and go-ipld-cbor accept non-strict blocks (and NaN/infinities), and @ipld/dag-cbor (JS) does not enforce map ordering or reject non-64-bit floats on decode. Byte-identical output therefore depends on encoder behaviour, not on decoders catching non-canonical input.  
  Fetch label: fetch:github.com
- The spec documents a JavaScript-specific pitfall: JS numbers do not distinguish integers from floats and bit-wise operations only work within 32 bits, so large integers and the int/float distinction can be lost on round-trip in JS writers unless BigInt is used.  
  Fetch label: fetch:github.com
