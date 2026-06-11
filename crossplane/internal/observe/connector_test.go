package observe

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
)

// fakeClient is an ExternalClient whose Observe result is fixed; Create/Update/Delete/
// Disconnect are promoted from NopClient.
type fakeClient struct {
	managed.NopClient
	obs          managed.ExternalObservation
	err          error
	observeCalls int
}

func (f *fakeClient) Observe(context.Context, resource.Managed) (managed.ExternalObservation, error) {
	f.observeCalls++
	return f.obs, f.err
}

// fakeConnector returns a fixed ExternalClient.
type fakeConnector struct {
	ec  managed.ExternalClient
	err error
}

func (c fakeConnector) Connect(context.Context, resource.Managed) (managed.ExternalClient, error) {
	return c.ec, c.err
}

// stubStrategy returns a fixed decision, recording whether it was consulted.
type stubStrategy struct {
	upToDate bool
	err      error
	called   bool
}

func (s *stubStrategy) UpToDate(context.Context, resource.Managed) (bool, error) {
	s.called = true
	return s.upToDate, s.err
}

func TestConnectorConnectPropagatesError(t *testing.T) {
	connectErr := errors.New("connect boom")
	c := NewConnector(fakeConnector{err: connectErr}, &stubStrategy{}, nil)

	if _, err := c.Connect(context.Background(), &fake.Managed{}); !errors.Is(err, connectErr) {
		t.Fatalf("Connect error = %v, want %v", err, connectErr)
	}
}

func TestObserve(t *testing.T) {
	exists := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}

	tests := []struct {
		name          string
		innerObs      managed.ExternalObservation
		innerErr      error
		strategy      *stubStrategy
		wantUpToDate  bool
		wantStrategy  bool // strategy should have been consulted
		wantObsErrNil bool
	}{
		{
			name:          "inner error passes through without consulting strategy",
			innerObs:      managed.ExternalObservation{},
			innerErr:      errors.New("observe boom"),
			strategy:      &stubStrategy{},
			wantStrategy:  false,
			wantObsErrNil: false,
		},
		{
			name:          "resource absent is passed through untouched",
			innerObs:      managed.ExternalObservation{ResourceExists: false},
			strategy:      &stubStrategy{upToDate: true},
			wantUpToDate:  false,
			wantStrategy:  false,
			wantObsErrNil: true,
		},
		{
			name:          "already up to date skips the strategy",
			innerObs:      managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			strategy:      &stubStrategy{upToDate: false},
			wantUpToDate:  true,
			wantStrategy:  false,
			wantObsErrNil: true,
		},
		{
			name:          "cosmetic drift is suppressed",
			innerObs:      exists,
			strategy:      &stubStrategy{upToDate: true},
			wantUpToDate:  true,
			wantStrategy:  true,
			wantObsErrNil: true,
		},
		{
			name:          "real drift is preserved",
			innerObs:      exists,
			strategy:      &stubStrategy{upToDate: false},
			wantUpToDate:  false,
			wantStrategy:  true,
			wantObsErrNil: true,
		},
		{
			name:          "strategy error preserves raw observation",
			innerObs:      exists,
			strategy:      &stubStrategy{err: errors.New("compare boom")},
			wantUpToDate:  false,
			wantStrategy:  true,
			wantObsErrNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := &fakeClient{obs: tt.innerObs, err: tt.innerErr}
			c := NewConnector(fakeConnector{ec: inner}, tt.strategy, nil)

			ec, err := c.Connect(context.Background(), &fake.Managed{})
			if err != nil {
				t.Fatalf("Connect returned error: %v", err)
			}

			obs, err := ec.Observe(context.Background(), &fake.Managed{})
			if tt.wantObsErrNil && err != nil {
				t.Fatalf("Observe returned unexpected error: %v", err)
			}
			if !tt.wantObsErrNil && err == nil {
				t.Fatalf("Observe expected an error, got nil")
			}
			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}
			if tt.strategy.called != tt.wantStrategy {
				t.Errorf("strategy consulted = %v, want %v", tt.strategy.called, tt.wantStrategy)
			}
			if inner.observeCalls != 1 {
				t.Errorf("inner Observe calls = %d, want 1", inner.observeCalls)
			}
		})
	}
}

// TestStrategiesAdaptManagedResource exercises the field-extractor seam used by the
// generated controllers, confirming the YAML and hash strategies wire through to the
// shared comparison logic and honor the "fields unavailable" guard.
func TestStrategiesAdaptManagedResource(t *testing.T) {
	ctx := context.Background()

	yaml := NewYAMLStrategy(func(resource.Managed) (string, string, bool) {
		return "title: cpu\nmodel:\n  threshold: 5\n", "model:\n  threshold: 5\ntitle: cpu\n", true
	})
	if ok, err := yaml.UpToDate(ctx, &fake.Managed{}); err != nil || !ok {
		t.Errorf("yaml strategy UpToDate = (%v, %v), want (true, nil)", ok, err)
	}

	yamlMissing := NewYAMLStrategy(func(resource.Managed) (string, string, bool) { return "", "", false })
	if ok, err := yamlMissing.UpToDate(ctx, &fake.Managed{}); err != nil || ok {
		t.Errorf("yaml strategy with missing fields = (%v, %v), want (false, nil)", ok, err)
	}

	hash := NewHashStrategy(func(resource.Managed) (string, string, bool) { return "abc", "abc", true })
	if ok, err := hash.UpToDate(ctx, &fake.Managed{}); err != nil || !ok {
		t.Errorf("hash strategy UpToDate = (%v, %v), want (true, nil)", ok, err)
	}
}
