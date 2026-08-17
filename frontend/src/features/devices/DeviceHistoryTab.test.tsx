import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { HttpResponse, delay, http } from 'msw';
import { DeviceHistoryTab } from '@/features/devices/DeviceHistoryTab';
import { TEST_TIMEOUTS } from '@/test/constants';
import {
    createMockAddressHistoryTuningCandidate,
    createMockAddressHistoryTuningResponse,
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

        // "Enabled"/"Disabled" are event kinds, and the resulting address state
        // they imply is only stated when it disagrees — which it does not here.
        expect(screen.getAllByText('Enabled')).toHaveLength(1);
        expect(screen.getAllByText('Disabled')).toHaveLength(1);

        // Check sources (displayed via SOURCE_LABELS)
        expect(screen.getByText('Heartbeat')).toBeInTheDocument();
        expect(screen.getByText('Web UI')).toBeInTheDocument();
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
        expect(screen.getByText('No lease renewals in this window')).toBeInTheDocument();
    });

    it('renders chart section title', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Renewal timing vs. TTL')).toBeInTheDocument();
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

    // ─── Tuning strip when device-scoped ────────────────────────────────────

    it('reads out the TTL tuning summary for the device it is scoped to', async () => {
        server.use(
            addressHandlers.tuning.success(
                createMockAddressHistoryTuningResponse({
                    devices: [createMockAddressHistoryTuningCandidate({ device_name: 'nas-01' })],
                }),
            ),
        );

        renderTab();

        // The heading renders before the query resolves, so the row is what
        // says the readout actually arrived.
        await waitFor(
            () => {
                expect(screen.getByText('nas-01')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // A ranking would degenerate here, but the TTL readout is most useful on
        // the device already being viewed — only its filter link is redundant.
        expect(screen.getByText('TTL tuning')).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Filter by nas-01' })).not.toBeInTheDocument();
    });

    it('shows time range and auto-refresh controls', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Renewal timing vs. TTL')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // TimeRangePresetSelect options
        expect(screen.getByText('24h')).toBeInTheDocument();

        // AutoRefreshSelect should be present
        expect(screen.getByText('5s')).toBeInTheDocument();
    });
});
