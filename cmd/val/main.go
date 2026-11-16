// Command val runs a Validatorium node: a bbolt working store beside an
// embedded Kubo node and a go-orbit-db pointer store.
//
// With no subcommand it wires the two storage tiers together and reports
// their addresses. The subcommands create, evaluate, attestations and
// release run the local premise workflow against the working store only.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/ipfsnode"
	"gnomatix/validatorium/internal/orbit"
)

// defaultPointerStoreName is the pointer store a node creates when it is not
// given an existing /orbitdb/... address to join.
const defaultPointerStoreName = "validatorium-pointers"

func main() {
	if len(os.Args) > 1 {
		if cmd, ok := subcommands[os.Args[1]]; ok {
			if err := cmd(os.Args[2:], os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, "val "+os.Args[1]+":", err)
				os.Exit(1)
			}
			return
		}
	}
	var (
		ipfsRepo    = flag.String("ipfs-repo", "./ipfs_repo", "path of the embedded Kubo repository")
		orbitDir    = flag.String("orbit-dir", "./orbitdb_repo", "directory for go-orbit-db cache and keystore")
		workingPath = flag.String("working-store", "./working_store.db", "path of the bbolt working store")
		pointerDB   = flag.String("pointer-store", defaultPointerStoreName, "pointer store name, or an /orbitdb/... address to join")
		offline     = flag.Bool("offline", false, "run the IPFS node on an isolated in-process network (no peers)")
		wait        = flag.Bool("serve", false, "keep the node running until interrupted")
	)
	flag.Parse()

	if err := run(*ipfsRepo, *orbitDir, *workingPath, *pointerDB, *offline, *wait); err != nil {
		fmt.Fprintln(os.Stderr, "val:", err)
		os.Exit(1)
	}
}

func run(ipfsRepo, orbitDir, workingPath, pointerDB string, offline, wait bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ws, err := bbolt.OpenWorkingStore(workingPath)
	if err != nil {
		return fmt.Errorf("open working store: %w", err)
	}
	defer ws.Close()

	var node *ipfsnode.Node
	if offline {
		node, err = ipfsnode.NewMockNode(ctx, mocknet.New())
	} else {
		node, err = ipfsnode.NewNode(ctx, ipfsRepo)
	}
	if err != nil {
		return fmt.Errorf("start IPFS node: %w", err)
	}
	defer node.Close()

	gs, err := orbit.Open(ctx, node, orbitDir, pointerDB, &orbit.Options{PinBlocks: true})
	if err != nil {
		return fmt.Errorf("open global store: %w", err)
	}
	defer gs.Close()

	candidates, err := ws.ListCandidates(ctx)
	if err != nil {
		return fmt.Errorf("list candidates: %w", err)
	}

	fmt.Printf("working store:   %s (%d candidate drafts)\n", workingPath, len(candidates))
	fmt.Printf("ipfs peer:       %s\n", node.PeerID())
	fmt.Printf("orbit identity:  %s\n", gs.Identity())
	fmt.Printf("pointer store:   %s\n", gs.Address())

	if wait {
		fmt.Println("serving; press Ctrl-C to stop")
		<-ctx.Done()
	}
	return nil
}
