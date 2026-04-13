import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DeviceSettingsTab } from '@/features/devices/DeviceSettingsTab';
import { TEST_TIMEOUTS } from '@/test/constants';
import { renderWithProviders } from '@/test/utils';

const mockDevice = { name: 'My Router', api_key_prefix: 'rtr_', device_type: 'static' as const };

function renderTab() {
    return renderWithProviders(<DeviceSettingsTab deviceId={1} />);
}

function renderTabWithDevice() {
    return renderWithProviders(<DeviceSettingsTab deviceId={1} device={mockDevice} />);
}

describe('DeviceSettingsTab', () => {
    it('shows loading skeleton when device is not yet loaded', () => {
        renderTab();

        expect(screen.getByRole('button', { name: 'Regenerate API key' })).toBeDisabled();
        expect(screen.queryByText('Enabled')).not.toBeInTheDocument();
    });

    it('opens confirmation dialog when Regenerate API key is clicked', async () => {
        const user = userEvent.setup();
        // deviceHandlers.regenerateApiKey.success() is in defaultHandlers

        renderTabWithDevice();

        await user.click(screen.getByRole('button', { name: 'Regenerate API key' }));

        expect(screen.getByRole('dialog')).toBeInTheDocument();
        expect(screen.getByText(/Regenerate API key for/i)).toBeInTheDocument();
    });

    it('calls regenerate API and shows new key dialog on confirm', async () => {
        const user = userEvent.setup();
        // deviceHandlers.regenerateApiKey.success() is in defaultHandlers

        renderTabWithDevice();

        await user.click(screen.getByRole('button', { name: 'Regenerate API key' }));
        await user.click(screen.getByRole('button', { name: 'Regenerate' }));

        await waitFor(
            () => {
                expect(screen.getByText('API key regenerated — save your new key')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        expect(screen.getByDisplayValue('regenerated_key_abc123xyz789')).toBeInTheDocument();
    });
});
