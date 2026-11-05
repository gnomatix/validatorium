# go-orbit-db parity with OrbitDB JS, maintenance status, Go-only fallbacks (Beads validatorium-bqe.3)

**Status: UNVERIFIED.** Extracted on 2026-11-05 from a deep-research run that was stopped
during its fetch phase, before adversarial verification and synthesis. Each line is a
claim as extracted from one fetched source by a reader agent; none has been checked
against a second source. Treat as leads to confirm, not as findings.

Fetched sources with extracted claims: 0; claims: 45

## Claims

- go-orbit-db's most recent release is v1.22.2, published 2026-06-03 by the github-actions bot, marked Latest, and containing a single bug fix to the pubsub head-announcement path (a fix needed to satisfy libp2p pubsub's seqno validator).  
  Fetch label: fetch:github.com
- Release cadence has slowed sharply: after frequent releases in 2022-2023 (v1.17.1 May 2022 through v1.22.0 on 2023-05-04), only two releases followed, v1.22.1 on 2024-04-19 (dependency-compatibility and lint fixes only) and v1.22.2 on 2026-06-03, i.e. roughly one patch release per year with no new features since v1.22.0.  
  Fetch label: fetch:github.com
- go-orbit-db tracks Kubo/libp2p upgrades as explicit features: v1.18.0 (2022-11-15) bumped to Kubo 0.16 and libp2p 0.13, and v1.21.0 (2023-04-20) updated to Kubo 0.19; no later release notes mention a Kubo bump, so the pinned Kubo version in the 1.22.x line dates from the 2023 upgrade.  
  Fetch label: fetch:github.com
- The head-exchange/replication protocol was changed in the 1.19-1.20 series: v1.19.0 (2022-11-17) introduced a breaking change to orbitdb.Open and StoreConstructor, and v1.20.0 (2023-03-28) added a `remoteheads` field to the head exchange message, indicating the Go wire format diverged from earlier versions (and is a point to check against the JS OrbitDB v2/v3/v4 heads exchange).  
  Fetch label: fetch:github.com
- All go-orbit-db releases are produced by an automated GitHub Actions pipeline with conventional-changelog bodies (Bug Fixes / Features sections, compare links); none are marked pre-release, and the signing key on releases from v1.17.1 through v1.22.0 is shown as expired while v1.22.1 and v1.22.2 carry a verified, non-expired signature.  
  Fetch label: fetch:github.com
- As of February 16, 2024, Berty collaborator jefft0 stated that Berty planned to rewrite go-orbit-db and that compatibility with the JavaScript OrbitDB version was not a focus; the issue was closed the same day.  
  Fetch label: fetch:github.com
- In January 2020, go-orbit-db maintainer glouvigny (Berty) stated that JS/Go compatibility had not been tested and could not be guaranteed.  
  Fetch label: fetch:github.com
- As of January 2020, go-orbit-db lacked the feed, docs (docstore) and counter store types, as well as some access controllers (such as Ethereum-based ones), and opening a store requiring them would throw an error.  
  Fetch label: fetch:github.com
- The maintainer identified two unverified compatibility dimensions in 2020: whether a JS client can talk to a Go client, whether each can load datastores created by the other, and whether addresses work across both implementations.  
  Fetch label: fetch:github.com
- A user in January 2020 reported being unable to get the Go and JS implementations to communicate with each other.  
  Fetch label: fetch:github.com
- go-orbit-db's stores package implements only three store types — document store, event log store and key-value store — with no feed store or counter store directory present, so it lacks parity with the JS OrbitDB feed/counter stores.  
  Fetch label: fetch:github.com
- go-orbit-db ships three access controller implementations — ipfs, orbitdb and simple — plus an interface/manifest layer, matching the JS IPFS and OrbitDB access controllers and allowing custom controllers via the interface.  
  Fetch label: fetch:github.com
- go-orbit-db's release cadence has slowed to roughly one release per year: v1.22.0 (4 May 2023), v1.22.1 (19 Apr 2024), and the latest v1.22.2 (3 Jun 2026), whose only change is a pubsub head-announcement fix.  
  Fetch label: fetch:github.com
- The master branch is still receiving commits as of late October 2026, but recent activity consists of dependabot dependency bumps and CI workflow updates merged by a single Berty maintainer (jefft0), not feature work.  
  Fetch label: fetch:github.com
- The repository has only 3 open issues, one of which (open since May 2025) reports that using OrbitDBAccessController fails to create a store, and the README positions the project as intending full compatibility with the JavaScript OrbitDB rather than claiming it.  
  Fetch label: fetch:github.com
- The latest tagged release of berty.tech/go-orbit-db on pkg.go.dev is v1.22.2, published Jun 3, 2026; the preceding releases were v1.22.1 (Apr 19, 2024) and v1.22.0 (May 4, 2023), implying roughly one release per year since 2023 versus dozens of releases per year in 2020-2022.  
  Fetch label: fetch:pkg.go.dev
