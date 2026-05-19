import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { authHandlers } from '@/test/mocks/handlers';
import { SettingsPage } from '@/pages/settings/SettingsPage';
import { AuthProvider } from '@/features/auth/AuthContext';
import { DATETIME_PREFS_KEY } from '@/lib/userPreferences';
import { TEST_TIMEOUTS } from '@/test/constants';

function renderSettingsPage() {
  return renderWithProviders(
    <AuthProvider>
      <SettingsPage />
    </AuthProvider>,
  );
}

async function switchToPreferencesTab(user: ReturnType<typeof userEvent.setup>) {
  await waitFor(
    () => { expect(screen.getByRole('tab', { name: 'Preferences' })).toBeInTheDocument(); },
    { timeout: TEST_TIMEOUTS.SHORT },
  );
  await user.click(screen.getByRole('tab', { name: 'Preferences' }));
}

describe('SettingsPage — Date & Time preferences', () => {
  beforeEach(() => {
    localStorage.clear();
    server.use(authHandlers.me.success());
  });

  it('renders the Date & Time card with current preference values', async () => {
    const user = userEvent.setup();
    renderSettingsPage();

    await switchToPreferencesTab(user);

    await waitFor(
      () => {
        expect(screen.getByText('Date & Time')).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    // Date format segmented control options
    expect(screen.getByText('MM/DD/YYYY')).toBeInTheDocument();
    expect(screen.getByText('DD/MM/YYYY')).toBeInTheDocument();
    // Time format segmented control options
    expect(screen.getByText('12-hour')).toBeInTheDocument();
    expect(screen.getByText('24-hour')).toBeInTheDocument();
    // Preview text should be present
    expect(screen.getByText(/Preview:/)).toBeInTheDocument();
  });

  it('changing the date order control persists to localStorage', async () => {
    const user = userEvent.setup();
    renderSettingsPage();

    await switchToPreferencesTab(user);

    await waitFor(
      () => { expect(screen.getByText('Date & Time')).toBeInTheDocument(); },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );

    await user.click(screen.getByText('DD/MM/YYYY'));

    await waitFor(() => {
      const stored = localStorage.getItem(DATETIME_PREFS_KEY);
      expect(stored).not.toBeNull();
      const parsed = JSON.parse(stored!);
      expect(parsed.dateOrder).toBe('DMY');
    });
  });

  it('changing the time format control persists to localStorage', async () => {
    const user = userEvent.setup();
    renderSettingsPage();

    await switchToPreferencesTab(user);

    await waitFor(
      () => { expect(screen.getByText('Date & Time')).toBeInTheDocument(); },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    await user.click(screen.getByText('24-hour'));

    await waitFor(() => {
      const stored = localStorage.getItem(DATETIME_PREFS_KEY);
      expect(stored).not.toBeNull();
      const parsed = JSON.parse(stored!);
      expect(parsed.timeFormat).toBe('24h');
    });
  });

  it('preview text updates live when prefs change', async () => {
    // Force a known 12h start state so the before/after comparison is deterministic
    localStorage.setItem(DATETIME_PREFS_KEY, JSON.stringify({ dateOrder: 'MDY', timeFormat: '12h' }));
    const user = userEvent.setup();
    renderSettingsPage();

    await switchToPreferencesTab(user);

    await waitFor(
      () => { expect(screen.getByText(/Preview:/)).toBeInTheDocument(); },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    // Switch to 24-hour and verify the preview no longer contains AM/PM
    await user.click(screen.getByText('24-hour'));

    await waitFor(() => {
      const preview = screen.getByText(/Preview:/);
      expect(preview.textContent).not.toMatch(/AM|PM/i);
    });
  });
});
