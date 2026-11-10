//go:build integration

// Package integration holds helpers for the integration suite.
package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockSubmission is the body the BACEN mock accepts on POST /v1/submissions.
type MockSubmission struct {
	Kind        string `json:"kind"`
	ReferenceID string `json:"reference_id"`
	Body        []byte `json:"body"`
}

// MockReceipt is what the mock returns for a submission or a status query.
type MockReceipt struct {
	ProtocolID      string    `json:"protocol_id"`
	Status          string    `json:"status"`
	RejectionReason string    `json:"rejection_reason,omitempty"`
	ReceivedAt      time.Time `json:"received_at"`
}

var mockKinds = map[string]bool{"DEC": true, "SCE-IED": true, "SCE-CREDITO": true, "SCE-CBE": true}

// BACENMock is an in-process stand-in for the BACEN submission gateway.
// A submission answers RECEIVED; the first query moves it to ACCEPTED, or to
// REJECTED when its reference was registered with RejectReference.
type BACENMock struct {
	srv     *httptest.Server
	mu      sync.Mutex
	seq     int
	byRef   map[string]string
	byID    map[string]*MockReceipt
	rejects map[string]string
}

// NewBACENMock starts the mock and closes it when t ends.
func NewBACENMock(t testing.TB) *BACENMock {
	t.Helper()
	m := &BACENMock{byRef: map[string]string{}, byID: map[string]*MockReceipt{}, rejects: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/submissions", m.submit)
	mux.HandleFunc("GET /v1/submissions/{id}", m.query)
	m.srv = httptest.NewServer(mux)
	t.Cleanup(m.srv.Close)
	return m
}

// URL is the mock's base URL.
func (m *BACENMock) URL() string { return m.srv.URL }

// RejectReference makes the submission with this reference end REJECTED.
func (m *BACENMock) RejectReference(ref, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rejects[ref] = reason
}

func (m *BACENMock) submit(w http.ResponseWriter, r *http.Request) {
	var in MockSubmission
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "malformed json", http.StatusBadRequest)
		return
	}
	switch {
	case !mockKinds[in.Kind]:
		http.Error(w, "unsupported kind", http.StatusBadRequest)
		return
	case strings.TrimSpace(in.ReferenceID) == "":
		http.Error(w, "reference_id required", http.StatusBadRequest)
		return
	case len(in.Body) == 0:
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	id, ok := m.byRef[in.ReferenceID]
	if !ok {
		m.seq++
		id = fmt.Sprintf("BCB-%s-%06d", in.Kind, m.seq)
		m.byRef[in.ReferenceID] = id
		m.byID[id] = &MockReceipt{ProtocolID: id, Status: "RECEIVED", ReceivedAt: time.Now().UTC()}
	}
	rec := *m.byID[id]
	if reason, bad := m.rejects[in.ReferenceID]; bad {
		m.byID[id].RejectionReason = reason
	}
	m.mu.Unlock()
	writeJSON(w, http.StatusAccepted, rec)
}

func (m *BACENMock) query(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	rec, ok := m.byID[r.PathValue("id")]
	if ok && rec.Status == "RECEIVED" {
		rec.Status = "ACCEPTED"
		if rec.RejectionReason != "" {
			rec.Status = "REJECTED"
		}
	}
	var out MockReceipt
	if ok {
		out = *rec
	}
	m.mu.Unlock()
	if !ok {
		http.Error(w, "protocol not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
