import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DeviceProvisioningPage } from '@/pages/devices/DeviceProvisioningPage';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { TEST_TIMEOUTS } from '@/test/constants';
import { provisioningHandlers } from '@/test/mocks/handlers';
import { createMockPendingRegistration } from '@/test/mocks/data';

function renderPage() {
    return renderWithProviders(<DeviceProvisioningPage />);
}

describe('DeviceProvisioningPage', () => {
    // ─── Happy path ──────────────────────────────────────────────────────────────

    describe('happy path', () => {
        it('renders the page heading and subtitle', () => {
            renderPage();
            expect(screen.getByRole('heading', { name: 'Device Provisioning', level: 1 })).toBeInTheDocument();
            expect(screen.getByText(/generate setup codes/i)).toBeInTheDocument();
        });

        it('shows empty-state message when no pending invites exist', async () => {
            server.use(provisioningHandlers.list.empty());
            renderPage();

            await waitFor(() => {
                expect(screen.getByText(/no pending invites/i)).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });

        it('renders a row for each pending registration', async () => {
            server.use(
                provisioningHandlers.list.success([
                    createMockPendingRegistration({ id: 1, device_name: 'Office Laptop' }),
                    createMockPendingRegistration({ id: 2, device_name: 'Home Server' }),
                ]),
            );
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Office Laptop')).toBeInTheDocument();
                expect(screen.getByText('Home Server')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── Create invite modal ──────────────────────────────────────────────────────

    describe('create invite modal', () => {
        it('opens the create invite modal when the button is clicked', async () => {
            const user = userEvent.setup();
            renderPage();

            await user.click(screen.getByRole('button', { name: /create invite/i }));

            await waitFor(() => {
                expect(screen.getByRole('dialog')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });

        it('closes the modal when cancel is clicked', async () => {
            const user = userEvent.setup();
            renderPage();

            await user.click(screen.getByRole('button', { name: /create invite/i }));
            await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());

            await user.click(screen.getByRole('button', { name: /cancel/i }));

            await waitFor(() => {
                expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });
    });
});
