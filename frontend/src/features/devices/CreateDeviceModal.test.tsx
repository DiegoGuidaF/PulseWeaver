import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http, HttpResponse } from 'msw';
import { useLocation } from 'react-router-dom';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { CreateDeviceModal } from './CreateDeviceModal';
import { TEST_TIMEOUTS } from '@/test/constants';
import { deviceHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { createMockDevice } from '@/test/mocks/data';
import type { ComponentProps } from 'react';

type ModalProps = Partial<ComponentProps<typeof CreateDeviceModal>>;

function LocationDisplay() {
    const loc = useLocation();
    return <div data-testid="location">{loc.pathname}{loc.search}</div>;
}

function renderModal(props: ModalProps = {}) {
    const onClose = props.onClose ?? vi.fn();
    return renderWithProviders(
        <>
            <CreateDeviceModal opened={true} onClose={onClose} {...props} />
            <LocationDisplay />
        </>
    );
}

describe('CreateDeviceModal', () => {
    it('renders form with name input and submit button', () => {
        renderModal({});

        expect(screen.getByLabelText('Name')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /create device/i })).toBeInTheDocument();
    });

    it('shows validation error for empty name', async () => {
        const user = userEvent.setup();
        renderModal({});

        const input = screen.getByLabelText('Name');
        await user.click(screen.getByRole('button', { name: /create device/i }));

        await waitFor(
            () => {
                expect(input).toBeInvalid();
                expect(screen.getByText(/at least|too short|too small|required/i)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('shows loading state during submission', async () => {
        const user = userEvent.setup();

        server.use(
            http.post(endpoints.devices, async () => {
                await delay('infinite');
                return responses.created(createMockDevice());
            })
        );

        renderModal({});

        await user.type(screen.getByLabelText('Name'), 'Test Device');
        await user.click(screen.getByRole('button', { name: /create device/i }));

        expect(screen.getByRole('button', { name: /create device/i })).toBeDisabled();
    });

    it('calls onClose after successful creation', async () => {
        const user = userEvent.setup();
        const onClose = vi.fn();

        server.use(
            http.post(endpoints.devices, () =>
                HttpResponse.json(createMockDevice(), { status: 201 })
            )
        );

        renderModal({ onClose });

        await user.type(screen.getByLabelText('Name'), 'New Device');
        await user.click(screen.getByRole('button', { name: /create device/i }));

        await waitFor(
            () => {
                expect(onClose).toHaveBeenCalled();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('navigates to the new device workspace after successful creation', async () => {
        const user = userEvent.setup();

        server.use(
            http.post(endpoints.devices, () =>
                HttpResponse.json(createMockDevice({ id: 42 }), { status: 201 })
            )
        );

        renderModal({});

        await user.type(screen.getByLabelText('Name'), 'My New Device');
        await user.click(screen.getByRole('button', { name: /create device/i }));

        await waitFor(
            () => {
                const loc = screen.getByTestId('location').textContent ?? '';
                expect(loc).toContain('/devices/owners/');
                expect(loc).toContain('device=42');
                expect(loc).toContain('tab=addresses');
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('shows error toast on API error', async () => {
        const user = userEvent.setup();

        server.use(
            http.post(endpoints.devices, () => responses.serverError())
        );

        renderModal({});

        const input = screen.getByLabelText('Name');
        await user.type(input, 'Test Device');
        await user.click(screen.getByRole('button', { name: /create device/i }));

        await waitFor(
            () => {
                expect(screen.getByText(/error creating device/i)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );

        // Form should not be reset on error
        expect((input as HTMLInputElement).value).toBe('Test Device');
    });

    it('shows error message when device name is already in use (409)', async () => {
        const user = userEvent.setup();

        server.use(deviceHandlers.create.conflict());

        renderModal({});

        const input = screen.getByLabelText('Name');
        await user.type(input, 'Duplicate Name');
        await user.click(screen.getByRole('button', { name: /create device/i }));

        await waitFor(
            () => {
                expect(
                    screen.getByText(/error creating device/i)
                ).toBeInTheDocument();
                expect(
                    screen.getByText(/already exists/i)
                ).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );

        expect((input as HTMLInputElement).value).toBe('Duplicate Name');
    });
});
