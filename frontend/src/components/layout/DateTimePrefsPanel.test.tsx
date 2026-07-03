import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders, setupUser } from '@/test/utils';
import { DateTimePrefsPanel } from '@/components/layout/DateTimePrefsPanel';
import { DATETIME_PREFS_KEY } from '@/lib/userPreferences';

describe('DateTimePrefsPanel', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('renders the date and time format controls with a preview', () => {
    renderWithProviders(<DateTimePrefsPanel />);

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
    const user = setupUser();
    renderWithProviders(<DateTimePrefsPanel />);

    await user.click(screen.getByText('DD/MM/YYYY'));

    await waitFor(() => {
      const stored = localStorage.getItem(DATETIME_PREFS_KEY);
      expect(stored).not.toBeNull();
      expect(JSON.parse(stored!).dateOrder).toBe('DMY');
    });
  });

  it('changing the time format control persists to localStorage', async () => {
    const user = setupUser();
    renderWithProviders(<DateTimePrefsPanel />);

    await user.click(screen.getByText('24-hour'));

    await waitFor(() => {
      const stored = localStorage.getItem(DATETIME_PREFS_KEY);
      expect(stored).not.toBeNull();
      expect(JSON.parse(stored!).timeFormat).toBe('24h');
    });
  });

  it('preview text updates live when prefs change', async () => {
    // Force a known 12h start state so the before/after comparison is deterministic
    localStorage.setItem(DATETIME_PREFS_KEY, JSON.stringify({ dateOrder: 'MDY', timeFormat: '12h' }));
    const user = setupUser();
    renderWithProviders(<DateTimePrefsPanel />);

    await user.click(screen.getByText('24-hour'));

    await waitFor(() => {
      const preview = screen.getByText(/Preview:/);
      expect(preview.textContent).not.toMatch(/AM|PM/i);
    });
  });
});
