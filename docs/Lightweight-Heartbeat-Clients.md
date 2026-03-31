# Lightweight Heartbeat Clients

If you don't need the full [PulseWeaver Heartbeat Client](https://github.com/DiegoGuidaF/pulseweaver-heartbeat-client) app, a simple `curl` command scheduled by your OS is all it takes. Zero dependencies beyond `curl`.

## Linux — systemd timer

A systemd **timer + oneshot service** is the idiomatic way to schedule recurring tasks on Linux.

### 1. Create the service

`/etc/systemd/system/pulseweaver-heartbeat.service`

```ini
[Unit]
Description=PulseWeaver heartbeat ping
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStart=curl -sf -X POST -H "X-API-Key: wdk_YOUR_KEY_HERE" https://pw.example.com/api/v1/heartbeat
```

### 2. Create the timer

`/etc/systemd/system/pulseweaver-heartbeat.timer`

```ini
[Unit]
Description=Send PulseWeaver heartbeat every 5 minutes

[Timer]
OnBootSec=30s
OnUnitActiveSec=5min
AccuracySec=30s

[Install]
WantedBy=timers.target
```

### 3. Enable and start

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now pulseweaver-heartbeat.timer

# Verify it's scheduled
systemctl list-timers pulseweaver-heartbeat.timer

# Test a manual run
sudo systemctl start pulseweaver-heartbeat.service
journalctl -u pulseweaver-heartbeat.service -n 5
```

> **Security note:** The example above inlines the API key in the unit file for simplicity. For production use, consider systemd's built-in credential management (`LoadCredential=` / `LoadCredentialEncrypted=`, available since systemd 247) which injects secrets at runtime via `$CREDENTIALS_DIRECTORY` — keeping them out of unit files, environment variables, and process listings. See the [systemd credentials docs](https://systemd.io/CREDENTIALS/) for details.

---

## macOS — launchd agent

On a Mac mini server or any headless macOS machine, a **launchd agent** with `curl` replaces the full app.

### 1. Create the plist

`~/Library/LaunchAgents/com.pulseweaver.heartbeat.plist`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.pulseweaver.heartbeat</string>

    <key>ProgramArguments</key>
    <array>
        <string>curl</string>
        <string>-sf</string>
        <string>-X</string>
        <string>POST</string>
        <string>-H</string>
        <string>X-API-Key: wdk_YOUR_KEY_HERE</string>
        <string>https://pw.example.com/api/v1/heartbeat</string>
    </array>

    <key>StartInterval</key>
    <integer>300</integer>

    <key>RunAtLoad</key>
    <true/>

    <key>StandardErrorPath</key>
    <string>/tmp/pulseweaver-heartbeat.err</string>
</dict>
</plist>
```

### 2. Load and start

```bash
launchctl load ~/Library/LaunchAgents/com.pulseweaver.heartbeat.plist

# Verify it's loaded
launchctl list | grep pulseweaver

# Test a manual run
launchctl start com.pulseweaver.heartbeat
cat /tmp/pulseweaver-heartbeat.err
```

To stop and unload: `launchctl unload ~/Library/LaunchAgents/com.pulseweaver.heartbeat.plist`

> **Security note:** The API key is inlined in the plist for simplicity. For better secret handling, store the key in the macOS Keychain and retrieve it with `security find-generic-password` in a wrapper script, or use an environment variable sourced from a file with restricted permissions (`chmod 600`).
