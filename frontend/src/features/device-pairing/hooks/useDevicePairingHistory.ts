import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { PairingListFilter } from "@/lib/api";
import { patchDevicePairingSummary } from "@/features/devices/fleetCache";
import { useListDevicePairings } from "./useListDevicePairings";

/**
 * A device's full pairing history, newest first — the same row the fleet's
 * pairing summary is derived from, with the same server-evaluated status.
 *
 * Reading it therefore settles whether the fleet copy behind the tab's dot is
 * still current, so the result is written back there on every change.
 */
export function useDevicePairingHistory(deviceId: number, ownerId: number) {
  const queryClient = useQueryClient();
  const query = useListDevicePairings(deviceId, PairingListFilter.ALL);
  const pairings = query.data;

  useEffect(() => {
    if (!pairings) return;
    const latest = pairings[0];
    patchDevicePairingSummary(
      queryClient,
      ownerId,
      deviceId,
      latest ? { status: latest.status, expires_at: latest.expires_at } : null,
    );
  }, [pairings, queryClient, ownerId, deviceId]);

  return query;
}
