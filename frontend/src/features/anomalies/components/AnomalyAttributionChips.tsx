import { Fragment, type ReactNode } from "react";
import { Anchor, Group, Text } from "@mantine/core";
import { useNavigate } from "react-router-dom";
import { buildRoute } from "@/lib/routes";
import { countryFlagEmoji } from "@/lib/countryFlag";
import type { Anomaly } from "@/lib/api";

interface AnomalyAttributionChipsProps {
    anomaly: Anomaly;
}

/**
 * Entities an anomaly is attributed to, as small inline chips. Device and
 * user link to their detail pages when the entity still exists; a device
 * page is owner-scoped (`/devices/owners/:ownerId`), so a device only links
 * when its owner (`user_id`) is also present. Host and country have no
 * per-entity detail route in this app, so they render as plain text.
 */
export function AnomalyAttributionChips({ anomaly }: AnomalyAttributionChipsProps) {
    const navigate = useNavigate();
    const parts: ReactNode[] = [];

    if (anomaly.device_name) {
        const href =
            anomaly.user_id != null
                ? `${buildRoute.userDevices(anomaly.user_id)}${
                      anomaly.device_id != null ? `?device=${anomaly.device_id}` : ""
                  }`
                : undefined;
        parts.push(
            href ? (
                <Anchor
                    key="device"
                    component="button"
                    type="button"
                    size="xs"
                    onClick={(e) => {
                        e.stopPropagation();
                        navigate(href);
                    }}
                >
                    {anomaly.device_name}
                </Anchor>
            ) : (
                <Text key="device" size="xs" c="dimmed">
                    {anomaly.device_name}
                </Text>
            ),
        );
    }

    if (anomaly.user_name) {
        parts.push(
            anomaly.user_id != null ? (
                <Anchor
                    key="user"
                    component="button"
                    type="button"
                    size="xs"
                    onClick={(e) => {
                        e.stopPropagation();
                        navigate(buildRoute.accessUserDetail(anomaly.user_id as number));
                    }}
                >
                    {anomaly.user_name}
                </Anchor>
            ) : (
                <Text key="user" size="xs" c="dimmed">
                    {anomaly.user_name}
                </Text>
            ),
        );
    }

    if (anomaly.target_host) {
        parts.push(
            <Text key="host" size="xs" c="dimmed" ff="monospace">
                {anomaly.target_host}
            </Text>,
        );
    }

    if (anomaly.country_code) {
        const flag = countryFlagEmoji(anomaly.country_code);
        parts.push(
            <Text key="country" size="xs" c="dimmed">
                {flag && `${flag} `}
                {anomaly.country_code}
            </Text>,
        );
    }

    if (parts.length === 0) return null;

    return (
        <Group gap={6} wrap="wrap">
            {parts.map((part, i) => (
                <Fragment key={i}>
                    {i > 0 && (
                        <Text size="xs" c="dimmed">
                            ·
                        </Text>
                    )}
                    {part}
                </Fragment>
            ))}
        </Group>
    );
}
