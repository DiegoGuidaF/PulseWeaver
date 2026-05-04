import { useQuery } from "@tanstack/react-query";
import { simulatePolicyAccessOptions } from "@/lib/api/@tanstack/react-query.gen";

export function usePolicySimulate(ip: string, host: string) {
  const query = useQuery({
    ...simulatePolicyAccessOptions({ query: { ip, host } }),
    enabled: false,
  });

  return {
    result: query.data,
    isFetching: query.isFetching,
    isError: query.isError,
    refetch: query.refetch,
  };
}
