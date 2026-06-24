import React from "react";
import ReactDOM from "react-dom/client";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import App from "./App";
import { getCurrentUserQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { toApiError } from "@/lib/api-client";
import "./lib/api-client/config"; // Initialize API client configuration
dayjs.extend(relativeTime);
import "@mantine/core/styles.css";
import "@mantine/dates/styles.css";
import '@mantine/charts/styles.css';
import "@mantine/notifications/styles.css";
// Unlayered build, imported after @mantine/core/styles.css so the datatable's
// `background: inherit` chain (pinned-column opacity) wins over core Table styles.
// The .layer.css variant must only be used when core styles are layered too.
import "mantine-datatable/styles.css";

const currentUserKey = getCurrentUserQueryKey();

function isCurrentUserQuery(queryKey: unknown): boolean {
  return (
    Array.isArray(queryKey) &&
    JSON.stringify(queryKey) === JSON.stringify(currentUserKey)
  );
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // A 401 means the session is gone; retrying only floods the network and
      // delays the redirect. Fail fast so the auth state flips immediately.
      retry: (failureCount, error) =>
        toApiError(error).status !== 401 && failureCount < 3,
    },
  },
  queryCache: new QueryCache({
    onError: (error, query) => {
      if (toApiError(error).status !== 401) return;
      // The current-user query owns the "logged out" signal: useCurrentUser maps
      // its 401 to null and ProtectedRoute redirects. Any other query hitting 401
      // means the session expired mid-use — flip the cached auth state to null so
      // that same ProtectedRoute redirect fires through the router (no full reload).
      if (isCurrentUserQuery(query.queryKey)) return;
      queryClient.setQueryData(currentUserKey, null);
    },
  }),
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  </React.StrictMode>,
);
