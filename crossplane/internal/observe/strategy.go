package observe

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// Strategy decides whether a managed resource that upjet flagged as out of date is in
// fact semantically up to date. Implementations extract the relevant desired/observed
// values from the concrete managed resource and defer the comparison to the shared
// normalize logic.
type Strategy interface {
	// UpToDate reports whether the observed state should be treated as up to date.
	// Returning false leaves upjet's original observation untouched.
	UpToDate(ctx context.Context, mg resource.Managed) (bool, error)
}

// YAMLFields extracts the desired (authored) and observed YAML documents from a managed
// resource. ok is false when either side is unavailable (e.g. before creation), in which
// case no suppression is attempted.
type YAMLFields func(mg resource.Managed) (desiredYAML, observedYAML string, ok bool)

// HashFields extracts the recorded-baseline and current-remote content hashes from a
// managed resource. ok is false when the values are unavailable.
type HashFields func(mg resource.Managed) (recordedHash, remoteHash string, ok bool)

type yamlStrategy struct{ fields YAMLFields }

// NewYAMLStrategy builds a Strategy for resources whose drift is purely cosmetic YAML
// formatting (monitor, dashboard). fields adapts the concrete generated managed resource
// to the desired/observed YAML documents.
func NewYAMLStrategy(fields YAMLFields) Strategy { return yamlStrategy{fields: fields} }

func (s yamlStrategy) UpToDate(ctx context.Context, mg resource.Managed) (bool, error) {
	desired, observed, ok := s.fields(mg)
	if !ok {
		return false, nil
	}
	return YAMLUpToDate(ctx, desired, observed)
}

type hashStrategy struct{ fields HashFields }

// NewHashStrategy builds a Strategy for resources whose stored content is redacted on
// read and compared via a server-computed hash (connected_app).
func NewHashStrategy(fields HashFields) Strategy { return hashStrategy{fields: fields} }

func (s hashStrategy) UpToDate(_ context.Context, mg resource.Managed) (bool, error) {
	recorded, remote, ok := s.fields(mg)
	if !ok {
		return false, nil
	}
	return HashUpToDate(recorded, remote), nil
}
