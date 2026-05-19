import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { AddressHistoryPage } from '@/pages/address-history/AddressHistoryPage';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { TEST_TIMEOUTS } from '@/test/constants';
import { addressHandlers } from '@/test/mocks/handlers';
import { createMockAddressHistoryResponse, createMockAddressHistoryEvent } from '@/test/mocks/data';

function renderPage() {
    return renderWithProviders(<AddressHistoryPage />);
}

describe('AddressHistoryPage', () => {
    it('renders address history events from the API', async () => {
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    events: [
                        createMockAddressHistoryEvent({ id: 1, ip: '10.0.1.1', device_name: 'Lab Box' }),
                        createMockAddressHistoryEvent({ id: 2, ip: '10.0.1.2', device_name: 'Dev Laptop' }),
                    ],
                }),
            ),
        );
        renderPage();

        await waitFor(() => {
            expect(screen.getByText('10.0.1.1')).toBeInTheDocument();
            expect(screen.getByText('Lab Box')).toBeInTheDocument();
        }, { timeout: TEST_TIMEOUTS.MEDIUM });
    });

    it('shows empty state when there are no events', async () => {
        server.use(addressHandlers.history.empty());
        renderPage();

        await waitFor(() => {
            expect(screen.getByText(/no address events found/i)).toBeInTheDocument();
        }, { timeout: TEST_TIMEOUTS.MEDIUM });
    });
});
