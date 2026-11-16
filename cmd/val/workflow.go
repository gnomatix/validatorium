package main

// Local premise workflow subcommands. They use only the bbolt working store,
// so they run with no IPFS node. Promotion and release publication need the
// global store and are not exposed here.

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/release"
	"gnomatix/validatorium/internal/storage"
)

// subcommands maps each workflow subcommand to its handler.
var subcommands = map[string]func(args []string, out io.Writer) error{
	"create":       cmdCreate,
	"evaluate":     cmdEvaluate,
	"attestations": cmdAttestations,
	"release":      cmdRelease,
}

func openWS(fs *flag.FlagSet) *string {
	return fs.String("working-store", "./working_store.db", "path of the bbolt working store")
}

// cmdCreate stores a new candidate premise and prints its UUID.
func cmdCreate(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	ws := openWS(fs)
	text := fs.String("text", "", "statement text of the premise (required)")
	agent := fs.String("agent", "gemini-cli", "attribution agent")
	file := fs.String("file", "", "file whose SHA-256 digest the premise asserts; adds the file-digest method and the file as evidence")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*text) == "" {
		return errors.New("create: -text is required")
	}
	m, err := model.New(*text, model.Attribution{Agent: *agent, AtTime: time.Now().UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	if *file != "" {
		abs, err := filepath.Abs(*file)
		if err != nil {
			return err
		}
		m.SupportGraph = []model.Evidence{
			{Kind: model.EvidenceMethod, Reference: evaluate.FileDigestMethod},
			{Kind: model.EvidenceData, Reference: "file://" + abs},
		}
	}
	if err := m.ValidateRecord(); err != nil {
		return err
	}
	st, err := bbolt.OpenWorkingStore(*ws)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.PutCandidate(context.Background(), m); err != nil {
		return err
	}
	fmt.Fprintln(out, m.UUID)
	return nil
}

// loadPremise returns the draft for uuid, or its current stored version.
func loadPremise(ctx context.Context, st *bbolt.WorkingStore, uuid string) (*model.Micropublication, error) {
	m, err := st.GetCandidate(ctx, uuid)
	if err == nil {
		return m, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	ref, err := st.GetCurrent(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("premise %s: %w", uuid, err)
	}
	return st.GetVersion(ctx, ref.CID)
}

// cmdEvaluate evaluates a premise, records the attestation, and prints the
// result and attestation UUID.
func cmdEvaluate(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	ws := openWS(fs)
	uuid := fs.String("uuid", "", "premise UUID (required)")
	sum := fs.String("sha256", "", "expected SHA-256 digest, for the file-digest method")
	evaluator := fs.String("evaluator", "gemini-cli", "evaluator identity recorded on the attestation (attribution only)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *uuid == "" {
		return errors.New("evaluate: -uuid is required")
	}
	st, err := bbolt.OpenWorkingStore(*ws)
	if err != nil {
		return err
	}
	defer st.Close()
	ctx := context.Background()
	prem, err := loadPremise(ctx, st, *uuid)
	if err != nil {
		return err
	}
	reg, err := evaluate.NewRegistry(evaluate.FileDigest{})
	if err != nil {
		return err
	}
	scope := map[string]interface{}{}
	if *sum != "" {
		scope["sha256"] = *sum
	}
	att, outcome, err := reg.Attest(ctx, prem, scope, *evaluator, time.Now())
	if err != nil {
		return err
	}
	if err := st.PutAttestation(ctx, att); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s %s %s\n", att.Result, att.UUID, outcome.Detail)
	return nil
}

// cmdAttestations prints every attestation recorded for a premise, one JSON
// object per line.
func cmdAttestations(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("attestations", flag.ContinueOnError)
	ws := openWS(fs)
	uuid := fs.String("uuid", "", "premise UUID (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *uuid == "" {
		return errors.New("attestations: -uuid is required")
	}
	st, err := bbolt.OpenWorkingStore(*ws)
	if err != nil {
		return err
	}
	defer st.Close()
	atts, err := st.ListAttestations(context.Background(), *uuid)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	for _, a := range atts {
		if err := enc.Encode(a); err != nil {
			return err
		}
	}
	return nil
}

// cmdRelease builds a release manifest from the working store and prints it
// with its CID and exclusions as JSON. It does not publish.
func cmdRelease(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	ws := openWS(fs)
	label := fs.String("label", "", "release label (required)")
	uuids := fs.String("uuids", "", "comma-separated candidate premise UUIDs (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *label == "" || *uuids == "" {
		return errors.New("release: -label and -uuids are required")
	}
	st, err := bbolt.OpenWorkingStore(*ws)
	if err != nil {
		return err
	}
	defer st.Close()
	var cands []string
	for _, u := range strings.Split(*uuids, ",") {
		if u = strings.TrimSpace(u); u != "" {
			cands = append(cands, u)
		}
	}
	res, err := release.Build(context.Background(), st, *label, time.Now(), cands)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		CID        string              `json:"cid"`
		Manifest   release.Manifest    `json:"manifest"`
		Exclusions []release.Exclusion `json:"exclusions"`
	}{res.CID, res.Manifest, res.Exclusions})
}
