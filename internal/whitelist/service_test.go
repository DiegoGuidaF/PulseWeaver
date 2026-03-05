package whitelist

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"github.com/matryer/is"
)

func TestService_Regenerate_WritesIPsToFile(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	filePath, mockProvider, service := setupService(t, 100*time.Millisecond, nil, nil)
	mockProvider.ips = []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	err := service.Regenerate(ctx)
	is.NoErr(err)

	content, err := os.ReadFile(filePath)
	is.NoErr(err)

	expected := "@wallydex_allowlist {\n    remote_ip 192.168.1.1 192.168.1.2 192.168.1.3\n}\n"
	is.Equal(string(content), expected)
}

func TestService_Regenerate_EmptyListIsNonRoutableIP(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	filePath, mockProvider, service := setupService(t, 100*time.Millisecond, nil, nil)
	mockProvider.ips = []string{}

	err := service.Regenerate(ctx)
	is.NoErr(err)

	content, err := os.ReadFile(filePath)
	is.NoErr(err)
	is.Equal(string(content), "@wallydex_allowlist {\n    remote_ip 255.255.255.255\n}\n")
}

func TestService_Regenerate_AtomicWrite(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	filePath, mockProvider, service := setupService(t, 100*time.Millisecond, nil, nil)
	mockProvider.ips = []string{"192.168.1.1"}

	err := service.Regenerate(ctx)
	is.NoErr(err)

	tempPath := filePath + ".tmp"
	_, err = os.Stat(tempPath)
	is.True(os.IsNotExist(err))

	_, err = os.Stat(filePath)
	is.NoErr(err)
}

func TestService_Regenerate_ProviderError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	filePath, mockProvider, service := setupService(t, 100*time.Millisecond, nil, nil)
	testErr := errors.New("provider error")
	mockProvider.err = testErr

	err := service.Regenerate(ctx)
	is.True(err != nil)
	is.True(errors.Is(err, testErr))

	_, err = os.Stat(filePath)
	is.True(os.IsNotExist(err))
}

func TestService_Run_FirstSignalIsDebounced(t *testing.T) {
	is := is.New(t)
	mockProvider := newMockProvider()
	mockProvider.ips = []string{"192.168.1.1"}

	filePath, service, cancel, done := newRunningService(t, 200*time.Millisecond, mockProvider)
	start := time.Now()

	service.OnAddressEvent(context.Background(), device.AddressEvent{})

	mockProvider.waitForNoCall(t, whitelistDebounceDelay/2)
	firstCallAt := mockProvider.waitForCall(t)
	is.True(firstCallAt.Sub(start) >= whitelistDebounceDelay-(10*time.Millisecond))

	waitForFileExists(t, filePath, time.Second)

	content, err := os.ReadFile(filePath)
	is.NoErr(err)
	is.Equal(string(content), "@wallydex_allowlist {\n    remote_ip 192.168.1.1\n}\n")

	cancel()
	<-done
}

func TestService_Run_DebouncesEvents(t *testing.T) {
	mockProvider := newMockProvider()
	mockProvider.ips = []string{"192.168.1.1"}

	_, service, cancel, done := newRunningService(t, 20*time.Millisecond, mockProvider)

	service.OnAddressEvent(context.Background(), device.AddressEvent{})

	// Burst of additional events within the debounce window should coalesce
	service.OnAddressEvent(context.Background(), device.AddressEvent{})
	service.OnAddressEvent(context.Background(), device.AddressEvent{})
	service.OnAddressEvent(context.Background(), device.AddressEvent{})

	mockProvider.waitForCall(t)
	mockProvider.waitForNoCall(t, 2*whitelistDebounceDelay)

	cancel()
	<-done
}

func TestService_Run_ContextCancellationExitsCleanly(t *testing.T) {
	is := is.New(t)
	mockProvider := newMockProvider()
	mockProvider.ips = []string{"192.168.1.1"}

	_, _, cancel, done := newRunningService(t, 100*time.Millisecond, mockProvider)

	cancel()

	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}
}

func TestService_Run_RespectsRateLimitBetweenRuns(t *testing.T) {
	is := is.New(t)
	mockProvider := newMockProvider()
	mockProvider.ips = []string{"192.168.1.1"}

	rateLimit := 200 * time.Millisecond
	_, service, cancel, done := newRunningService(t, rateLimit, mockProvider)

	service.OnAddressEvent(context.Background(), device.AddressEvent{})
	firstCallAt := mockProvider.waitForCall(t)

	service.OnAddressEvent(context.Background(), device.AddressEvent{})
	mockProvider.waitForNoCall(t, rateLimit-(whitelistDebounceDelay/2))
	secondCallAt := mockProvider.waitForCall(t)
	is.True(secondCallAt.Sub(firstCallAt) >= rateLimit-(10*time.Millisecond))

	cancel()
	err := <-done
	is.NoErr(err)
}

