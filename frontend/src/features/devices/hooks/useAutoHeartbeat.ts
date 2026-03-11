import { useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { deviceHeartbeatMutation } from '@/lib/api/@tanstack/react-query.gen';
import {
  getAutoHeartbeatSettings,
  storeClientIp,
  type AutoHeartbeatSettings,
} from '@/lib/autoHeartbeat';

export function useAutoHeartbeat() {
  const [settings, setSettings] = useState<AutoHeartbeatSettings | null>(
    getAutoHeartbeatSettings,
  );
  const [clientIp, setClientIp] = useState<string | null>(null);

  // Sync settings from localStorage (reacts to same-tab dispatches too)
  useEffect(() => {
    const handler = () => setSettings(getAutoHeartbeatSettings());
    window.addEventListener('storage', handler);
    return () => window.removeEventListener('storage', handler);
  }, []);

  const mutation = useMutation({
    ...deviceHeartbeatMutation(),
    onSuccess: (address) => {
      setClientIp(address.ip);
      storeClientIp(address.ip);
    },
    // silent — no toast
  });

  // Main interval
  useEffect(() => {
    if (!settings) {
      setClientIp(null);
      return;
    }
    const fire = () =>
      mutation.mutate({ path: { device_id: settings.deviceId } });
    fire();
    const id = setInterval(fire, settings.intervalSeconds * 1_000);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings?.deviceId, settings?.intervalSeconds]);

  // Re-fire on tab focus
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible' && settings)
        mutation.mutate({ path: { device_id: settings.deviceId } });
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings?.deviceId]);

  return { clientIp, activeDeviceId: settings?.deviceId ?? null };
}
