import { useCallback } from "react";
import { useSearchParams } from "react-router";

const PARAM = "request";

/**
 * Which access-log entry the detail drawer is showing, held in the URL
 * (`?request=<id>`) so the browser Back button closes the drawer — the expected
 * gesture, especially on mobile.
 *
 * Anything that is not a positive integer reads as no selection. The id feeds a
 * render-phase comparison in the drawer, where a `NaN` from a hand-edited param
 * never equals itself and would re-render until React gives up.
 */
export function useRequestParam() {
    const [searchParams, setSearchParams] = useSearchParams();

    const parsed = Number(searchParams.get(PARAM));
    const requestId = Number.isInteger(parsed) && parsed > 0 ? parsed : null;

    const openRequest = useCallback(
        (id: number) => {
            setSearchParams((prev) => {
                const next = new URLSearchParams(prev);
                next.set(PARAM, String(id));
                return next;
            });
        },
        [setSearchParams],
    );

    const closeRequest = useCallback(() => {
        // Replace rather than push so the cleared state doesn't add a history
        // entry that Back would step into and re-open the drawer.
        setSearchParams(
            (prev) => {
                const next = new URLSearchParams(prev);
                next.delete(PARAM);
                return next;
            },
            { replace: true },
        );
    }, [setSearchParams]);

    return { requestId, openRequest, closeRequest };
}
