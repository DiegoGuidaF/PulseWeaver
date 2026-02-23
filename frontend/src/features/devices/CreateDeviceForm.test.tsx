import {describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {delay} from 'msw';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';
import {CreateDeviceForm} from './CreateDeviceForm';
import {TEST_TIMEOUTS} from '@/test/constants';
import {handlers, responses} from "@/test/mocks/handlers.ts";
import {createMockDevice} from "@/test/mocks/data.ts";

describe('CreateDeviceForm', () => {
    it('renders form with input and submit button', () => {
        renderWithProviders(<CreateDeviceForm/>);

        expect(screen.getByLabelText('New Device Name')).toBeInTheDocument();
        expect(screen.getByRole('button', {name: /add device/i})).toBeInTheDocument();
    });

    it('shows validation error for empty name', async () => {
        const user = userEvent.setup();
        renderWithProviders(<CreateDeviceForm/>);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', {name: /add device/i});
        await user.click(submitButton);

        await waitFor(() => {
            // Check that input is marked as invalid
            expect(input).toHaveAttribute('aria-invalid', 'true');
            // Check for any error message text (zod validation messages vary)
            const errorMessage = screen.getByRole('textbox', {name: /new device name/i}).closest('div')?.querySelector('p');
            expect(errorMessage).toBeInTheDocument();
        });
    });

    it('shows loading state during submission', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.devices.createDeviceHandler(undefined, async () => {
                await delay('infinite');
                return responses.created(createMockDevice());
            })
        );

        renderWithProviders(<CreateDeviceForm/>);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', {name: /add device/i});

        await user.type(input, 'Test Device');
        await user.click(submitButton);

        // Check loading state
        expect(screen.getByRole('button', {name: /creating/i})).toBeInTheDocument();
        expect(screen.getByRole('button', {name: /creating/i})).toBeDisabled();
    });

    it('successfully creates device, shows API key dialog, and resets form', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.devices.createDeviceHandler()
        );

        renderWithProviders(<CreateDeviceForm/>);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', {name: /add device/i});

        await user.type(input, 'New Device');
        await user.click(submitButton);

        // Dialog should open with the API key that is only returned on creation
        const dialog = await screen.findByRole('dialog', {
            name: /device created — save your api key/i,
        });
        expect(dialog).toBeInTheDocument();
        expect(
            screen.getByDisplayValue(
                'test_api_key_12345678901234567890123456789012',
            ),
        ).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /copy/i })).toBeInTheDocument();

        // Close the dialog
        const closeButton = screen.getByRole('button', {name: /i've saved it/i});
        await user.click(closeButton);

        // After closing, the form should be visible and reset
        await waitFor(() => {
            expect(screen.getByLabelText('New Device Name')).toHaveValue('');
        }, {timeout: TEST_TIMEOUTS.MEDIUM});

        expect(screen.getByRole('button', {name: /add device/i})).toBeInTheDocument();
    });

    it('shows error toast on API error', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.devices.createDeviceHandler(undefined, () => {
                return responses.serverError();
            })
        );

        renderWithProviders(<CreateDeviceForm/>);

        const input = screen.getByLabelText('New Device Name');
        const submitButton = screen.getByRole('button', {name: /add device/i});

        await user.type(input, 'Test Device');
        await user.click(submitButton);

        // Wait for error toast to appear (user feedback is important to test)
        await waitFor(() => {
            expect(screen.getByText(/error creating device/i)).toBeInTheDocument();
        }, {timeout: TEST_TIMEOUTS.MEDIUM});

        // Form should not be reset on error
        expect((input as HTMLInputElement).value).toBe('Test Device');
    });
});