func TestService_Run_ContinuesOnRegenerateError(t *testing.T) {
	is := is.New(t)
	callCount := 0

	mockProvider := newMockProvider()
	mockProvider.onCall = func() ([]string, error) {
		callCount++
		if callCount == 2 {
			return nil, errors.New("provider error")
		}
		return []string{"192.168.1.1"}, nil
	}

	_, service, cancel, done := newRunningService(t, 20*time.Millisecond, mockProvider)

	service.OnAddressEvent(context.Background(), device.AddressEvent{})
	mockProvider.waitForCall(t)

	service.OnAddressEvent(context.Background(), device.AddressEvent{})
	mockProvider.waitForCall(t)

	cancel()
	err := <-done
	is.NoErr(err)

	is.Equal(callCount, 2)
}

type mockChangeNotifier struct {
	calls   int
	lastCtx context.Context
}

func (m *mockChangeNotifier) NotifyChange(ctx context.Context) {
	m.calls++
	m.lastCtx = ctx
}

func TestService_Regenerate_EnqueuesNotificationOnChange(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	notifier := &mockChangeNotifier{}
	_, mockProvider, service := setupService(t, 100*time.Millisecond, nil, notifier)
	mockProvider.ips = []string{"192.168.1.1", "192.168.1.2"}

	err := service.Regenerate(ctx)
	is.NoErr(err)
	is.Equal(notifier.calls, 1)
	is.True(notifier.lastCtx != nil)
}

// mockEnabledIPsProvider is a synchronized mock implementation of EnabledIPsProvider.
type mockEnabledIPsProvider struct {
	ips      []string
	err      error
	onCall   func() ([]string, error)
	callChan chan time.Time
}

func newMockProvider() *mockEnabledIPsProvider {
	return &mockEnabledIPsProvider{
		callChan: make(chan time.Time, 10),
	}
}

func (m *mockEnabledIPsProvider) GetEnabledUniqueIPs(_ context.Context) ([]string, error) {
	select {
	case m.callChan <- time.Now():
	default:
	}

	if m.onCall != nil {
		return m.onCall()
	}
	return m.ips, m.err
}

// waitForCall blocks until GetEnabledUniqueIPs is called, or times out.
func (m *mockEnabledIPsProvider) waitForCall(t *testing.T) time.Time {
	t.Helper()
	select {
	case calledAt := <-m.callChan:
		return calledAt
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for GetEnabledUniqueIPs to be called")
		return time.Time{}
	}
}

// waitForNoCall verifies GetEnabledUniqueIPs is not called during the window.
func (m *mockEnabledIPsProvider) waitForNoCall(t *testing.T, d time.Duration) {
	t.Helper()
	select {
	case <-m.callChan:
		t.Fatal("unexpected GetEnabledUniqueIPs call")
	case <-time.After(d):
	}
}

func waitForFileExists(t *testing.T, filePath string, timeout time.Duration) {
	t.Helper()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	checkTicker := time.NewTicker(5 * time.Millisecond)
	defer checkTicker.Stop()

	for {
		if _, err := os.Stat(filePath); err == nil {
			return
		}

		select {
		case <-checkTicker.C:
		case <-deadline.C:
			t.Fatalf("timed out waiting for file %q to exist", filePath)
		}
	}
}

// setupService encapsulates common setup for synchronous Regenerate tests.
// If provider is nil, it creates a new mockProvider.
func setupService(t *testing.T, rateLimit time.Duration, provider *mockEnabledIPsProvider, notifier ChangeNotifier) (string, *mockEnabledIPsProvider, *Service) {
	t.Helper()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "whitelist.txt")

	if provider == nil {
		provider = newMockProvider()
	}
	if notifier == nil {
		notifier = &mockChangeNotifier{}
	}

	conf := config.ConfWhitelist{
		FilePath:  filePath,
		RateLimit: rateLimit,
	}

	service := NewService(provider, conf, notifier, slog.New(slog.DiscardHandler))

	return filePath, provider, service
}

// newRunningService encapsulates common setup for tests exercising Service.RunListener.
// It builds upon setupService by launching RunListener in a background goroutine.
// Returns the service so tests can trigger events via OnAddressEvent.
func newRunningService(t *testing.T, rateLimit time.Duration, provider *mockEnabledIPsProvider) (filePath string, service *Service, cancel context.CancelFunc, done <-chan error) {
	t.Helper()

	ctx, cancelCtx := context.WithCancel(context.Background())
	t.Cleanup(cancelCtx) // Prevent goroutine leaks if test fails early

	// Reuse the synchronous setup
	filePath, _, service = setupService(t, rateLimit, provider, nil)

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- service.RunListener(ctx)
	}()

	return filePath, service, cancelCtx, doneCh
}
