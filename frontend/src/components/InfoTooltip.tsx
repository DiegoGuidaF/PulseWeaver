import { Tooltip, ThemeIcon, type FloatingPosition } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";

interface InfoTooltipProps {
    /** Explanation shown on hover/focus. Keep it to a sentence or two. */
    label: string;
    /** Accessible name for the icon trigger; defaults to a generic phrasing. */
    "aria-label"?: string;
    position?: FloatingPosition;
}

/** Small "?" affordance that explains what a metric means and, where useful, what a healthy value looks like. */
export function InfoTooltip({ label, position = "top", "aria-label": ariaLabel = "More information" }: InfoTooltipProps) {
    return (
        <Tooltip label={label} multiline w={260} withArrow position={position}>
            <ThemeIcon variant="transparent" color="gray" size="sm" aria-label={ariaLabel}>
                <IconInfoCircle size={16} stroke={1.5} />
            </ThemeIcon>
        </Tooltip>
    );
}
