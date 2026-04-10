package service

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/orchestration"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestManagedExecutionBackendSupported(t *testing.T) {
	t.Parallel()

	if !managedExecutionBackendSupported(string(orchestration.RuntimeBackendModal)) {
		t.Fatalf("expected modal backend to be supported")
	}

	for _, backend := range []string{
		string(orchestration.RuntimeBackendDaytona),
		string(orchestration.RuntimeBackendLocal),
	} {
		if managedExecutionBackendSupported(backend) {
			t.Fatalf("expected backend %q to remain unsupported", backend)
		}
	}
}

func TestFallbackManagedExecutionBackendPrefersModalWhenEnabled(t *testing.T) {
	t.Parallel()
	policy := db.RuntimePolicy{
		AllowModal:   true,
		AllowDaytona: true,
	}

	if got := fallbackManagedExecutionBackend(policy); got != string(orchestration.RuntimeBackendModal) {
		t.Fatalf("expected modal fallback backend, got %q", got)
	}
}
