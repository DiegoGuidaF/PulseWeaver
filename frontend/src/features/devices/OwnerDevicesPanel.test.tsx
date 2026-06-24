import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { ROUTES } from '@/lib/routes';
import { OwnerDevicesPanel } from '@/features/devices/OwnerDevicesPanel';
import type { DeviceListEntry, DeviceListOwner } from '@/lib/api';
import { DeviceState, UserRole } from '@/lib/api';
import { TEST_TIMEOUTS } from '@/test/constants';
import { deviceHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders, setupUser } from '@/test/utils';
import { createMockDeviceOwnerGroup } from '@/test/mocks/data';

const defaultOwner: DeviceListOwner = {
    id: 1,
    username: 'testuser',
    display_name: 'Test User',
    role: UserRole.USER,
    bypass_host_check: false,
    host_groups: [],
    device_count: 1,
    live_address_count: 0,
};

const defaultDevice: DeviceListEntry = {
    id: 1,
    name: 'Test Device',
    state: DeviceState.HEALTHY,
    live_address_count: 0,
    rules: [],
    created_at: '2024-01-01T00:00:00Z',
};

interface PanelProps {
    owner?: DeviceListOwner;
    devices?: DeviceListEntry[];
    selectedDeviceId?: number;
    onSelectDevice?: (id: number) => void;
    onAddDevice?: () => void;
}

function renderPanel(props: PanelProps = {}) {
    const onSelectDevice = props.onSelectDevice ?? vi.fn();
    const onAddDevice = props.onAddDevice ?? vi.fn();
    const {
        owner = defaultOwner,
        devices = [defaultDevice],
        selectedDeviceId,
    } = props;

    function App() {
        return (
            <Routes>
                <Route
                    path="/"
                    element={
                        <OwnerDevicesPanel
                            owner={owner}
                            devices={devices}
                            selectedDeviceId={selectedDeviceId}
                            onSelectDevice={onSelectDevice}
                            onAddDevice={onAddDevice}
                        />
                    }
                />
                <Route path={ROUTES.userDevices} element={<div data-testid="owner-workspace" />} />
            </Routes>
        );
    }

    const result = renderWithProviders(<App />, { initialEntries: ['/'] });
    return { ...result, onSelectDevice };
}

const secondOwnerGroup = createMockDeviceOwnerGroup({
    owner: {
        id: 2,
        username: 'other',
        display_name: 'Other User',
        role: UserRole.USER,
        bypass_host_check: false,
        host_groups: [],
        device_count: 1,
        live_address_count: 0,
    },
    devices: [
        {
            id: 2,
            name: 'Other Device',
            state: DeviceState.HEALTHY,
            live_address_count: 0,
            rules: [],
            created_at: '2024-01-01T00:00:00Z',
        },
    ],
});

