package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// PremiseStore is the working-store access the premise endpoints need. A
// Store that also implements it gets the endpoints; otherwise they answer
// 501.
type PremiseStore interface {
	PutCandidate(ctx context.Context, m *model.Micropublication) error
	GetCandidate(ctx context.Context, uuid string) (*model.Micropublication, error)
	ListCandidates(ctx context.Context) ([]string, error)
	GetCurrent(ctx context.Context, uuid string) (storage.VersionRef, error)
	GetVersion(ctx context.Context, cid string) (*model.Micropublication, error)
	ListCurrent(ctx context.Context) ([]string, error)
	ListAttestations(ctx context.Context, premiseUUID string) ([]*model.ValidityAttestation, error)
}

// Premise statuses reported by the API.
const (
	// StatusValid: a published current version with a true, unexpired
	// attestation for that exact version.
	StatusValid = "valid"
	// StatusPublishedNotValid: a published current version whose admitting
	// attestation has since expired or been superseded by no true one.
	StatusPublishedNotValid = "published-not-currently-valid"
	// StatusCandidate: a draft in the working store, not published.
	StatusCandidate = "candidate"
)

// AttestationView is one evaluation as the API reports it.
type AttestationView struct {
	UUID        string `json:"uuid"`
	Version     string `json:"version"`
	Result      string `json:"result"`
	Method      string `json:"method"`
	Evaluator   string `json:"evaluator"`
	EvaluatedAt string `json:"evaluated_at"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// PremiseView is the API's answer to "does this premise exist, and is it
// valid?". Validity comes only from attestations of the exact version; the
// author's attribution carries no evidential weight.
type PremiseView struct {
	UUID         string                `json:"uuid"`
	Status       string                `json:"status"`
	Version      string                `json:"version,omitempty"`
	Statement    string                `json:"statement"`
	Terms        []model.TermGrounding `json:"terms,omitempty"`
	Support      []model.Evidence      `json:"support,omitempty"`
	Challenge    []model.Evidence      `json:"challenge,omitempty"`
	Attribution  model.Attribution     `json:"attribution"`
	Attestations []AttestationView     `json:"attestations"`
}

func (s *Server) premiseStore(w http.ResponseWriter) (PremiseStore, bool) {
	ps, ok := s.Store.(PremiseStore)
	if !ok {
		writeErr(w, http.StatusNotImplemented, "premise endpoints need a working store")
	}
	return ps, ok
}

// view builds the answer for one premise UUID, or storage.ErrNotFound.
func (s *Server) view(ctx context.Context, ps PremiseStore, uuid string) (*PremiseView, error) {
	now := s.now()
	atts, err := ps.ListAttestations(ctx, uuid)
	if err != nil {
		return nil, err
	}
	var m *model.Micropublication
	pv := &PremiseView{UUID: uuid}
	if ref, err := ps.GetCurrent(ctx, uuid); err == nil {
		if m, err = ps.GetVersion(ctx, ref.CID); err != nil {
			return nil, err
		}
		pv.Version, pv.Status = ref.CID, StatusPublishedNotValid
		for _, a := range atts {
			if a.PremiseVersionCID == ref.CID && a.EstablishesValidityAt(now) {
				pv.Status = StatusValid
			}
		}
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	} else if m, err = ps.GetCandidate(ctx, uuid); err != nil {
		return nil, err
	} else {
		pv.Status = StatusCandidate
	}
	pv.Statement, pv.Terms = m.Claim.Text, m.Claim.Terms
	pv.Support, pv.Challenge, pv.Attribution = m.SupportGraph, m.ChallengeGraph, m.Attribution
	pv.Attestations = []AttestationView{}
	for _, a := range atts {
		pv.Attestations = append(pv.Attestations, AttestationView{UUID: a.UUID, Version: a.PremiseVersionCID, Result: string(a.Result), Method: a.Method, Evaluator: a.Evaluator, EvaluatedAt: a.EvaluatedAt, ExpiresAt: a.ExpiresAt})
	}
	return pv, nil
}

func (s *Server) getPremise(w http.ResponseWriter, r *http.Request, uuid string) {
	ps, ok := s.premiseStore(w)
	if !ok {
		return
	}
	pv, err := s.view(r.Context(), ps, uuid)
	if errors.Is(err, storage.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no such premise")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pv)
}

// searchPremises answers GET /api/premises?q=text with every premise whose
// statement contains the text (case-insensitive), valid ones first. It scans
// the working store; the indexed search of docs/specs/indexing-retrieval.md
// replaces the scan when it exists.
func (s *Server) searchPremises(w http.ResponseWriter, r *http.Request) {
	ps, ok := s.premiseStore(w)
	if !ok {
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if q == "" {
		writeErr(w, http.StatusBadRequest, "q is required")
		return
	}
	ctx := r.Context()
	seen := map[string]bool{}
	var ids []string
	for _, list := range []func(context.Context) ([]string, error){ps.ListCurrent, ps.ListCandidates} {
		got, err := list(ctx)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, id := range got {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	results := []*PremiseView{}
	for _, id := range ids {
		pv, err := s.view(ctx, ps, id)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(pv.Statement), q) {
			results = append(results, pv)
		}
		if len(results) >= 200 {
			break
		}
	}
	rank := map[string]int{StatusValid: 0, StatusPublishedNotValid: 1, StatusCandidate: 2}
	sort.SliceStable(results, func(i, j int) bool { return rank[results[i].Status] < rank[results[j].Status] })
	writeJSON(w, http.StatusOK, map[string]interface{}{"query": r.URL.Query().Get("q"), "results": results})
}

type createPremiseRequest struct {
	Statement string                `json:"statement"`
	Terms     []model.TermGrounding `json:"terms"`
	Support   []model.Evidence      `json:"support"`
	Agent     string                `json:"agent"`
}

// createPremise records a candidate premise. Anyone may state anything; the
// candidate is a draft and becomes valid only through evaluation.
func (s *Server) createPremise(w http.ResponseWriter, r *http.Request) {
	ps, ok := s.premiseStore(w)
	if !ok {
		return
	}
	var req createPremiseRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Statement) == "" || strings.TrimSpace(req.Agent) == "" {
		writeErr(w, http.StatusBadRequest, "statement and agent are required")
		return
	}
	m, err := model.New(req.Statement, model.Attribution{Agent: req.Agent, AtTime: s.now().UTC().Format(time.RFC3339)})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	m.Claim.Terms, m.SupportGraph = req.Terms, req.Support
	if err := ps.PutCandidate(r.Context(), m); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"uuid": m.UUID, "status": StatusCandidate})
}
