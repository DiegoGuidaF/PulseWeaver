import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { delay, http } from 'msw';
import { DeviceHistoryTab } from '@/features/devices/DeviceHistoryTab';
import { TEST_TIMEOUTS } from '@/test/constants';
import { addressHandlers, endpoints } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

function renderTab() {
    return renderWithProviders(<DeviceHistoryTab deviceId={1} />);
}

describe('DeviceHistoryTab', () => {
    it('shows loading skeleton while fetching', () => {
        server.use(
            http.get(endpoints.addressHistory, async () => {
                await delay('infinite');
            })
        );

        renderTab();

        // Should not show data content while loading
        expect(screen.queryByText('Active IPs over time')).not.toBeInTheDocument();
    });

    it('renders event table with mock data', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('10.0.0.1')).toHaveLength(2);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        expect(screen.getByText('10.0.0.2')).toBeInTheDocument();

        // Check status badges
        const enabledBadges = screen.getAllByText('Enabled');
        expect(enabledBadges.length).toBe(2);
        expect(screen.getByText('Disabled')).toBeInTheDocument();

        // Check source badges (displayed via SOURCE_LABELS)
        expect(screen.getByText('Heartbeat')).toBeInTheDocument();
        expect(screen.getByText('Manual')).toBeInTheDocument();
        expect(screen.getByText('Expiry')).toBeInTheDocument();
    });

    it('shows empty state when no events', async () => {
        server.use(addressHandlers.history.empty());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('No address events found.')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByText('No activity in this period')).toBeInTheDocument();
    });

    it('renders chart section title', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Active IPs over time')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('does not show device column when locked to a device', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('10.0.0.1')).toHaveLength(2);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // Device column should be hidden since deviceId is locked
        const headers = screen.getAllByRole('columnheader');
        const deviceHeader = headers.find((h) => h.textContent?.includes('Device'));
        expect(deviceHeader).toBeUndefined();
    });

    it('shows time range and auto-refresh controls', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Active IPs over time')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // TimeRangePresetSelect options
        expect(screen.getByText('24h')).toBeInTheDocument();

        // AutoRefreshSelect should be present
        expect(screen.getByText('5s')).toBeInTheDocument();
    });
});