describe('OwnerDevicesPanel', () => {
    beforeEach(() => {
        // Single group by default — no JUMP section
        server.use(deviceHandlers.list([createMockDeviceOwnerGroup()]));
    });

    // ─── Owner card ──────────────────────────────────────────────────────────────

    describe('owner card', () => {
        it('shows owner display name', () => {
            renderPanel();
            expect(screen.getByText('Test User')).toBeInTheDocument();
        });

        it('shows admin badge for admin role', () => {
            renderPanel({ owner: { ...defaultOwner, role: UserRole.ADMIN } });
            expect(screen.getByText('admin')).toBeInTheDocument();
        });

        it('does not show admin badge for non-admin role', () => {
            renderPanel();
            expect(screen.queryByText('admin')).not.toBeInTheDocument();
        });

        it('shows bypass badge when bypass_host_check is true', () => {
            renderPanel({ owner: { ...defaultOwner, bypass_host_check: true } });
            expect(screen.getByText('All hosts')).toBeInTheDocument();
        });

        it('shows device count', () => {
            renderPanel({ owner: { ...defaultOwner, device_count: 3 } });
            expect(screen.getByText(/3 devices/)).toBeInTheDocument();
        });

        it('appends live count when owner has live addresses', () => {
            renderPanel({ owner: { ...defaultOwner, live_address_count: 2 } });
            expect(screen.getByText(/2 IPs live/)).toBeInTheDocument();
        });
    });

    // ─── Device list ─────────────────────────────────────────────────────────────

    describe('device list', () => {
        it('renders all device names', () => {
            renderPanel({
                devices: [
                    defaultDevice,
                    { ...defaultDevice, id: 2, name: 'Second Device' },
                ],
            });
            expect(screen.getByText('Test Device')).toBeInTheDocument();
            expect(screen.getByText('Second Device')).toBeInTheDocument();
        });

        it('shows "X live" status for live devices', () => {
            renderPanel({
                devices: [{ ...defaultDevice, live_address_count: 3 }],
            });
            expect(screen.getByText(/3 live/)).toBeInTheDocument();
        });

        it('shows "stale" status for stale devices', () => {
            renderPanel({
                devices: [{ ...defaultDevice, state: DeviceState.STALE }],
            });
            expect(screen.getByText(/stale/)).toBeInTheDocument();
        });

        it('shows "never seen" for devices without last_seen_at', () => {
            renderPanel({ devices: [defaultDevice] });
            expect(screen.getByText('never seen')).toBeInTheDocument();
        });

        it('calls onSelectDevice with the device id when clicked', async () => {
            const user = setupUser();
            const onSelectDevice = vi.fn();
            renderPanel({ onSelectDevice });

            await user.click(screen.getByText('Test Device'));

            expect(onSelectDevice).toHaveBeenCalledWith(1);
        });
    });

    // ─── Add device ──────────────────────────────────────────────────────────────

    describe('add device', () => {
        it('calls onAddDevice when "add device" is clicked', async () => {
            const user = setupUser();
            const onAddDevice = vi.fn();
            renderPanel({ onAddDevice });

            await user.click(screen.getByRole('button', { name: /add device/i }));

            expect(onAddDevice).toHaveBeenCalledOnce();
        });
    });

    // ─── Name filter ───────────────────────────────────────────────────────────────

    describe('name filter', () => {
        const manyDevices: DeviceListEntry[] = [
            { ...defaultDevice, id: 1, name: 'Living room TV' },
            { ...defaultDevice, id: 2, name: 'Bob Phone' },
            { ...defaultDevice, id: 3, name: 'Work Laptop' },
            { ...defaultDevice, id: 4, name: 'Spare Phone' },
        ];

        it('hides the filter for short lists', () => {
            renderPanel({ devices: manyDevices.slice(0, 3) });
            expect(screen.queryByPlaceholderText('Filter by name…')).not.toBeInTheDocument();
        });

        it('filters devices by name, case-insensitively', async () => {
            const user = setupUser();
            renderPanel({ devices: manyDevices });

            await user.type(screen.getByPlaceholderText('Filter by name…'), 'phone');

            expect(screen.getByText('Bob Phone')).toBeInTheDocument();
            expect(screen.getByText('Spare Phone')).toBeInTheDocument();
            expect(screen.queryByText('Living room TV')).not.toBeInTheDocument();
            expect(screen.queryByText('Work Laptop')).not.toBeInTheDocument();
        });

        it('shows a no-match message when nothing matches', async () => {
            const user = setupUser();
            renderPanel({ devices: manyDevices });

            await user.type(screen.getByPlaceholderText('Filter by name…'), 'zzz');

            expect(screen.getByText(/No devices match/i)).toBeInTheDocument();
        });

        it('shows a plain empty message when the owner has no devices', () => {
            renderPanel({ devices: [] });

            expect(screen.getByText('No devices yet.')).toBeInTheDocument();
            expect(screen.queryByText(/No devices match/i)).not.toBeInTheDocument();
        });
    });

    // ─── JUMP section ────────────────────────────────────────────────────────────

    describe('jump section', () => {
        it('hides jump section when no other owners exist', async () => {
            renderPanel();

            // Wait for device list data to be fully loaded
            await screen.findByText('Test Device', {}, { timeout: TEST_TIMEOUTS.SHORT });

            expect(screen.queryByText('Jump')).not.toBeInTheDocument();
        });

        it('shows jump section when other owners exist in the list', async () => {
            server.use(deviceHandlers.list([createMockDeviceOwnerGroup(), secondOwnerGroup]));

            renderPanel();

            await waitFor(
                () => expect(screen.getByText('Jump')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('navigates to the selected owner workspace via jump autocomplete', async () => {
            server.use(deviceHandlers.list([createMockDeviceOwnerGroup(), secondOwnerGroup]));

            renderPanel();

            // Wait for the JUMP section to appear (data has loaded)
            await waitFor(
                () => expect(screen.getByPlaceholderText('other owner...')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // fireEvent.change is required: user.type does not trigger Mantine Autocomplete in happy-dom
            fireEvent.change(screen.getByPlaceholderText('other owner...'), { target: { value: 'Other' } });

            const option = await screen.findByRole('option', { name: 'Other User' });
            fireEvent.click(option);

            await waitFor(
                () => expect(screen.getByTestId('owner-workspace')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });
});
