import {beforeEach, describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import {DashboardPage} from '@/pages/DashboardPage';
import {TEST_TIMEOUTS} from '@/test/constants';
import {handlers} from '@/test/mocks/handlers.ts';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';

describe('DashboardPage', () => {
  beforeEach(() => {
    server.use(handlers.devices.getDeviceListHandler([]));
  });

  it('renders heading, create form, and empty device list', async () => {
    renderWithProviders(<DashboardPage/>);

    expect(screen.getByRole('heading', {name: 'WallyDic Manager'})).toBeInTheDocument();
    expect(screen.getByLabelText('New Device Name')).toBeInTheDocument();
    await waitFor(
      () => {
        expect(screen.getByText('No devices found.')).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
  });
});
