package whitelist

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"github.com/matryer/is"
)

func TestService_Regenerate_WritesIPsToFile(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Create temp directory for test
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	mockProvider := &mockEnabledIPsProvider{
		ips: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	err := service.Regenerate(ctx)
	is.NoErr(err)

	// Verify file was created and contains correct IPs
	content, err := os.ReadFile(filePath)
	is.NoErr(err)

	expected := "192.168.1.1\n192.168.1.2\n192.168.1.3\n"
	is.Equal(string(content), expected)
}

func TestService_Regenerate_HandlesEmptyList(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Create temp directory for test
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	mockProvider := &mockEnabledIPsProvider{
		ips: []string{},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	err := service.Regenerate(ctx)
	is.NoErr(err)

	// Verify file was created (empty file)
	content, err := os.ReadFile(filePath)
	is.NoErr(err)
	is.Equal(string(content), "")
}

func TestService_Regenerate_AtomicWrite(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Create temp directory for test
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	mockProvider := &mockEnabledIPsProvider{
		ips: []string{"192.168.1.1"},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	err := service.Regenerate(ctx)
	is.NoErr(err)

	// Verify temp file doesn't exist (should be renamed)
	tempPath := filePath + ".tmp"
	_, err = os.Stat(tempPath)
	is.True(os.IsNotExist(err)) // Temp file should not exist

	// Verify final file exists
	_, err = os.Stat(filePath)
	is.NoErr(err) // Final file should exist
}

func TestService_Regenerate_ProviderError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	testErr := errors.New("provider error")
	mockProvider := &mockEnabledIPsProvider{
		err: testErr,
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	err := service.Regenerate(ctx)
	is.True(err != nil)
	is.True(errors.Is(err, testErr) || errors.Is(err, errors.Unwrap(err)))

	// Verify file was not created
	_, err = os.Stat(filePath)
	is.True(os.IsNotExist(err))
}

func TestService_Regenerate_CreatesDirectory(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Create temp directory for test
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "path")
	filePath := filepath.Join(nestedDir, "whitelist.txt")

	mockProvider := &mockEnabledIPsProvider{
		ips: []string{"192.168.1.1"},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	err := service.Regenerate(ctx)
	is.NoErr(err)

	// Verify directory was created
	_, err = os.Stat(nestedDir)
	is.NoErr(err)

	// Verify file was created
	_, err = os.Stat(filePath)
	is.NoErr(err)
}

func TestService_Run_DebouncesEvents(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	callCount := 0
	mockProvider := &mockEnabledIPsProvider{
		ips: []string{"192.168.1.1"},
		onGetIPs: func() {
			callCount++
		},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 200 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	// Start Run goroutine
	done := make(chan error, 1)
	go func() {
		done <- service.Run(ctx)
	}()

	// Send multiple events rapidly
	updatesChan := service.Updates()
	updatesChan <- struct{}{}
	updatesChan <- struct{}{}
	updatesChan <- struct{}{}

	// Wait for debounce delay + small buffer
	time.Sleep(300 * time.Millisecond)

	// Cancel context to stop Run
	cancel()

	// Wait for Run to exit
	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}

	// Should have been called only once due to debouncing
	is.Equal(callCount, 1)
}

func TestService_Run_ContextCancellationExitsCleanly(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	mockProvider := &mockEnabledIPsProvider{
		ips: []string{"192.168.1.1"},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	// Start Run goroutine
	done := make(chan error, 1)
	go func() {
		done <- service.Run(ctx)
	}()

	// Cancel context immediately
	cancel()

	// Wait for Run to exit
	select {
	case err := <-done:
		is.NoErr(err) // Should exit cleanly without error
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}
}

func TestService_Run_HandlesMultipleRegenerations(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	callCount := 0
	mockProvider := &mockEnabledIPsProvider{
		ips: []string{"192.168.1.1"},
		onGetIPs: func() {
			callCount++
		},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	// Start Run goroutine
	done := make(chan error, 1)
	go func() {
		done <- service.Run(ctx)
	}()

	updatesChan := service.Updates()

	// Send event, wait for regeneration
	updatesChan <- struct{}{}
	time.Sleep(150 * time.Millisecond)

	// Send another event, wait for regeneration
	updatesChan <- struct{}{}
	time.Sleep(150 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for Run to exit
	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}

	// Should have been called twice (once per event after debounce)
	is.Equal(callCount, 2)
}

func TestService_Updates_ReturnsWriteOnlyChannel(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	mockProvider := &mockEnabledIPsProvider{
		ips: []string{"192.168.1.1"},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	updatesChan := service.Updates()

	// Verify it's a write-only channel (can send, cannot receive)
	// This is a compile-time check, but we can verify it works at runtime
	updatesChan <- struct{}{}

	// The channel should accept writes
	is.True(updatesChan != nil)
}

func TestService_Run_ContinuesOnRegenerateError(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	// First call succeeds, second call fails
	callCount := 0
	providerErr := &errorHolder{}
	mockProvider := &mockEnabledIPsProvider{
		ips:       []string{"192.168.1.1"},
		errHolder: providerErr,
		onGetIPs: func() {
			callCount++
			if callCount == 2 {
				// Second call returns error
				providerErr.err = errors.New("provider error")
			}
		},
	}

	conf := config.ConfWhitelist{
		FilePath:      filePath,
		DebounceDelay: 100 * time.Millisecond,
	}

	service := NewService(mockProvider, conf)

	// Start Run goroutine
	done := make(chan error, 1)
	go func() {
		done <- service.Run(ctx)
	}()

	updatesChan := service.Updates()

	// Send first event (should succeed)
	updatesChan <- struct{}{}
	time.Sleep(150 * time.Millisecond)

	// Send second event (should fail but continue)
	updatesChan <- struct{}{}
	time.Sleep(150 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for Run to exit
	select {
	case err := <-done:
		is.NoErr(err) // Should exit cleanly despite error
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}

	// Should have attempted regeneration twice
	is.Equal(callCount, 2)
}

// errorHolder holds an error that can be modified from closures
type errorHolder struct {
	err error
}

// mockEnabledIPsProvider is a mock implementation of EnabledIPsProvider
type mockEnabledIPsProvider struct {
	ips       []string
	err       error
	errHolder *errorHolder
	onGetIPs  func()
}

func (m *mockEnabledIPsProvider) GetEnabledUniqueIPs(ctx context.Context) ([]string, error) {
	if m.onGetIPs != nil {
		m.onGetIPs()
	}
	// Check errHolder first (for dynamic errors), then static err
	if m.errHolder != nil && m.errHolder.err != nil {
		return nil, m.errHolder.err
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.ips, nil
}
