import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { DeviceCreatePane } from '@/features/devices/DeviceCreatePane';
import { TEST_TIMEOUTS } from '@/test/constants';
import { renderWithProviders, setupUser } from '@/test/utils';

function renderPane(onCreated = vi.fn()) {
    renderWithProviders(
        <DeviceCreatePane
            ownerId={1}
            ownerName="Bob"
            onCancel={vi.fn()}
            onCreated={onCreated}
        />,
    );
    return { onCreated };
}

describe('DeviceCreatePane', () => {
    it('locks the owner and offers the three credential choices', () => {
        renderPane();

        expect(screen.getByText('New device for Bob')).toBeInTheDocument();
        expect(screen.getByText('owner: Bob')).toBeInTheDocument();
        expect(screen.getByRole('radio', { name: /None/i })).toBeChecked();
        expect(screen.getByRole('radio', { name: /API key/i })).toBeInTheDocument();
        expect(screen.getByRole('radio', { name: /Pairing code/i })).toBeInTheDocument();
    });

    it('creates a credential-less device and hands off to the detail', async () => {
        const user = setupUser();
        const { onCreated } = renderPane();

        await user.type(screen.getByLabelText(/Name/i), 'Office Printer');
        await user.click(screen.getByRole('button', { name: 'Create device' }));

        await waitFor(
            () => expect(onCreated).toHaveBeenCalledWith(1, 'addresses'),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it('reveals the minted key once for an API-key device', async () => {
        const user = setupUser();
        const { onCreated } = renderPane();

        await user.type(screen.getByLabelText(/Name/i), 'Phone');
        await user.click(screen.getByRole('radio', { name: /API key/i }));
        await user.click(screen.getByRole('button', { name: 'Create device' }));

        await waitFor(
            () => {
                expect(
                    screen.getByDisplayValue('test_api_key_12345678901234567890123456789012'),
                ).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
        expect(onCreated).not.toHaveBeenCalled();

        await user.click(screen.getByRole('button', { name: /open device/i }));
        expect(onCreated).toHaveBeenCalledWith(1, 'addresses');
    });
});
