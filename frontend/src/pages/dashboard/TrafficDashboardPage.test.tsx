import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { TrafficDashboardPage } from '@/pages/dashboard/TrafficDashboardPage';
import { renderWithProviders } from '@/test/utils';

function renderPage() {
    return renderWithProviders(<TrafficDashboardPage />);
}

describe('TrafficDashboardPage', () => {
    it('renders the posture-and-traffic subtitle', () => {
        renderPage();
        expect(screen.getByText(/security posture and traffic/i)).toBeInTheDocument();
    });

    it('renders the time range selector', () => {
        renderPage();
        // TimeRangePresetSelect now lives on the traffic section inside DashboardView,
        // scoping the preset to traffic only.
        expect(screen.getByRole('combobox')).toBeInTheDocument();
    });
});
