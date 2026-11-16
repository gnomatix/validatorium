package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"

	"gnomatix/validatorium/internal/autopublish"
	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/ipfsnode"
	"gnomatix/validatorium/internal/namespace"
	"gnomatix/validatorium/internal/orbit"
	"gnomatix/validatorium/internal/promote"
)

// cmdPublish evaluates working-store candidates and publishes each one whose
// evaluation establishes validity to the global store (automatic publication
// on validity). With no -uuid it runs every candidate draft.
func cmdPublish(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	ws := openWS(fs)
	ipfsRepo := fs.String("ipfs-repo", "./ipfs_repo", "path of the embedded Kubo repository")
	orbitDir := fs.String("orbit-dir", "./orbitdb_repo", "directory for go-orbit-db cache and keystore")
	pointerDB := fs.String("pointer-store", defaultPointerStoreName, "pointer store name, or an /orbitdb/... address to join")
	offline := fs.Bool("offline", false, "run the IPFS node on an isolated in-process network (no peers)")
	uuids := fs.String("uuid", "", "comma-separated premise UUIDs; empty means every candidate draft")
	sum := fs.String("sha256", "", "expected SHA-256 digest, for the file-digest method")
	evaluator := fs.String("evaluator", "gemini-cli", "evaluator identity recorded on attestations (attribution only)")
	nsList := fs.String("ns", "", "comma-separated namespaces to place published versions in")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var namespaces []namespace.Name
	for _, n := range splitList(*nsList) {
		name, err := namespace.Parse(n)
		if err != nil {
			return err
		}
		namespaces = append(namespaces, name)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	st, err := bbolt.OpenWorkingStore(*ws)
	if err != nil {
		return err
	}
	defer st.Close()

	ids := splitList(*uuids)
	if len(ids) == 0 {
		if ids, err = st.ListCandidates(ctx); err != nil {
			return err
		}
	}
	if len(ids) == 0 {
		return errors.New("no candidate drafts to evaluate")
	}

	var node *ipfsnode.Node
	if *offline {
		node, err = ipfsnode.NewMockNode(ctx, mocknet.New())
	} else {
		node, err = ipfsnode.NewNode(ctx, *ipfsRepo)
	}
	if err != nil {
		return fmt.Errorf("start IPFS node: %w", err)
	}
	defer node.Close()
	gs, err := orbit.Open(ctx, node, *orbitDir, *pointerDB, &orbit.Options{PinBlocks: true})
	if err != nil {
		return fmt.Errorf("open global store: %w", err)
	}
	defer gs.Close()

	reg, err := evaluate.NewRegistry(evaluate.FileDigest{})
	if err != nil {
		return err
	}
	scope := map[string]interface{}{}
	if *sum != "" {
		scope["sha256"] = *sum
	}
	pub := &autopublish.Publisher{
		Promoter:   &promote.Promoter{Working: st, Global: gs},
		Evaluators: reg,
		Evaluator:  *evaluator,
		Namespaces: namespaces,
	}
	for _, id := range ids {
		o, err := pub.Run(ctx, id, scope)
		if err != nil {
			return err
		}
		if o.Published {
			fmt.Fprintf(out, "published %s %s attestation %s\n", id, o.Promotion.Version.CID, o.Promotion.AttestationCID)
		} else {
			fmt.Fprintf(out, "not-published %s %s (%s)\n", id, o.Attestation.Result, o.Reason)
		}
	}
	fmt.Fprintf(out, "pointer store %s\n", gs.Address())
	return nil
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
