import {beforeEach, describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {delay} from 'msw';
import {DeviceDetailPage} from '@/pages/DeviceDetailPage';
import {createMockDevice, createMockDeviceAddressLeaseRule} from '@/test/mocks/data';
import {TEST_TIMEOUTS} from '@/test/constants';
import {handlers, responses} from '@/test/mocks/handlers.ts';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';

function renderPage(route = '/devices/1') {
  return renderWithProviders(<DeviceDetailPage/>, {
    initialEntries: [route],
    path: '/devices/:deviceId',
  });
}

describe('DeviceDetailPage', () => {
  beforeEach(() => {
    server.use(
      handlers.devices.getDeviceHandler(
        createMockDevice({name: 'My Router', api_key_prefix: 'rtr_'})
      ),
      handlers.addresses.getAddressListHandler([]),
      handlers.rules.getDeviceAddressLeaseRuleHandler(null)
    );
  });

  it('redirects for non-numeric deviceId', async () => {
    renderPage('/devices/abc');

    await waitFor(
      () => {
        expect(screen.queryByText('My Router')).not.toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    expect(screen.queryByText(/API key prefix/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', {name: /addresses/i})).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', {name: /settings & rules/i})).not.toBeInTheDocument();
  });

  it('shows loading skeleton', () => {
    server.use(
      handlers.devices.getDeviceHandler(undefined, async () => {
        await delay('infinite');
        return responses.ok(createMockDevice({name: 'My Router', api_key_prefix: 'rtr_'}));
      })
    );

    renderPage();

    expect(screen.queryByText('My Router')).not.toBeInTheDocument();
  });

  it('shows device header after load', async () => {
    renderPage();

    await waitFor(
      () => {
        expect(screen.getByRole('heading', {name: 'My Router'})).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    expect(screen.getByRole('link', {name: /back to devices/i})).toBeInTheDocument();
  });

  it('shows error when device fetch fails', async () => {
    server.use(
      handlers.devices.getDeviceHandler(undefined, () => responses.serverError())
    );

    renderPage();

    await waitFor(
      () => {
        expect(screen.getByText(/Error loading device:/i)).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
  });

  it('switches to Settings tab', async () => {
    const user = userEvent.setup();
    server.use(
      handlers.rules.getDeviceAddressLeaseRuleHandler(
        createMockDeviceAddressLeaseRule({enabled: true, ttl_seconds: 3600})
      )
    );

    renderPage();

    await waitFor(
      () => {
        expect(screen.getByRole('heading', {name: 'My Router'})).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    await user.click(screen.getByRole('tab', {name: /settings & rules/i}));

    expect(screen.getByText('Auto-expiry rule')).toBeVisible();
    expect(screen.queryByText('Add IP address')).not.toBeInTheDocument();
  });
});
