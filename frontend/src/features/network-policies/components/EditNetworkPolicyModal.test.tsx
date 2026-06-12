import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { EditNetworkPolicyModal } from './EditNetworkPolicyModal';
import { renderWithProviders, setupUser } from '@/test/utils';
import { TEST_TIMEOUTS } from '@/test/constants';
import { createMockNetworkPolicyDetail } from '@/test/mocks/data';

function renderModal(overrides?: Parameters<typeof createMockNetworkPolicyDetail>[0]) {
    const policy = createMockNetworkPolicyDetail({
        name: 'Office',
        cidr: '192.168.1.0/24',
        description: 'HQ network',
        ...overrides,
    });
    const onUpdate = vi.fn();
    const onClose = vi.fn();
    renderWithProviders(
        <EditNetworkPolicyModal policy={policy} opened onClose={onClose} onUpdate={onUpdate} />,
    );
    return { onUpdate, onClose };
}

describe('EditNetworkPolicyModal', () => {
    it('prefills the form from the policy', () => {
        renderModal();

        expect(screen.getByLabelText(/name/i)).toHaveValue('Office');
        expect(screen.getByLabelText(/cidr range/i)).toHaveValue('192.168.1.0/24');
        expect(screen.getByLabelText(/description/i)).toHaveValue('HQ network');
    });

    it('saves edited fields via onUpdate', async () => {
        const user = setupUser();
        const { onUpdate } = renderModal();

        const name = screen.getByLabelText(/name/i);
        await user.clear(name);
        await user.type(name, 'Renamed');
        await user.click(screen.getByRole('button', { name: /save changes/i }));

        await waitFor(
            () => expect(onUpdate).toHaveBeenCalledTimes(1),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
        expect(onUpdate.mock.calls[0][0]).toMatchObject({ name: 'Renamed', cidr: '192.168.1.0/24' });
    });

    it('blocks save for a too-broad CIDR', async () => {
        const user = setupUser();
        const { onUpdate } = renderModal();

        const cidr = screen.getByLabelText(/cidr range/i);
        await user.clear(cidr);
        await user.type(cidr, '0.0.0.0/0');
        await user.click(screen.getByRole('button', { name: /save changes/i }));

        await waitFor(
            () => expect(screen.getByText(/too broad/i)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
        expect(onUpdate).not.toHaveBeenCalled();
    });
});
