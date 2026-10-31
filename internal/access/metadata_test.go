package access_test

import (
	"context"
	"testing"

	"aihub/internal/access"
)

func TestMetadataRoundTrip(t *testing.T) {
	want := access.Metadata{
		Principal:         access.PrincipalID,
		ProviderSessionID: "aihub-ses-0123456789abcdef0123456789abcdef",
		Host:              "localhost",
		Origin:            "https://example.com",
	}

	ctx := access.WithMetadata(context.Background(), want)
	got, ok := access.MetadataFrom(ctx)
	if !ok {
		t.Fatal("MetadataFrom() must find the envelope")
	}

	if got != want {
		t.Fatalf("MetadataFrom() = %+v, want %+v", got, want)
	}
}

func TestMetadataAbsent(t *testing.T) {
	if _, ok := access.MetadataFrom(context.Background()); ok {
		t.Fatal("MetadataFrom() must report absence on a plain context")
	}
}
