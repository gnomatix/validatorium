package main

import (
	"context"
	"fmt"
	"os"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/ipfsnode"
	"gnomatix/validatorium/internal/orbit"
)

func main() {
	fmt.Println("Validatorium CLI starting...")

	ctx := context.Background()

	// Initialize basic skeleton components to ensure they link and compile
	node, err := ipfsnode.NewNode(ctx, "./ipfs_repo")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start IPFS node: %v\n", err)
		os.Exit(1)
	}
	defer node.Close()

	mgr, err := orbit.NewManager(ctx, node, "./orbitdb_repo")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start OrbitDB manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

	store, err := bbolt.NewStore("./working_store.db", "premises")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open bbolt working store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	fmt.Println("Validatorium CLI skeleton started successfully.")
}
