import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http } from 'msw';
import { DeviceHistoryTab } from '@/features/devices/DeviceHistoryTab';
import { TEST_TIMEOUTS } from '@/test/constants';
import { addressHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

function renderTab() {
    return renderWithProviders(<DeviceHistoryTab deviceId={1} />);
}

describe('DeviceHistoryTab', () => {
    it('shows loading skeleton while fetching', () => {
        server.use(
            http.get(endpoints.deviceAddressHistory, async () => {
                await delay('infinite');
                return responses.ok({ buckets: [], events: [] });
            })
        );

        renderTab();

        // Should not show content while loading
        expect(screen.queryByText('Address Activity')).not.toBeInTheDocument();
    });

    it('renders event log table with mock data', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Event log')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // Check event IPs are rendered (10.0.0.1 appears twice: enabled + disabled)
        expect(screen.getAllByText('10.0.0.1')).toHaveLength(2);
        expect(screen.getByText('10.0.0.2')).toBeInTheDocument();

        // Check status badges
        const enabledBadges = screen.getAllByText('Enabled');
        expect(enabledBadges.length).toBe(2);
        expect(screen.getByText('Disabled')).toBeInTheDocument();

        // Check source badges
        expect(screen.getByText('heartbeat')).toBeInTheDocument();
        expect(screen.getByText('manual')).toBeInTheDocument();
        expect(screen.getByText('expiry')).toBeInTheDocument();
    });

    it('shows empty state when no events', async () => {
        server.use(addressHandlers.history.empty());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('No events in this period')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByText('No activity in this period')).toBeInTheDocument();
    });

    it('renders time range selector with all options', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Address Activity')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        expect(screen.getByText('24h')).toBeInTheDocument();
        expect(screen.getByText('7 days')).toBeInTheDocument();
        expect(screen.getByText('30 days')).toBeInTheDocument();
    });

    it('changes time range when selector is clicked', async () => {
        const user = userEvent.setup();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Address Activity')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // Click 7 days — should trigger refetch (the component recalculates `from`)
        await user.click(screen.getByText('7 days'));

        // The component should still render (no crash on range change)
        await waitFor(
            () => {
                expect(screen.getByText('Event log')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
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
});
