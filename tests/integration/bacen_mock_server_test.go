//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/tests/integration"
)

func postSubmission(t *testing.T, url string, body any) (*http.Response, integration.MockReceipt) {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(url+"/v1/submissions", "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	var r integration.MockReceipt
	if resp.StatusCode == http.StatusAccepted {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&r))
	}
	return resp, r
}

func getSubmission(t *testing.T, url, id string) (*http.Response, integration.MockReceipt) {
	t.Helper()
	resp, err := http.Get(url + "/v1/submissions/" + id)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	var r integration.MockReceipt
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&r))
	}
	return resp, r
}

func TestBACENMockAcceptsEachKind(t *testing.T) {
	t.Parallel()
	m := integration.NewBACENMock(t)
	for _, kind := range []string{"DEC", "SCE-IED", "SCE-CREDITO", "SCE-CBE"} {
		t.Run(kind, func(t *testing.T) {
			resp, r := postSubmission(t, m.URL(), integration.MockSubmission{Kind: kind, ReferenceID: "ref-" + kind, Body: []byte("<Doc/>")})
			require.Equal(t, http.StatusAccepted, resp.StatusCode)
			assert.NotEmpty(t, r.ProtocolID)
			assert.Equal(t, "RECEIVED", r.Status)

			resp, q := getSubmission(t, m.URL(), r.ProtocolID)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "ACCEPTED", q.Status)
		})
	}
}

func TestBACENMockIdempotentByReference(t *testing.T) {
	t.Parallel()
	m := integration.NewBACENMock(t)
	in := integration.MockSubmission{Kind: "DEC", ReferenceID: "same", Body: []byte("<Doc/>")}
	_, a := postSubmission(t, m.URL(), in)
	_, b := postSubmission(t, m.URL(), in)
	assert.Equal(t, a.ProtocolID, b.ProtocolID)
}

func TestBACENMockRejectsInvalid(t *testing.T) {
	t.Parallel()
	m := integration.NewBACENMock(t)
	cases := map[string]integration.MockSubmission{
		"unknown kind": {Kind: "RDE-ROF", ReferenceID: "r", Body: []byte("<Doc/>")},
		"no reference": {Kind: "DEC", Body: []byte("<Doc/>")},
		"empty body":   {Kind: "DEC", ReferenceID: "r"},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			resp, _ := postSubmission(t, m.URL(), in)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestBACENMockRejectionAndUnknownProtocol(t *testing.T) {
	t.Parallel()
	m := integration.NewBACENMock(t)
	m.RejectReference("bad", "E042 invalid nature code")
	_, r := postSubmission(t, m.URL(), integration.MockSubmission{Kind: "SCE-IED", ReferenceID: "bad", Body: []byte("<Doc/>")})
	_, q := getSubmission(t, m.URL(), r.ProtocolID)
	assert.Equal(t, "REJECTED", q.Status)
	assert.Equal(t, "E042 invalid nature code", q.RejectionReason)

	resp, _ := getSubmission(t, m.URL(), "does-not-exist")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
