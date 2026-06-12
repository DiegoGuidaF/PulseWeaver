import { describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { createMemoryRouter, Link, RouterProvider } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import { TEST_TIMEOUTS } from '@/test/constants';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';
import { setupUser } from '@/test/utils';

function DirtyPage({ dirty }: { dirty: boolean }) {
    useUnsavedChangesGuard(dirty);
    return <Link to="/other">Go to other</Link>;
}

function renderGuard(dirty: boolean) {
    const router = createMemoryRouter(
        [
            { path: '/', element: <DirtyPage dirty={dirty} /> },
            { path: '/other', element: <div>Other page</div> },
        ],
        { initialEntries: ['/'] },
    );
    return render(
        <MantineProvider>
            <ModalsProvider>
                <RouterProvider router={router} />
            </ModalsProvider>
        </MantineProvider>,
    );
}

describe('useUnsavedChangesGuard — in-app navigation blocking', () => {
    it('navigates immediately when there are no unsaved changes', async () => {
        const user = setupUser();
        renderGuard(false);

        await user.click(screen.getByRole('link', { name: /go to other/i }));

        expect(await screen.findByText('Other page')).toBeInTheDocument();
        expect(screen.queryByText(/discard unsaved changes/i)).not.toBeInTheDocument();
    });

    it('blocks navigation and stays on the page when "Keep editing" is chosen', async () => {
        const user = setupUser();
        renderGuard(true);

        await user.click(screen.getByRole('link', { name: /go to other/i }));

        const dialog = await screen.findByRole('dialog');
        expect(dialog).toHaveTextContent(/discard unsaved changes/i);

        await user.click(screen.getByRole('button', { name: /keep editing/i }));

        await waitFor(
            () => expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText('Other page')).not.toBeInTheDocument();
        expect(screen.getByRole('link', { name: /go to other/i })).toBeInTheDocument();
    });

    it('proceeds with navigation when "Discard changes" is confirmed', async () => {
        const user = setupUser();
        renderGuard(true);

        await user.click(screen.getByRole('link', { name: /go to other/i }));

        await screen.findByRole('dialog');
        await user.click(screen.getByRole('button', { name: /discard changes/i }));

        expect(await screen.findByText('Other page')).toBeInTheDocument();
    });
});
