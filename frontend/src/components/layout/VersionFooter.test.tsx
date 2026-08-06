import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { versionHandlers } from '@/test/mocks/handlers';
import { VersionFooter } from '@/components/layout/VersionFooter';

describe('VersionFooter', () => {
    it('renders the version with a shortened commit', async () => {
        renderWithProviders(<VersionFooter />);

        await waitFor(() => {
            expect(screen.getByText('v1.2.3 · abcdef1')).toBeInTheDocument();
        });
    });

    it('omits the commit when the build is un-stamped', async () => {
        server.use(
            versionHandlers.get.success({ version: 'dev', commit: 'unknown' }),
        );
        renderWithProviders(<VersionFooter />);

        await waitFor(() => {
            expect(screen.getByText('dev')).toBeInTheDocument();
        });
        expect(screen.queryByText(/unknown/)).not.toBeInTheDocument();
    });

    // App chrome must not surface a load failure for something this peripheral.
    it('renders nothing when the version query fails', async () => {
        server.use(versionHandlers.get.serverError());
        renderWithProviders(
            <div data-testid="slot">
                <VersionFooter />
            </div>,
        );

        await waitFor(() => {
            expect(screen.getByTestId('slot')).toBeEmptyDOMElement();
        });
    });
});
