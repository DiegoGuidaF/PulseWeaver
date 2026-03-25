import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { DashboardView } from './DashboardView';
import { TEST_TIMEOUTS } from '@/test/constants';
import { renderWithProviders } from '@/test/utils';

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
                expect(screen.getByText('Allowed')).toBeInTheDocument();
                expect(screen.getByText('Denied')).toBeInTheDocument();
                expect(screen.getByText('Unique IPs')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders traffic chart heading', async () => {
        renderWithProviders(<DashboardView />);

        await waitFor(
            () => {
                expect(screen.getByText('Traffic Over Time')).toBeInTheDocument();
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
});
