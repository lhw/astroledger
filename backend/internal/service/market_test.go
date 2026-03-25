package service

import (
	"strings"
	"testing"
)

func TestValidateResolutionRequestMetadataAllowsHTTPAndHTTPS(t *testing.T) {
	t.Parallel()

	for _, link := range []string{"https://example.com/evidence", "http://example.com/patch"} {
		if err := validateResolutionRequestMetadata(link, "looks resolved"); err != nil {
			t.Fatalf("expected %q to be allowed, got %v", link, err)
		}
	}
}

func TestValidateResolutionRequestMetadataRejectsUnsafeSchemes(t *testing.T) {
	t.Parallel()

	err := validateResolutionRequestMetadata("javascript:alert(1)", "")
	if err == nil {
		t.Fatal("expected javascript scheme to be rejected")
	}
}

func TestValidateResolutionRequestMetadataRejectsOversizedNote(t *testing.T) {
	t.Parallel()

	err := validateResolutionRequestMetadata("https://example.com", strings.Repeat("a", 501))
	if err == nil {
		t.Fatal("expected oversized note to be rejected")
	}
}