- go-orbit-db's README states the project is a Go port derived from the JavaScript orbit-db codebase and that it intends to be fully compatible with the JS version, but the pkg.go.dev page does not name which JS orbit-db major version (v1/v2/v3/v4) it tracks.  
  Fetch label: fetch:pkg.go.dev
- The module ships store packages for eventlog, key-value and document stores only (eventlogstore, kvstore, documentstore directories); no counter or feed store package is listed on pkg.go.dev.  
  Fetch label: fetch:pkg.go.dev
- go-orbit-db ships three access controller packages: accesscontroller/ipfs, accesscontroller/orbitdb, and accesscontroller/simple (the latter being non-persistent), plus a replicator package for peer replication.  
  Fetch label: fetch:pkg.go.dev
- go-orbit-db's public constructor NewOrbitDB takes a coreiface.CoreAPI from github.com/ipfs/kubo/core/coreiface, i.e. it is built against Kubo's CoreAPI rather than a standalone libp2p/IPLD stack; the module is Apache-2.0 licensed, maintained by Berty, and has 25 known importers.  
  Fetch label: fetch:pkg.go.dev
- github.com/stateless-minds/go-orbit-db is a community fork of Berty's go-orbit-db (canonical import replaced) whose most recent tagged release is v1.23.15, published 24 June 2025; it carries 18 tagged versions on pkg.go.dev including v1.22.2/v1.22.3 (Dec 2024) and a v1.23.x line (v1.23.0 Jul 2023 through v1.23.15 Jun 2025), with a burst of 11 tags between late May and late June 2025.  
  Fetch label: fetch:pkg.go.dev
- The Go OrbitDB port (as packaged in this fork, which tracks Berty's layout) ships exactly three store implementations — documentstore, eventlogstore and kvstore — and has no feed store or counter store package, unlike OrbitDB JS.  
  Fetch label: fetch:pkg.go.dev
- The Go port provides four access-controller packages — ipfs, orbitdb, orbitdb-selfenroll and simple (the latter non-persistent) — and the main constructor registers default access controllers and store types.  
  Fetch label: fetch:pkg.go.dev
- The Go port's identity, keystore and log/entry handling come from a separate go-ipfs-log module (here the stateless-minds fork of Berty's go-ipfs-log), and the OrbitDB instance is constructed on a Kubo CoreAPI (github.com/ipfs/kubo/core/coreiface) plus libp2p core types — i.e. it is built to embed Kubo directly rather than talk to a remote daemon.  
  Fetch label: fetch:pkg.go.dev
- The README asserts JS-compatibility only as an intention (a port derived from the original JS orbit-db), not as a verified property; the fork has zero known importers on pkg.go.dev and its events package is marked deprecated in favour of EventBus.  
  Fetch label: fetch:pkg.go.dev
- The orbitdb/go-orbitdb repository (under the OrbitDB GitHub org, not Berty) merged on 2026-09-26 a port of @orbitdb/core 4.0.0 that is wire-compatible with JS OrbitDB v4; the PR states the previous Go prototype could not exchange entries with a JS peer at all.  
  Fetch label: fetch:github.com
- The port claims byte-identical oplog entries and hashes with the same LWW ordering as JS v4, and uses secp256k1 keys via go-libp2p that can open JS keystores (identity/entry format parity).  
  Fetch label: fetch:github.com
- go-orbitdb's sync protocol matches JS v4: pubsub topic equals the database address, head exchange over the /orbitdb/heads/<address> protocol, and raw block publishing.  
  Fetch label: fetch:github.com
- Supported store types in the port are the v4 set (Events, KeyValue, KeyValueIndexed, Documents) with IPFS and OrbitDB access controllers; there is no feed or counter store, matching JS v4 rather than the legacy v1/v2 store list.  
  Fetch label: fetch:github.com
- Live interop CI tests show a JS OrbitDB v4 peer on Helia replicating bidirectionally with the Go implementation, including concurrent writers, documents, access-controller grants, and encrypted databases.  
  Fetch label: fetch:github.com
- As indexed by ecosyste.ms (package record synced 2026-11-05), the latest release of berty.tech/go-orbit-db is v1.22.2, published 2026-06-03, out of 56 total versions since the first release on 2019-08-13; the previous release v1.22.1 was 2024-04-19 and v1.22.0 was 2023-05-04, i.e. roughly one patch release per year since 2023.  
  Fetch label: fetch:repos.ecosyste.ms
