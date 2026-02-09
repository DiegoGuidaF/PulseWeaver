# WallyDex

**WallyDex** is a self-hosted device IP address management service.
Its primary goal is keeping an updated list of your device's IPs in order to only whitelist those on your proxy.

__How it works:__
WallyDex exports your enrolled IPs into a simple .txt whitelist format.
This file can be consumed directly by Caddy (using the import directive) to restrict access to your private services,
ensuring only known devices can connect.

# AI usage
This app has not been vibe coded. I am a software developer with 9+ years of experience.
However AI has been used extensively in some parts of the code to speed development as well as to learn from it. This is
my first time doing a project in Go (my usual stack is Java/Kotlin) as well as doing a React frontend.
AI has been used mostly for writing tests and the frontend.

## Features
- **Device Inventory:** Track hardware with metadata.
- **IP Management:** Assign multiple IPs to devices with IPv4/IPv6 validation.
- **Caddy Integration:** Automatically generates a `whitelist.txt` compatible with Caddy's `import` directive for instant access control.
- **Production Grade:** Built with idiomatic Go (Chi, sqlx, SQLite WAL mode).

## Tech Stack
- **Backend:** Go 1.24+, Chi, SQLite, sqlx
- **Frontend:** React, TypeScript, Vite
