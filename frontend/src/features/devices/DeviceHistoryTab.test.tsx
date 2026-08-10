import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { HttpResponse, delay, http } from 'msw';
import { DeviceHistoryTab } from '@/features/devices/DeviceHistoryTab';
import { TEST_TIMEOUTS } from '@/test/constants';
import {
    createMockAddressHistoryAtRiskDevice,
    createMockAddressHistoryHistogramResponse,
    createMockAddressHistoryResponse,
} from '@/test/mocks/data';
import { addressHandlers, endpoints } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders, setupUser } from '@/test/utils';

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
        expect(screen.queryByText('Devices at risk over time')).not.toBeInTheDocument();
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

        // Check status badges. "Enabled"/"Disabled" render in both the Status
        // column and the Event column (an enabled/disabled event_kind always
        // pairs with a matching is_enabled), so counts reflect both.
        const enabledBadges = screen.getAllByText('Enabled');
        expect(enabledBadges.length).toBe(3);
        expect(screen.getAllByText('Disabled').length).toBeGreaterThan(0);

        // Check source badges (displayed via SOURCE_LABELS)
        expect(screen.getByText('Heartbeat')).toBeInTheDocument();
        expect(screen.getByText('Manual')).toBeInTheDocument();
        expect(screen.getByText('Expiry')).toBeInTheDocument();
    });

    it('shows empty state when no events', async () => {
        server.use(addressHandlers.history.empty(), addressHandlers.historyHistogram.empty());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('No address events found.')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByText('No devices at risk in this period')).toBeInTheDocument();
    });

    it('renders chart section title', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Devices at risk over time')).toBeInTheDocument();
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

    // ─── Locked device filter ───────────────────────────────────────────────

    it('scopes both endpoints to the locked device', async () => {
        const eventsQueries: string[] = [];
        const histogramQueries: string[] = [];
        server.use(
            http.get(endpoints.addressHistory, ({ request }) => {
                eventsQueries.push(new URL(request.url).search);
                return HttpResponse.json(createMockAddressHistoryResponse());
            }),
            http.get(endpoints.addressHistoryHistogram, ({ request }) => {
                histogramQueries.push(new URL(request.url).search);
                return HttpResponse.json(createMockAddressHistoryHistogramResponse());
            })
        );

        renderTab();

        await waitFor(
            () => {
                expect(eventsQueries.length).toBeGreaterThan(0);
                expect(histogramQueries.length).toBeGreaterThan(0);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // The lock is an ordinary filter param, so it must reach both endpoints —
        // a histogram missing it would chart the whole fleet next to one device's rows.
        expect(eventsQueries[0]).toContain('device_id=1');
        expect(histogramQueries[0]).toContain('device_id=1');
    });

    it('offers no removable chip or chooser entry for the locked device', async () => {
        const user = setupUser();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('10.0.0.1')).toHaveLength(2);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // A lock is not a user filter, so it never appears as a clearable chip.
        expect(screen.queryByText('Device:')).not.toBeInTheDocument();

        // Nor as a column the chooser could bring back.
        await user.click(screen.getByRole('button', { name: 'Columns' }));
        expect(await screen.findByRole('checkbox', { name: 'IP' })).toBeInTheDocument();
        expect(screen.queryByRole('checkbox', { name: 'Device' })).not.toBeInTheDocument();
    });

    // ─── At-risk strip hidden when device-scoped ────────────────────────────

    it('does not render the at-risk strip, even when the histogram carries ranked devices', async () => {
        server.use(
            addressHandlers.historyHistogram.success(
                createMockAddressHistoryHistogramResponse({
                    at_risk_devices: [createMockAddressHistoryAtRiskDevice({ device_name: 'nas-01' })],
                }),
            ),
        );

        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('10.0.0.1')).toHaveLength(2);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // The ranking degenerates to the one device already being viewed, so
        // the strip is dropped entirely rather than shown with a single row.
        expect(screen.queryByText('nas-01')).not.toBeInTheDocument();
        expect(screen.queryByText('At-risk devices')).not.toBeInTheDocument();
    });

    it('shows time range and auto-refresh controls', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Devices at risk over time')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // TimeRangePresetSelect options
        expect(screen.getByText('24h')).toBeInTheDocument();

        // AutoRefreshSelect should be present
        expect(screen.getByText('5s')).toBeInTheDocument();
    });
});
