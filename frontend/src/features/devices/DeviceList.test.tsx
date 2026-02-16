import {describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import {delay} from 'msw';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';
import {DeviceList} from './DeviceList';
import {createMockDevice} from '@/test/mocks/data';
import {handlers, responses} from "@/test/mocks/handlers.ts";

describe('DeviceList', () => {

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
