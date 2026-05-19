import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { TrafficDashboardPage } from '@/pages/dashboard/TrafficDashboardPage';
import { renderWithProviders } from '@/test/utils';

function renderPage() {
    return renderWithProviders(<TrafficDashboardPage />);
}

describe('TrafficDashboardPage', () => {
    it('renders the traffic overview subtitle', () => {
        renderPage();
        expect(screen.getByText(/traffic overview/i)).toBeInTheDocument();
    });

    it('renders the time range selector', () => {
        renderPage();
        // The TimeRangePresetSelect renders a combobox/select for the preset
        expect(screen.getByRole('combobox')).toBeInTheDocument();
    });
});
