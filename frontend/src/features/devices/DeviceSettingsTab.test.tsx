import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DeviceSettingsTab } from '@/features/devices/DeviceSettingsTab';
import { TEST_TIMEOUTS } from '@/test/constants';
import { renderWithProviders } from '@/test/utils';
import { server } from '@/test/setup';
import { deviceHandlers } from '@/test/mocks/handlers';

const mockDeviceWithKey = { name: 'My Router', api_key_prefix: 'rtr_' };
const mockDeviceNoKey = { name: 'My Router', api_key_prefix: null };

function renderTab() {
    return renderWithProviders(<DeviceSettingsTab deviceId={1} />);
}

function renderTabWithKey() {
    return renderWithProviders(<DeviceSettingsTab deviceId={1} device={mockDeviceWithKey} />);
}

function renderTabNoKey() {
    return renderWithProviders(<DeviceSettingsTab deviceId={1} device={mockDeviceNoKey} />);
}

describe('DeviceSettingsTab', () => {
    it('shows no API key action before the device is loaded', () => {
        renderTab();

        expect(screen.queryByRole('button', { name: 'Generate key' })).not.toBeInTheDocument();
        expect(screen.queryByRole('button', { name: /Regenerate/i })).not.toBeInTheDocument();
    });

    it('shows the Generate key button when the device has no key', () => {
        renderTabNoKey();

        expect(screen.getByRole('button', { name: 'Generate key' })).not.toBeDisabled();
        expect(screen.queryByRole('button', { name: /Regenerate/i })).not.toBeInTheDocument();
    });

    it('shows the Regenerate and Remove buttons when the device has a key', () => {
        renderTabWithKey();

        expect(screen.getByRole('button', { name: /Regenerate/i })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Remove key' })).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Generate key' })).not.toBeInTheDocument();
    });

    it('opens the confirmation dialog when Regenerate is clicked', async () => {
        const user = userEvent.setup();
        // deviceHandlers.regenerateApiKey.success() is in defaultHandlers

        renderTabWithKey();

        await user.click(screen.getByRole('button', { name: /Regenerate/i }));

        expect(screen.getByRole('dialog')).toBeInTheDocument();
        expect(screen.getByText(/Regenerate API key for/i)).toBeInTheDocument();
    });

    it('calls regenerate API and shows new key dialog on confirm', async () => {
        const user = userEvent.setup();
        // deviceHandlers.regenerateApiKey.success() is in defaultHandlers

        renderTabWithKey();

        await user.click(screen.getByRole('button', { name: /Regenerate/i }));
        await user.click(screen.getByRole('button', { name: 'Regenerate' }));

        await waitFor(
            () => {
                expect(screen.getByText('API key generated — save it')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        expect(screen.getByDisplayValue('regenerated_key_abc123xyz789')).toBeInTheDocument();
    });

    it('opens the delete confirmation dialog when Remove key is clicked', async () => {
        const user = userEvent.setup();

        renderTabWithKey();

        await user.click(screen.getByRole('button', { name: 'Remove key' }));

        expect(screen.getByRole('dialog')).toBeInTheDocument();
        expect(screen.getByText(/Remove API key for/i)).toBeInTheDocument();
    });

    it('calls delete API on confirm and closes the modal', async () => {
        const user = userEvent.setup();
        server.use(deviceHandlers.deleteApiKey.success());

        renderTabWithKey();

        await user.click(screen.getByRole('button', { name: 'Remove key' }));
        await user.click(screen.getByRole('button', { name: 'Delete key' }));

        await waitFor(
            () => {
                expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });
});
