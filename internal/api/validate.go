package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"gnomatix/validatorium/internal/autopublish"
	"gnomatix/validatorium/internal/evaluate"
	"gnomatix/validatorium/internal/storage"
)

// Validator evaluates one working-store candidate with the evaluator its
// method names and publishes it when the result establishes validity.
// *autopublish.Publisher implements it.
type Validator interface {
	Run(ctx context.Context, premiseUUID string, scope map[string]interface{}) (autopublish.Outcome, error)
}

// ValidationView is the answer to POST /api/premises/{uuid}/validate.
type ValidationView struct {
	UUID           string `json:"uuid"`
	Result         string `json:"result"`
	Method         string `json:"method"`
	Detail         string `json:"detail,omitempty"`
	Published      bool   `json:"published"`
	Version        string `json:"version,omitempty"`
	AttestationCID string `json:"attestation_cid,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type validateRequest struct {
	// Scope is the applicable context passed to the evaluator and recorded
	// on the attestation (data-model.md §6).
	Scope map[string]interface{} `json:"scope"`
}

// validatePremise launches evaluation of a candidate: anyone may ask for a
// premise to be evaluated; only the evaluation result decides publication.
func (s *Server) validatePremise(w http.ResponseWriter, r *http.Request, uuid string) {
	if s.Validator == nil {
		writeErr(w, http.StatusNotImplemented, "validation needs a server started with a global store (val api -publish)")
		return
	}
	var req validateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	o, err := s.Validator.Run(r.Context(), uuid, req.Scope)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		writeErr(w, http.StatusNotFound, "no such candidate")
		return
	case errors.Is(err, evaluate.ErrNoMethod):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	case err != nil:
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	v := ValidationView{UUID: uuid, Published: o.Published, Reason: o.Reason, Detail: o.Evaluation.Detail}
	if o.Attestation != nil {
		v.Result, v.Method = string(o.Attestation.Result), o.Attestation.Method
	}
	if o.Published {
		v.Version, v.AttestationCID = o.Promotion.Version.CID, o.Promotion.AttestationCID
	}
	writeJSON(w, http.StatusOK, v)
}
