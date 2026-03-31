import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { DevicesPage } from '@/pages/DevicesPage';
import { TEST_TIMEOUTS } from '@/test/constants';
import { deviceHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

describe('DevicesPage', () => {
    beforeEach(() => {
        server.use(deviceHandlers.list([]));
    });

    it('renders heading, create form, and empty device list', async () => {
        renderWithProviders(<DevicesPage />);

        expect(screen.getByRole('heading', { name: 'Devices', level: 1 })).toBeInTheDocument();
        expect(screen.getByLabelText('New Device Name')).toBeInTheDocument();
        await waitFor(
            () => {
                expect(screen.getByText('No devices found.')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });
});
