import { useMemo } from "react";
import { useDeviceList } from "./useDeviceList";

export function useOwnerGroup(ownerId: number) {
  const { data, isLoading, error } = useDeviceList();

  const group = useMemo(
    () => data?.find((g) => g.owner.id === ownerId),
    [data, ownerId],
  );

  return { data: group, isLoading, error };
}
