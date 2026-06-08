import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateNetworkPolicyModal } from './CreateNetworkPolicyModal';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { TEST_TIMEOUTS } from '@/test/constants';
import { networkPolicyHandlers } from '@/test/mocks/handlers';

function renderModal() {
    const onCreated = vi.fn();
    const onClose = vi.fn();
    renderWithProviders(
        <CreateNetworkPolicyModal opened onClose={onClose} onCreated={onCreated} />,
    );
    return { onCreated, onClose };
}

describe('CreateNetworkPolicyModal — CIDR validation', () => {
    it('accepts a native IPv6 CIDR', async () => {
        const user = userEvent.setup();
        server.use(networkPolicyHandlers.create.success());
        const { onCreated } = renderModal();

        await user.type(screen.getByLabelText(/name/i), 'IPv6 Net');
        await user.type(screen.getByLabelText(/cidr range/i), '2001:db8::/32');
        await user.click(screen.getByRole('button', { name: /create policy/i }));

        await waitFor(
            () => expect(onCreated).toHaveBeenCalledTimes(1),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
        expect(screen.queryByText(/enter a valid cidr range/i)).not.toBeInTheDocument();
    });

    it('rejects a malformed CIDR', async () => {
        const user = userEvent.setup();
        const { onCreated } = renderModal();

        await user.type(screen.getByLabelText(/name/i), 'Bad Net');
        await user.type(screen.getByLabelText(/cidr range/i), 'not-a-cidr');
        await user.click(screen.getByRole('button', { name: /create policy/i }));

        await waitFor(
            () => expect(screen.getByText(/enter a valid cidr range/i)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
        expect(onCreated).not.toHaveBeenCalled();
    });
});
