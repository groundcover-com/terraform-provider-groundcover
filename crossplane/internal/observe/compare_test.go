package observe

import (
	"context"
	"testing"
)

func TestYAMLUpToDate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		desired  string
		observed string
		want     bool
	}{
		{
			name:     "identical",
			desired:  "title: cpu\nmodel:\n  threshold: 5\n",
			observed: "title: cpu\nmodel:\n  threshold: 5\n",
			want:     true,
		},
		{
			name:     "key order differs only",
			desired:  "title: cpu\nmodel:\n  threshold: 5\n",
			observed: "model:\n  threshold: 5\ntitle: cpu\n",
			want:     true,
		},
		{
			name:     "server adds fields the user did not author",
			desired:  "title: cpu\nmodel:\n  threshold: 5\n",
			observed: "title: cpu\nmodel:\n  threshold: 5\nlink: https://app/x\nisProvisioned: true\n",
			want:     true,
		},
		{
			name:     "equivalent duration formatting",
			desired:  "title: cpu\nmodel:\n  for: 1h\n",
			observed: "title: cpu\nmodel:\n  for: 1h0m0s\n",
			want:     true,
		},
		{
			name:     "real value change is not suppressed",
			desired:  "title: cpu\nmodel:\n  threshold: 5\n",
			observed: "title: cpu\nmodel:\n  threshold: 10\n",
			want:     false,
		},
		{
			name:     "empty desired is treated as up to date",
			desired:  "",
			observed: "title: cpu\n",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YAMLUpToDate(ctx, tt.desired, tt.observed)
			if err != nil {
				t.Fatalf("YAMLUpToDate returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("YAMLUpToDate(%q, %q) = %v, want %v", tt.desired, tt.observed, got, tt.want)
			}
		})
	}
}

func TestHashUpToDate(t *testing.T) {
	tests := []struct {
		name     string
		recorded string
		remote   string
		want     bool
	}{
		{name: "equal hashes are up to date", recorded: "abc", remote: "abc", want: true},
		{name: "different hashes drift", recorded: "abc", remote: "def", want: false},
		{name: "no recorded baseline is not drift", recorded: "", remote: "def", want: true},
		{name: "no remote hash is not drift", recorded: "abc", remote: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HashUpToDate(tt.recorded, tt.remote); got != tt.want {
				t.Errorf("HashUpToDate(%q, %q) = %v, want %v", tt.recorded, tt.remote, got, tt.want)
			}
		})
	}
}
