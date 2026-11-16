// Package api serves Validatorium's HTTP interface for the ITVB Seal of Truth
// self-certification registry, so that a browser plugin can ask a running
// Validatorium instance about a page.
//
// GET /api/verification?url=U answers in the ITVB response format
// (the-zuck-stops-here docs/minutiae/seal-of-truth/seal-of-truth.md, "Verification
// API Endpoint"), with a "validatorium" block that states the record class.
// A self-certification records what its author certified; it is not a
// Validatorium validity result.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"
)

// ITVB endpoints and assets the responses point at.
const (
	ITVBBase        = "https://truth.dataglut.org"
	VerificationURL = ITVBBase + "/seal-of-truth.html"
)

// Store is the registry storage the server needs.
type Store interface {
	PutCertification(ctx context.Context, c *model.SelfCertification) (storage.VersionRef, error)
	GetCertification(ctx context.Context, uuid string) (*model.SelfCertification, storage.VersionRef, error)
	CertificationsForSubject(ctx context.Context, subject string) ([]*model.SelfCertification, error)
}

// Server is the HTTP handler.
type Server struct {
	Store Store
	// Now supplies the time; nil means time.Now.
	Now func() time.Time
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// ServeHTTP routes the API.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case path == "/api/verification" && r.Method == http.MethodGet:
		s.verification(w, r)
	case path == "/api/certifications" && r.Method == http.MethodPost:
		s.register(w, r)
	case strings.HasPrefix(path, "/api/certifications/") && strings.HasSuffix(path, "/obsolete") && r.Method == http.MethodPost:
		s.obsolete(w, r, strings.TrimSuffix(strings.TrimPrefix(path, "/api/certifications/"), "/obsolete"))
	case strings.HasPrefix(path, "/api/certifications/") && r.Method == http.MethodGet:
		s.get(w, r, strings.TrimPrefix(path, "/api/certifications/"))
	default:
		writeErr(w, http.StatusNotFound, "no such endpoint")
	}
}

type badgeURLs struct {
	SealSmall   string `json:"seal_small"`
	SealMedium  string `json:"seal_medium"`
	SealLarge   string `json:"seal_large"`
	BadgeSmall  string `json:"badge_small"`
	BadgeMedium string `json:"badge_medium"`
	BadgeLarge  string `json:"badge_large"`
}

var itvbBadges = badgeURLs{
	SealSmall:   ITVBBase + "/assets/images/verified-truth-seal-100.png",
	SealMedium:  ITVBBase + "/assets/images/verified-truth-seal-150.png",
	SealLarge:   ITVBBase + "/assets/images/verified-truth-seal.png",
	BadgeSmall:  ITVBBase + "/assets/images/verified-truth-badge-100.png",
	BadgeMedium: ITVBBase + "/assets/images/verified-truth-badge-150.png",
	BadgeLarge:  ITVBBase + "/assets/images/verified-truth-badge.png",
}

// VerificationResponse is the ITVB verification response plus the
// Validatorium block.
type VerificationResponse struct {
	Verified        bool              `json:"verified"`
	Standard        string            `json:"standard,omitempty"`
	Method          string            `json:"method,omitempty"`
	Date            string            `json:"date,omitempty"`
	BadgeURLs       *badgeURLs        `json:"badge_urls,omitempty"`
	VerificationURL string            `json:"verification_url"`
	Validatorium    ValidatoriumBlock `json:"validatorium"`
}

// ValidatoriumBlock states what the registry holds for the subject.
type ValidatoriumBlock struct {
	Subject string `json:"subject"`
	// Class is "self-certified": an author's own certification, not a
	// Validatorium validity result.
	Class string `json:"class"`
	// Current lists the current certifications; Obsolete counts the
	// obsoleted ones, which stay on record.
	Current  []CertRef `json:"current"`
	Obsolete int       `json:"obsolete"`
}

// CertRef identifies one certification version.
type CertRef struct {
	UUID           string `json:"uuid"`
	Version        string `json:"version"`
	Signer         string `json:"signer"`
	SignerVerified bool   `json:"signer_verified"`
}

func (s *Server) verification(w http.ResponseWriter, r *http.Request) {
	subject := r.URL.Query().Get("url")
	if subject == "" {
		writeErr(w, http.StatusBadRequest, "url parameter required")
		return
	}
	certs, err := s.Store.CertificationsForSubject(r.Context(), subject)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := VerificationResponse{
		VerificationURL: VerificationURL,
		Validatorium:    ValidatoriumBlock{Subject: subject, Class: "self-certified", Current: []CertRef{}},
	}
	var latest *model.SelfCertification
	for _, c := range certs {
		if c.State.Status != model.Current {
			resp.Validatorium.Obsolete++
			continue
		}
		_, ref, err := s.Store.GetCertification(r.Context(), c.UUID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		resp.Validatorium.Current = append(resp.Validatorium.Current, CertRef{UUID: c.UUID, Version: ref.CID, Signer: c.Signer, SignerVerified: c.SignerVerified})
		if latest == nil || c.Date > latest.Date {
			latest = c
		}
	}
	if latest != nil {
		resp.Verified = true
		resp.Standard, resp.Method, resp.Date = latest.Standard, latest.Method, latest.Date
		b := itvbBadges
		resp.BadgeURLs = &b
		w.Header().Set("X-Content-Verification", "verified-true")
		w.Header().Set("X-Verification-Standard", latest.Standard)
		w.Header().Set("X-Verification-Date", latest.Date)
	}
	writeJSON(w, http.StatusOK, resp)
}

type registerRequest struct {
	Subject  string `json:"subject"`
	Standard string `json:"standard"`
	Method   string `json:"method"`
	Date     string `json:"date"`
	Signer   string `json:"signer"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Standard == "" {
		req.Standard = model.SealOfTruth
	}
	c, err := model.NewSelfCertification(req.Subject, req.Standard, req.Method, req.Date, req.Signer,
		model.Attribution{Agent: req.Signer, AtTime: s.now().UTC().Format(time.RFC3339)})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ref, err := s.Store.PutCertification(r.Context(), c)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"uuid": ref.UUID, "version": ref.CID})
}

type obsoleteRequest struct {
	By           string   `json:"by"`
	Reason       string   `json:"reason"`
	SupersededBy []string `json:"superseded_by"`
}

func (s *Server) obsolete(w http.ResponseWriter, r *http.Request, uuid string) {
	var req obsoleteRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	cur, _, err := s.Store.GetCertification(r.Context(), uuid)
	if errors.Is(err, storage.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no such certification")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	next, err := cur.Obsoleted(req.By, req.Reason, req.SupersededBy)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	ref, err := s.Store.PutCertification(r.Context(), next)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"uuid": ref.UUID, "version": ref.CID})
}

func (s *Server) get(w http.ResponseWriter, r *http.Request, uuid string) {
	c, ref, err := s.Store.GetCertification(r.Context(), uuid)
	if errors.Is(err, storage.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "no such certification")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"version": ref.CID, "certification": c})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