- The berty/go-orbit-db GitHub repository (as synced by ecosyste.ms on 2026-10-27) is not archived, was last pushed 2026-10-23, has 457 stars, 60 forks, 6 open issues, 56 tags, and a lifetime total of 184 commits from 16 committers.  
  Fetch label: fetch:repos.ecosyste.ms
- The OpenSSF Scorecard run recorded for berty/go-orbit-db on 2025-08-11 gave an overall score of 3.2/10, scoring the 'Maintained' check 0 because no commits and no issue activity were found in the prior 90 days, and flagging 10 existing vulnerabilities in dependencies.  
  Fetch label: fetch:repos.ecosyste.ms
- ecosyste.ms counts 37 dependents of berty.tech/go-orbit-db and lists 40 dependent repositories, the large majority being forks of berty/berty; the notable non-Berty users are mrusme/superhighway84 (792 stars, pinned to v1.17.1, last pushed 2025-12-15), IceFireDB forks, and georgeGeorgakakos/optimusdb (last pushed 2026-10-14). Berty's own berty/berty (9,345 stars, pushed 2026-11-03) and berty/weshnet (256 stars, pushed 2026-10-30) are listed as dependents, confirming continued in-house use.  
  Fetch label: fetch:repos.ecosyste.ms
- Across all indexed go.mod/go.sum manifests, no dependent repository pins a v1.22.x release of go-orbit-db; the most common requirements are v1.17.1 (33 manifest entries) and v1.10.11 (25), and the berty/berty go.mod entry indexed by ecosyste.ms (manifest record dated 2023-01-17) requires v1.19.1 as a direct dependency. (Caveat: these manifest snapshots may be stale relative to the live repositories.)  
  Fetch label: fetch:repos.ecosyste.ms
- orbitdb/go-orbitdb (under the orbitdb GitHub org, distinct from berty/go-orbit-db) is a Go port of @orbitdb/core 4 that claims byte-for-byte compatibility with the JS v4 identities, log entries, manifests, access controllers and sync protocol.  
  Fetch label: fetch:github.com
- go-orbitdb supports the OrbitDB v4 database types Events, KeyValue, KeyValueIndexed and Documents (not the v0.x-era feed/counter stores), with custom types registrable via UseDatabaseType.  
  Fetch label: fetch:github.com
- Replication uses the v4 sync protocol: gossipsub for live updates plus a /orbitdb/heads/<address> head-exchange stream, with manifests, access controllers, heads and missing ancestors fetched over bitswap; there is no DHT or delegated routing by default, so peers must be connected manually.  
  Fetch label: fetch:github.com
- go-orbitdb ships IPFS and OrbitDB access controllers (with custom controllers via UseAccessController) and a publickey identity provider backed by a secp256k1 keystore that can read JavaScript keystores.  
  Fetch label: fetch:github.com
- JS interoperability is verified by golden vectors generated from @orbitdb/core and by live two-way replication tests against a JavaScript OrbitDB peer on Helia (@orbitdb/core 4.0.0) run on every push; the project requires Go 1.26+, has about 240 commits, 9 stars and no published releases.  
  Fetch label: fetch:github.com
- OrbitDB v2+ (JS) oplog entries carry an explicit format version field `v: 2`, and the documented entry schema consists of exactly the fields v, id, key, sig, next, refs, clock {id, time}, payload, and identity, with the entry addressed by a content hash (example `zdpu...`, i.e. dag-cbor/base58btc CID). This is the wire schema a Go implementation must match for cross-language head/entry parity.  
  Fetch label: fetch:github.com
- In OrbitDB v2+, the `identity` field of an entry is a hash reference (a `zdpu...` CID) to a separately stored identity object rather than an inline identity record, and the `key` field is a hex-encoded public key that also serves as the clock's writer id. Any Go peer interoperating with JS v2+ must resolve identities by hash and use the same key/clock-id convention.  
  Fetch label: fetch:github.com
- The OrbitDB oplog is defined as a Merkle-CRDT (Merkle-DAG) in which each entry is an IPLD structure stored in IPFS, retrievable by hash, cryptographically signed by the writer and verified by receivers; concurrent entries are totally ordered by a default Last-Write-Wins sort based on a Lamport logical clock (time first, then writer hash as tiebreaker).  
  Fetch label: fetch:github.com
- The entry payload shape is database-type specific (e.g. `{ op: 'ADD', key: null, value: ... }` for an event log), so store-type parity between Go and JS requires matching per-store payload conventions, not just the entry envelope.  
  Fetch label: fetch:github.com
- The OPLOG.md document does not describe the replication/heads-exchange protocol, the dag-cbor/sha256 encoding, identity verification details, or any compatibility notes between entry versions (v1 vs v2); its last substantive edit was in mid-2023, so it alone cannot settle wire-level parity questions and must be cross-checked against the OrbitDB source and go-orbit-db's entry/identity code.  
  Fetch label: fetch:github.com
