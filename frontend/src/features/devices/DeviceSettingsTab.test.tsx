import {beforeEach, describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {delay} from 'msw';
import {DeviceSettingsTab} from '@/features/devices/DeviceSettingsTab';
import {createMockDeviceAddressLeaseRule} from '@/test/mocks/data';
import {TEST_TIMEOUTS} from '@/test/constants';
import {handlers, responses} from '@/test/mocks/handlers.ts';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';

const mockDevice = { name: 'My Router', api_key_prefix: 'rtr_' };

function renderTab() {
  return renderWithProviders(<DeviceSettingsTab deviceId={1}/>);
}

function renderTabWithDevice() {
  return renderWithProviders(<DeviceSettingsTab deviceId={1} device={mockDevice}/>);
}

describe('DeviceSettingsTab', () => {
  beforeEach(() => {
    server.use(
      handlers.rules.getDeviceAddressLeaseRuleHandler(
        createMockDeviceAddressLeaseRule({enabled: true, ttl_seconds: 3600})
      )
    );
  });

  it('shows loading skeleton', () => {
    server.use(
      handlers.rules.getDeviceAddressLeaseRuleHandler(undefined, async () => {
        await delay('infinite');
        return responses.ok(createMockDeviceAddressLeaseRule());
      })
    );

    renderTab();

    expect(screen.queryByText('Enabled')).not.toBeInTheDocument();
    expect(screen.queryByText(/disabled/i)).not.toBeInTheDocument();
  });

  it('shows disabled state when no rule (404)', async () => {
    server.use(handlers.rules.getDeviceAddressLeaseRuleHandler(null));

    renderTab();

    await waitFor(
      () => {
        expect(screen.getByText(/Auto-expiry is currently/i)).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    expect(screen.getByRole('button', {name: 'Enable auto-expiry'})).toBeInTheDocument();
    expect(screen.queryByRole('button', {name: 'Turn off auto-expiry'})).not.toBeInTheDocument();
    expect(screen.getByRole('spinbutton', {name: /expires after/i})).toHaveValue(5);
    expect(screen.getByRole('combobox', {name: /unit/i})).toHaveValue('minutes');
  });

  it('shows enabled state with TTL', async () => {
    renderTab();

    await waitFor(
      () => {
        expect(screen.getByText('Status:')).toBeInTheDocument();
        expect(screen.getByText('Enabled')).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    expect(screen.getByText('1 hour')).toBeInTheDocument();
    expect(screen.getByRole('button', {name: 'Change TTL'})).toBeInTheDocument();
    expect(screen.getByRole('button', {name: 'Turn off auto-expiry'})).toBeInTheDocument();
    expect(screen.queryByRole('button', {name: 'Save'})).not.toBeInTheDocument();
  });

  it('enables auto-expiry and shows toast', async () => {
    const user = userEvent.setup();
    server.use(
      handlers.rules.getDeviceAddressLeaseRuleHandler(null),
      handlers.rules.putDeviceAddressLeaseRuleHandler()
    );

    renderTab();

    await waitFor(
      () => {
        expect(screen.getByRole('button', {name: 'Enable auto-expiry'})).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    await user.click(screen.getByRole('button', {name: 'Enable auto-expiry'}));

    await waitFor(
      () => {
        expect(screen.getByText('Address lease rule saved')).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.MEDIUM}
    );
  });

  it('edits TTL and shows toast', async () => {
    const user = userEvent.setup();
    server.use(handlers.rules.putDeviceAddressLeaseRuleHandler());

    renderTab();

    await waitFor(
      () => {
        expect(screen.getByRole('button', {name: 'Change TTL'})).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    await user.click(screen.getByRole('button', {name: 'Change TTL'}));

    const valueInput = screen.getByRole('spinbutton', {name: /expires after/i});
    await user.clear(valueInput);
    await user.type(valueInput, '2');
    await user.selectOptions(screen.getByRole('combobox', {name: /unit/i}), 'days');
    await user.click(screen.getByRole('button', {name: 'Save'}));

    await waitFor(
      () => {
        expect(screen.getByText('Address lease rule saved')).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.MEDIUM}
    );
    expect(screen.queryByRole('button', {name: 'Cancel'})).not.toBeInTheDocument();
  });

  it('cancels TTL edit', async () => {
    const user = userEvent.setup();
    renderTab();

    await waitFor(
      () => {
        expect(screen.getByRole('button', {name: 'Change TTL'})).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    await user.click(screen.getByRole('button', {name: 'Change TTL'}));
    await user.click(screen.getByRole('button', {name: 'Cancel'}));

    expect(screen.queryByRole('button', {name: 'Cancel'})).not.toBeInTheDocument();
    expect(screen.getByRole('button', {name: 'Change TTL'})).toBeInTheDocument();
    expect(screen.getByRole('button', {name: 'Turn off auto-expiry'})).toBeInTheDocument();
    expect(screen.queryByText('Address lease rule saved')).not.toBeInTheDocument();
  });

  it('turns off auto-expiry', async () => {
    const user = userEvent.setup();
    server.use(handlers.rules.deleteDeviceAddressLeaseRuleHandler);

    renderTab();

    await waitFor(
      () => {
        expect(screen.getByRole('button', {name: 'Turn off auto-expiry'})).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
    await user.click(screen.getByRole('button', {name: 'Turn off auto-expiry'}));

    await waitFor(
      () => {
        expect(screen.getByText('Address lease rule disabled')).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.MEDIUM}
    );
  });

  it('opens confirmation dialog when Regenerate API key is clicked', async () => {
    const user = userEvent.setup();
    server.use(handlers.rules.getDeviceAddressLeaseRuleHandler(null));

    renderTabWithDevice();

    await user.click(screen.getByRole('button', {name: 'Regenerate API key'}));

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText(/Regenerate API key for/i)).toBeInTheDocument();
  });

  it('calls regenerate API and shows new key dialog on confirm', async () => {
    const user = userEvent.setup();
    server.use(
      handlers.rules.getDeviceAddressLeaseRuleHandler(null),
      handlers.devices.regenerateApiKeyHandler(),
    );

    renderTabWithDevice();

    await user.click(screen.getByRole('button', {name: 'Regenerate API key'}));
    await user.click(screen.getByRole('button', {name: 'Regenerate'}));

    await waitFor(
      () => {
        expect(screen.getByText('API key regenerated — save your new key')).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.MEDIUM}
    );
    expect(screen.getByDisplayValue('regenerated_key_abc123xyz789')).toBeInTheDocument();
  });

  it('shows error on fetch failure', async () => {
    server.use(
      handlers.rules.getDeviceAddressLeaseRuleHandler(undefined, () => responses.serverError())
    );

    renderTab();

    await waitFor(
      () => {
        expect(screen.getByText(/Error loading rule:/i)).toBeInTheDocument();
      },
      {timeout: TEST_TIMEOUTS.SHORT}
    );
  });
});
