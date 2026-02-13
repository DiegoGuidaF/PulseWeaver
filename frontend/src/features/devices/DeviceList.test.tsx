import {describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import {delay, http, HttpResponse} from 'msw';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';
import {DeviceList} from './DeviceList';
import {createMockDevice} from '@/test/mocks/data';

describe('DeviceList', () => {

    it('shows loading skeleton initially', async () => {
        // Override handler to delay response indefinitely
        server.use(
            http.get('/api/v1/devices', async () => {
                await delay('infinite');
                return HttpResponse.json([])
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
            http.get('/api/v1/devices', () => {
                return HttpResponse.json(mockDevices);
            })
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
            http.get('/api/v1/devices', () => {
                return HttpResponse.json([]);
            })
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
            http.get('/api/v1/devices', () => {
                return HttpResponse.json(
                    {error: 'Failed to fetch devices'},
                    {status: 500}
                );
            })
        );

        renderWithProviders(<DeviceList/>);

        await waitFor(() => {
            expect(screen.getByText(/Error:/i)).toBeInTheDocument();
        });
    });
});
