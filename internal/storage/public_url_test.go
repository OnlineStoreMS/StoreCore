package storage

import "testing"

func TestPublicURLResolverRewritesHost(t *testing.T) {
	r := &PublicURLResolver{
		baseURL:   "https://osms.zfcycle.com/minio/storecore",
		keyPrefix: "attachments",
	}
	in := "http://192.168.3.41:9100/storecore/attachments/stores/logo/20260711213225_2fe8f73e.png"
	want := "https://osms.zfcycle.com/minio/storecore/attachments/stores/logo/20260711213225_2fe8f73e.png"
	if got := r.Resolve(in); got != want {
		t.Fatalf("Resolve() = %q, want %q", got, want)
	}
}

func TestPublicURLResolverKeepsExternal(t *testing.T) {
	r := &PublicURLResolver{
		baseURL:   "https://osms.zfcycle.com/minio/storecore",
		keyPrefix: "attachments",
	}
	in := "https://cdn.example.com/logo.png"
	if got := r.Resolve(in); got != in {
		t.Fatalf("Resolve() = %q, want unchanged %q", got, in)
	}
}
