package submission_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/pkg/bacen/submission"
)

type fakeSubmitter struct{ last submission.Payload }

func (f *fakeSubmitter) Submit(_ context.Context, p submission.Payload) (submission.Receipt, error) {
	if err := p.Validate(); err != nil {
		return submission.Receipt{}, err
	}
	f.last = p
	return submission.Receipt{ProtocolID: "P-1", Status: submission.StatusReceived}, nil
}

func (f *fakeSubmitter) Query(_ context.Context, id string) (submission.Receipt, error) {
	if id != "P-1" {
		return submission.Receipt{}, submission.ErrProtocolNotFound
	}
	return submission.Receipt{ProtocolID: id, Status: submission.StatusAccepted}, nil
}

var _ submission.Submitter = (*fakeSubmitter)(nil)

func TestPayloadValidate(t *testing.T) {
	t.Parallel()
	body := []byte("<Doc/>")
	cases := []struct {
		name string
		p    submission.Payload
		ok   bool
	}{
		{"dec", submission.Payload{Kind: submission.KindDEC, ReferenceID: "r1", Body: body}, true},
		{"sce-ied", submission.Payload{Kind: submission.KindSCEIED, ReferenceID: "r1", Body: body}, true},
		{"sce-credito", submission.Payload{Kind: submission.KindSCECredito, ReferenceID: "r1", Body: body}, true},
		{"sce-cbe", submission.Payload{Kind: submission.KindSCECBE, ReferenceID: "r1", Body: body}, true},
		{"unknown kind", submission.Payload{Kind: "RDE-ROF", ReferenceID: "r1", Body: body}, false},
		{"no reference", submission.Payload{Kind: submission.KindDEC, Body: body}, false},
		{"empty body", submission.Payload{Kind: submission.KindDEC, ReferenceID: "r1"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.p.Validate()
			if tc.ok {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, submission.ErrInvalidPayload)
		})
	}
}

func TestStatusTerminal(t *testing.T) {
	t.Parallel()
	assert.False(t, submission.StatusReceived.Terminal())
	assert.False(t, submission.StatusProcessing.Terminal())
	assert.True(t, submission.StatusAccepted.Terminal())
	assert.True(t, submission.StatusRejected.Terminal())
}

func TestSubmitterContract(t *testing.T) {
	t.Parallel()
	s := &fakeSubmitter{}
	r, err := s.Submit(context.Background(), submission.Payload{Kind: submission.KindDEC, ReferenceID: "r1", Body: []byte("<Doc/>")})
	require.NoError(t, err)
	assert.Equal(t, "P-1", r.ProtocolID)
	_, err = s.Query(context.Background(), "missing")
	assert.ErrorIs(t, err, submission.ErrProtocolNotFound)
}
