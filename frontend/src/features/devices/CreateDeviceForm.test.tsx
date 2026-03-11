import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http } from 'msw';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { CreateDeviceForm } from './CreateDeviceForm';
import { TEST_TIMEOUTS } from '@/test/constants';
import { deviceHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { createMockDevice } from '@/test/mocks/data';

describe('CreateDeviceForm', () => {
    it('renders form with input and submit button', () => {
        renderWithProviders(<CreateDeviceForm />);

        expect(screen.getByLabelText('New Device Name')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /add device/i })).toBeInTheDocument();
    });

    it('shows validation error for empty name', async () => {
        const user = userEvent.setup();
        renderWithProviders(<CreateDeviceForm />);

        const input = screen.getByRole('textbox', { name: /new device name/i });
        const submitButton = screen.getByRole('button', { name: /add device/i });
        await user.click(submitButton);

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

        renderWithProviders(<CreateDeviceForm />);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', { name: /add device/i });

        await user.type(input, 'Test Device');
        await user.click(submitButton);

        // Check loading state
        expect(screen.getByRole('button', { name: /creating/i })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /creating/i })).toBeDisabled();
    });

    it('successfully creates device, shows API key dialog, and resets form', async () => {
        const user = userEvent.setup();

        const clipboardSpy = vi
            .spyOn(navigator.clipboard, 'writeText')
            .mockResolvedValue(undefined);

        // defaultHandlers provides create.success() — no server.use() needed

        renderWithProviders(<CreateDeviceForm />);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', { name: /add device/i });

        await user.type(input, 'New Device');
        await user.click(submitButton);

        expect(
            screen.getByRole('dialog', {
                name: /device created — save your api key/i,
            }),
        ).toBeInTheDocument();

        expect(
            screen.getByDisplayValue(
                'test_api_key_12345678901234567890123456789012',
            ),
        ).toBeInTheDocument();

        const copyButton = screen.getByRole('button', { name: /copy/i });
        await user.click(copyButton);
        expect(clipboardSpy).toHaveBeenCalledWith(
            'test_api_key_12345678901234567890123456789012',
        );

        // Pressing escape doesn't close the dialog
        await user.keyboard('{Escape}');
        expect(
            screen.getByRole('dialog', {
                name: /device created — save your api key/i,
            }),
        ).toBeInTheDocument();

        const closeButton = screen.getByRole('button', { name: /i've saved it/i });
        await user.click(closeButton);

        await waitFor(
            () => {
                expect(
                    screen.queryByRole('dialog', {
                        name: /device created — save your api key/i,
                    }),
                ).not.toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        expect(screen.getByLabelText('New Device Name')).toHaveValue('');
        expect(screen.getByRole('button', { name: /add device/i })).toBeInTheDocument();
        clipboardSpy.mockRestore();
    });

    it('shows error toast on API error', async () => {
        const user = userEvent.setup();

        server.use(
            http.post(endpoints.devices, () => responses.serverError())
        );

        renderWithProviders(<CreateDeviceForm />);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', { name: /add device/i });

        await user.type(input, 'Test Device');
        await user.click(submitButton);

        // Wait for error toast to appear (user feedback is important to test)
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

        renderWithProviders(<CreateDeviceForm />);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', { name: /add device/i });

        await user.type(input, 'Duplicate Name');
        await user.click(submitButton);

        await waitFor(
            () => {
                expect(
                    screen.getByText(/error creating device/i)
                ).toBeInTheDocument();
                expect(
                    screen.getByText(/device name already in use/i)
                ).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );

        expect((input as HTMLInputElement).value).toBe('Duplicate Name');
    });
});
