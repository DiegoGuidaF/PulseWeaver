import {describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {delay} from 'msw';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';
import {DeviceList} from './DeviceList';
import {createMockDevice} from '@/test/mocks/data';
import {handlers, responses} from "@/test/mocks/handlers.ts";
import {TEST_TIMEOUTS} from '@/test/constants';

describe('DeviceList', () => {

    it('renders manage link for each device that navigates to detail page', async () => {
        const mockDevices = [
            createMockDevice({id: 1, name: 'Device One'}),
        ];
        server.use(handlers.devices.getDeviceListHandler(mockDevices));

        renderWithProviders(<DeviceList/>);

        await waitFor(() => {
            expect(screen.getByText('Device One')).toBeInTheDocument();
        });

        const manageLink = screen.getByRole('link', {name: /manage/i});
        expect(manageLink).toHaveAttribute('href', '/devices/1');
    });

    it('shows delete button for each device', async () => {
        const mockDevices = [
            createMockDevice({id: 1, name: 'Device One'}),
        ];
        server.use(handlers.devices.getDeviceListHandler(mockDevices));

        renderWithProviders(<DeviceList/>);

        await waitFor(() => {
            expect(screen.getByText('Device One')).toBeInTheDocument();
        });
        expect(
            screen.getByRole('button', {name: /delete device device one/i})
        ).toBeInTheDocument();
    });

    it('opens confirm dialog when delete is clicked', async () => {
        const user = userEvent.setup();
        const mockDevices = [
            createMockDevice({id: 1, name: 'To Delete'}),
        ];
        server.use(handlers.devices.getDeviceListHandler(mockDevices));

        renderWithProviders(<DeviceList/>);

        await waitFor(() => {
            expect(screen.getByText('To Delete')).toBeInTheDocument();
        });

        await user.click(
            screen.getByRole('button', {name: /delete device to delete/i})
        );

        expect(screen.getByRole('dialog', {name: /delete device/i})).toBeInTheDocument();
        expect(
            screen.getByText(/delete device "to delete"\?/i)
        ).toBeInTheDocument();
        expect(
            screen.getByText(/hidden from the list and cannot receive addresses/i)
        ).toBeInTheDocument();
    });

    it('calls delete and removes device from list on confirm', async () => {
        const user = userEvent.setup();
        let listCallCount = 0;
        server.use(
            handlers.devices.getDeviceListHandler(undefined, async () => {
                listCallCount++;
                if (listCallCount === 1) {
                    return responses.ok([
                        createMockDevice({id: 1, name: 'To Delete'}),
                    ]);
                }
                return responses.ok([]);
            }),
            handlers.devices.deleteDeviceHandler
        );

        renderWithProviders(<DeviceList/>);

        await waitFor(() => {
            expect(screen.getByText('To Delete')).toBeInTheDocument();
        });

        await user.click(
            screen.getByRole('button', {name: /delete device to delete/i})
        );
        await user.click(
            screen.getByRole('button', {name: /^delete$/i})
        );

        await waitFor(
            () => {
                expect(screen.getByText(/device deleted/i)).toBeInTheDocument();
            },
            {timeout: TEST_TIMEOUTS.MEDIUM}
        );

        await waitFor(
            () => {
                expect(screen.getByText('No devices found.')).toBeInTheDocument();
            },
            {timeout: TEST_TIMEOUTS.MEDIUM}
        );
    });

    it('shows loading skeleton initially', async () => {
        server.use(
            handlers.devices.getDeviceListHandler(undefined, async () => {
                await delay('infinite');
                return responses.ok([]);
            })
        );

        renderWithProviders(<DeviceList/>);

        // Verify title is shown
        expect(screen.getByText('Devices')).toBeInTheDocument();

        // Verify skeleton elements are present
        const skeletons = screen.getAllByRole('generic');
        expect(skeletons.length).toBeGreaterThan(0);
    });

    it('renders devices after successful fetch', async () => {
        const mockDevices = [
            createMockDevice({id: 1, name: 'Test Device 1'}),
            createMockDevice({id: 2, name: 'Test Device 2'}),
        ];

        server.use(
            handlers.devices.getDeviceListHandler(mockDevices)
        );

        renderWithProviders(<DeviceList/>);

        // Wait for devices to appear
        await waitFor(() => {
            expect(screen.getByText('Test Device 1')).toBeInTheDocument();
        });

        expect(screen.getByText('Test Device 2')).toBeInTheDocument();
        expect(screen.getByText('1')).toBeInTheDocument(); // Device ID
        expect(screen.getByText('2')).toBeInTheDocument(); // Device ID
    });

    it('shows empty state when no devices are found', async () => {
        server.use(
            handlers.devices.getDeviceListHandler([])
        );

        renderWithProviders(<DeviceList/>);

        await waitFor(() => {
            expect(screen.getByText('No devices found.')).toBeInTheDocument();
        });

        expect(
            screen.getByText('Add a device above to get started.')
        ).toBeInTheDocument();
    });

    it('shows error message on API error', async () => {
        server.use(
            handlers.devices.getDeviceListHandler(undefined, async () => {
                return responses.serverError()
            })
        );

        renderWithProviders(<DeviceList/>);

        await waitFor(() => {
            expect(screen.getByText(/Error:/i)).toBeInTheDocument();
        });
    });
});
