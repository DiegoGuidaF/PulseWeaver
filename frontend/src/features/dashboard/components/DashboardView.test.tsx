import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { DashboardView } from './DashboardView';
import { TEST_TIMEOUTS } from '@/test/constants';
import { renderWithProviders, setupUser } from '@/test/utils';
import { server } from '@/test/setup';
import { dashboardHandlers } from '@/test/mocks/handlers';

describe('DashboardView', () => {
    it('renders stat cards with data from the API', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('150')).toBeInTheDocument();
                expect(screen.getByText('120')).toBeInTheDocument();
                expect(screen.getByText('30')).toBeInTheDocument();
                expect(screen.getByText('8')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders stat card labels', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('Total Requests')).toBeInTheDocument();
                // "Allowed" and "Denied" also appear in attribution / country tables.
                expect(screen.getAllByText('Allowed').length).toBeGreaterThan(0);
                expect(screen.getAllByText('Denied').length).toBeGreaterThan(0);
                expect(screen.getByText('Unique IPs')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('no longer renders the Avg Response Time card', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => expect(screen.getByText('Total Requests')).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText('Avg Response Time')).not.toBeInTheDocument();
    });

    it('renders traffic chart heading', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('Traffic over time')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders services chart heading', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('Requests by Service')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders top denied IPs table with data', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('Top Denied IPs')).toBeInTheDocument();
                expect(screen.getByText('203.0.113.42')).toBeInTheDocument();
                expect(screen.getByText('198.51.100.7')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders the posture strip with cards', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('Security posture')).toBeInTheDocument();
                expect(screen.getByText('Bypass users')).toBeInTheDocument();
                expect(screen.getByText('Locked-out users')).toBeInTheDocument();
                expect(screen.getByText('Bypass-check policies')).toBeInTheDocument();
                expect(screen.getByText('Shared IPs')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('surfaces pending host suggestions only when present', async () => {
        server.use(dashboardHandlers.posture({ pending_suggestion_count: 3 }));
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => expect(screen.getByText(/pending host suggestions to review/i)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders the three attribution tables with entity rows', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('By network policy')).toBeInTheDocument();
                expect(screen.getByText('By user')).toBeInTheDocument();
                expect(screen.getByText('By device')).toBeInTheDocument();
                expect(screen.getByText('Docker')).toBeInTheDocument();
                expect(screen.getByText('Diego Guida')).toBeInTheDocument();
                expect(screen.getByText('Workstation')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('marks a deleted attribution entity', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => expect(screen.getByText('Old Laptop')).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText('(deleted)')).toBeInTheDocument();
    });

    it('expands an attribution table to show the long tail', async () => {
        const user = setupUser();
        const many = Array.from({ length: 12 }, (_, i) => ({
            entity_id: i + 1,
            entity_name: `Device ${i + 1}`,
            allow_count: 100 - i,
            deny_count: 0,
        }));
        server.use(dashboardHandlers.attributionSplit({ device: many }));
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => expect(screen.getByText('Device 1')).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        // Collapsed: top 8 only — the 9th-ranked device is hidden.
        expect(screen.queryByText('Device 9')).not.toBeInTheDocument();

        await user.click(screen.getByRole('button', { name: /show all 12/i }));
        expect(screen.getByText('Device 9')).toBeInTheDocument();
    });

    it('shows the no-reconciliation caveat for attribution tables', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => expect(screen.getByText(/do not sum to total\s+traffic/i)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });
});
