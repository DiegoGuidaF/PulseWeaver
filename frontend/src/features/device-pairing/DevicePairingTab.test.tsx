import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen } from '@testing-library/react';
import { DevicePairingTab } from '@/features/device-pairing/DevicePairingTab';
import type { DevicePairing } from '@/lib/api';
import { DeviceState } from '@/lib/api';
import type { DeviceState as DeviceStateType } from '@/lib/api';
import { endpoints } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { createMockDevicePairing } from '@/test/mocks/data';

// The tab issues two list calls — ?status=pending and ?status=all. Branch the
// mock on the query param so we can stage each link state independently.
function mockPairings({ pending, all }: { pending: DevicePairing[]; all: DevicePairing[] }) {
    server.use(
        http.get(endpoints.devicePairings, ({ request }) => {
            const status = new URL(request.url).searchParams.get('status');
            return HttpResponse.json(status === 'pending' ? pending : all);
        }),
    );
}

function renderTab(state: DeviceStateType = DeviceState.HEALTHY) {
    return renderWithProviders(<DevicePairingTab deviceId={1} deviceState={state} />);
}

describe('DevicePairingTab link states', () => {
    it('shows the create form when the device was never paired', async () => {
        mockPairings({ pending: [], all: [] });
        renderTab();

        expect(await screen.findByText('Generate a pairing code')).toBeInTheDocument();
        expect(screen.queryByText('Linked to companion app')).not.toBeInTheDocument();
    });

    it('shows the active code when one is outstanding and never claimed', async () => {
        mockPairings({
            pending: [createMockDevicePairing({ status: 'pending' })],
            all: [createMockDevicePairing({ status: 'pending' })],
        });
        renderTab();

        expect(await screen.findByText('Active pairing code')).toBeInTheDocument();
        expect(screen.queryByText(/current link stays active/i)).not.toBeInTheDocument();
    });

    it('leads with link status when the device is claimed', async () => {
        mockPairings({
            pending: [],
            all: [createMockDevicePairing({ id: 9, status: 'used' })],
        });
        renderTab();

        expect(await screen.findByText('Linked to companion app')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /generate another code/i })).toBeInTheDocument();
        // The bare create form must not be the hero for an already-paired device.
        expect(screen.queryByText('Generate a pairing code')).not.toBeInTheDocument();
    });

    it('flags a replacement code as non-destructive when re-pairing a claimed device', async () => {
        mockPairings({
            pending: [createMockDevicePairing({ id: 2, status: 'pending' })],
            all: [
                createMockDevicePairing({ id: 2, status: 'pending' }),
                createMockDevicePairing({ id: 9, status: 'used' }),
            ],
        });
        renderTab();

        expect(await screen.findByText('New pairing code')).toBeInTheDocument();
        expect(screen.getByText(/current link stays active/i)).toBeInTheDocument();
    });

    it('shows a focused expired card when an unclaimed code expired', async () => {
        mockPairings({ pending: [], all: [] });
        renderTab(DeviceState.EXPIRED_CLAIM);

        expect(await screen.findByText('Pairing code expired')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /generate new code/i })).toBeInTheDocument();
    });
});
