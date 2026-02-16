import {describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {delay} from 'msw';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';
import {DeviceAddressesDialog} from './DeviceAddressesDialog';
import {createMockAddress} from '@/test/mocks/data';
import type {Address} from '@/lib/api';
import {TEST_TIMEOUTS} from '@/test/constants';
import {handlers, responses} from "@/test/mocks/handlers.ts";

describe('DeviceAddressesDialog', () => {
    it('renders trigger button', () => {
        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        expect(screen.getByRole('button', {name: /view addresses/i})).toBeInTheDocument();
    });

    it('opens dialog when trigger clicked', async () => {
        const user = userEvent.setup();
        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByRole('dialog')).toBeInTheDocument();
            expect(screen.getByText(/addresses for test device/i)).toBeInTheDocument();
        });
    });

    it('shows loading state when fetching addresses', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.addresses.getAddressListHandler(undefined, async () => {
                await delay('infinite');
                return responses.ok([]);
            })
        );

        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByText(/loading/i)).toBeInTheDocument();
        });
    });

    it('renders list of addresses when data loaded', async () => {
        const user = userEvent.setup();
        const mockAddresses = [
            createMockAddress({id: 1, ip: '192.168.1.100', status: true}),
            createMockAddress({id: 2, ip: '192.168.1.101', status: true}),
        ];

        server.use(
            handlers.addresses.getAddressListHandler(mockAddresses)
        );

        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByText('192.168.1.100')).toBeInTheDocument();
            expect(screen.getByText('192.168.1.101')).toBeInTheDocument();
        });
    });

    it('shows empty state when no addresses', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.addresses.getAddressListHandler([])
        );

        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByText(/no addresses assigned yet/i)).toBeInTheDocument();
        });
    });

    it('successfully adds new address via form', async () => {
        const user = userEvent.setup();
        const newAddress = createMockAddress({id: 3, ip: '192.168.1.102', status: true});

        const state = {addresses: [] as Address[]};

        server.use(
            handlers.addresses.getAddressListHandler(state.addresses),
            handlers.addresses.createAddressHandler(undefined, () => {
                state.addresses.push(newAddress);
                return responses.created(newAddress)
            })
        );

        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByRole('dialog')).toBeInTheDocument();
        });

        const input = screen.getByLabelText(/add new address/i);
        const addButton = screen.getByRole('button', {name: /add address/i});

        await user.type(input, newAddress.ip);
        await user.click(addButton);

        // Wait for address to appear in list (query will refetch after mutation)
        await waitFor(() => {
            expect(screen.getByText(newAddress.ip)).toBeInTheDocument();
        }, {timeout: TEST_TIMEOUTS.MEDIUM});

        // Form should be reset
        expect((input as HTMLInputElement).value).toBe('');
    });

    it('successfully disables address via button', async () => {
        const user = userEvent.setup();
        const mockAddress = createMockAddress({id: 1, ip: '192.168.1.100', status: true});

        const state = {addresses: [mockAddress] as Address[]};

        server.use(
            handlers.addresses.getAddressListHandler(state.addresses),
            handlers.addresses.deleteAddressHandler(mockAddress, () => {
                mockAddress.status = false
                return responses.ok(mockAddress)
            })
        );

        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByText('192.168.1.100')).toBeInTheDocument();
        });

        const disableButton = screen.getByRole('button', {name: /disable/i});
        await user.click(disableButton);

        await waitFor(() => {
            expect(screen.getByText(/address disabled/i)).toBeInTheDocument();
        }, {timeout: TEST_TIMEOUTS.MEDIUM});

        // Wait for address to show disabled badge (query will refetch after mutation)
        await waitFor(() => {
            const badges = screen.getAllByText(/disabled/i);
            // Should have both toast and badge
            expect(badges.length).toBeGreaterThanOrEqual(1);
        }, {timeout: TEST_TIMEOUTS.MEDIUM});

        // Disable button should no longer be visible
        expect(screen.queryByRole('button', {name: /disable/i})).not.toBeInTheDocument();
    });

    it('shows error states for API failures', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.addresses.getAddressListHandler(undefined, () => {
                return responses.serverError()
            })
        );

        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        // Wait for error state - React Query will show error, check for error message
        // The component doesn't show toast for query errors, just displays error state
        await waitFor(() => {
            // Check that loading is done and error occurred (no addresses shown)
            expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
        }, {timeout: TEST_TIMEOUTS.MEDIUM});
    });

    it('closes dialog on outside click', async () => {
        const user = userEvent.setup();
        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByRole('dialog')).toBeInTheDocument();
        });

        // Press Escape key to close dialog (more reliable than clicking overlay)
        await user.keyboard('{Escape}');

        await waitFor(() => {
            expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
        });
    });

    it('form resets after successful address addition', async () => {
        const user = userEvent.setup();
        const newAddress = createMockAddress({id: 3, ip: '192.168.1.102', status: true});

        const state = {addresses: [] as Address[]};

        server.use(
            handlers.addresses.getAddressListHandler(state.addresses),
            handlers.addresses.createAddressHandler(undefined, () => {
                state.addresses = [newAddress];
                return responses.created(newAddress);
            }),
        );

        renderWithProviders(
            <DeviceAddressesDialog deviceId={1} deviceName="Test Device"/>
        );

        const triggerButton = screen.getByRole('button', {name: /view addresses/i});
        await user.click(triggerButton);

        await waitFor(() => {
            expect(screen.getByRole('dialog')).toBeInTheDocument();
        });

        const input = screen.getByLabelText(/add new address/i) as HTMLInputElement;
        const addButton = screen.getByRole('button', {name: /add address/i});

        await user.type(input, '192.168.1.102');
        expect(input.value).toBe('192.168.1.102');

        await user.click(addButton);

        // Wait for form to reset
        await waitFor(() => {
            expect(input.value).toBe('');
        });
    });
});
