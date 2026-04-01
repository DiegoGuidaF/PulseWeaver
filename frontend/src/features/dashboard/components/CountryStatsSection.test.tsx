import { describe, expect, it } from 'vitest';
import { screen, waitFor, fireEvent } from '@testing-library/react';
import { CountryStatsSection } from './CountryStatsSection';
import { TEST_TIMEOUTS } from '@/test/constants';
import { renderWithProviders } from '@/test/utils';

describe('CountryStatsSection', () => {
    it('renders the Access Map heading', async () => {
        renderWithProviders(<CountryStatsSection />);

        await waitFor(
            () => expect(screen.getByText('Access Map')).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders the Top Countries heading', async () => {
        renderWithProviders(<CountryStatsSection />);

        await waitFor(
            () => expect(screen.getByText('Top Countries')).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders metric toggle with Denied and Total options', async () => {
        renderWithProviders(<CountryStatsSection />);

        await waitFor(
            () => {
                expect(screen.getByText('Denied')).toBeInTheDocument();
                expect(screen.getByText('Total')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders country data from API in the top countries table', async () => {
        renderWithProviders(<CountryStatsSection />);

        await waitFor(
            () => {
                // Text is rendered as "🇺🇸 United States" — use regex
                expect(screen.getByText(/United States/)).toBeInTheDocument();
                expect(screen.getByText(/Germany/)).toBeInTheDocument();
                expect(screen.getByText(/China/)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('default metric sorts table by denied count', async () => {
        renderWithProviders(<CountryStatsSection />);

        await waitFor(
            () => {
                const rows = screen.getAllByRole('row').slice(1);
                // China has 70 denied, US has 20, DE has 5
                expect(rows[0]).toHaveTextContent('China');
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('switching to Total metric re-sorts the table', async () => {
        renderWithProviders(<CountryStatsSection />);

        // Wait for data to load
        await waitFor(
            () => screen.getByText(/China/),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        fireEvent.click(screen.getAllByText('Total')[0]);

        await waitFor(
            () => {
                const rows = screen.getAllByRole('row').slice(1);
                // US has 100 total, China has 75, DE has 50
                expect(rows[0]).toHaveTextContent('United States');
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('renders the world map SVG when data loads', async () => {
        renderWithProviders(<CountryStatsSection />);

        await waitFor(
            () => {
                expect(screen.getByRole('img')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });
});
