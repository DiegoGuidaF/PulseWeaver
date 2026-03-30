//go:build test

package geoip

import (
	"bytes"
	"compress/gzip"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matryer/is"
)

// makeGZIP wraps data in a gzip archive.
func makeGZIP(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

// makeServer returns an httptest.Server that serves gzip'd payload for all paths.
func makeServer(t *testing.T, payload []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
}

func nopLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestEnsureFile_FreshCache(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "test.mmdb")

	// Write a fresh file (mod time = now).
	is.NoErr(os.WriteFile(dst, []byte("mmdb data"), 0o644))
	is.NoErr(os.Chtimes(dst, time.Now(), time.Now()))

	// A server that must NOT be hit.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("download attempted for a fresh file")
	}))
	defer srv.Close()

	path, err := syncFile(context.Background(), dst, srv.URL, nopLogger())
	is.NoErr(err)
	is.Equal(path, dst)
}

func TestEnsureFile_StaleCache_DownloadSucceeds(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "test.mmdb")

	// Write a stale file (mod time 31 days ago).
	is.NoErr(os.WriteFile(dst, []byte("old data"), 0o644))
	staleTime := time.Now().Add(-31 * 24 * time.Hour)
	is.NoErr(os.Chtimes(dst, staleTime, staleTime))

	// Serve a valid gzip'd payload.
	newData := []byte("new mmdb data")
	srv := makeServer(t, makeGZIP(t, newData))
	defer srv.Close()

	path, err := syncFile(context.Background(), dst, srv.URL, nopLogger())
	is.NoErr(err)
	is.Equal(path, dst)

	got, err := os.ReadFile(dst)
	is.NoErr(err)
	is.Equal(got, newData)
}

func TestEnsureFile_NoCache_DownloadFails(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "test.mmdb")

	// No cached file, server unreachable.
	_, err := syncFile(context.Background(), dst+"_nonexistent", "http://127.0.0.1:0/nope", nopLogger())
	is.True(err != nil)
}

func TestEnsureFile_StaleCache_DownloadFails_UsesStale(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "test.mmdb")

	// Write a stale file.
	is.NoErr(os.WriteFile(dst, []byte("stale data"), 0o644))
	staleTime := time.Now().Add(-31 * 24 * time.Hour)
	is.NoErr(os.Chtimes(dst, staleTime, staleTime))

	// Server that returns 503.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	// Should return stale path without error (fail-open).
	path, err := syncFile(context.Background(), dst, srv.URL, nopLogger())
	is.NoErr(err)
	is.Equal(path, dst)
}

func TestSyncDBIPFiles_CreatesDir(t *testing.T) {
	is := is.New(t)
	dir := filepath.Join(t.TempDir(), "subdir", "geoip")

	// Dir doesn't exist yet; a failing server means download will fail and
	// return an error (no stale file), but the dir creation itself must succeed.
	_, _, err := SyncDBIPFiles(context.Background(), dir, nopLogger())
	// Error expected (no server running) — but dir must have been created.
	_ = err
	_, statErr := os.Stat(dir)
	is.NoErr(statErr)
}
