# Changelog

All notable changes to this project will be documented in this file.

## [0.5.0] - 2026-06-26

### Bug Fixes

- *(ui)* keep dashboard entity tables within their cards ([`5dd3a6e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5dd3a6ee78a667708f3cb4b73be384f72df06b46))
- *(ui)* Show plain empty message when owner has no devices ([`0718d30`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0718d305859365508271b7ff6030690d303b8e11))
- *(ui)* Confine pairing toggle click area to its content ([`91c89c6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/91c89c66615919b606ec18bfb07ad93f95ca0abe))
- *(ui)* Redirect to login on logout and session-expiry 401 ([`0805815`](https://github.com/DiegoGuidaF/PulseWeaver/commit/080581576e0597a64ef9def01fe83a8f64038bfe))
- Make docker data volume startup reliable ([`0b418ea`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0b418ea59c62345ffd023f09094e5c9228cf7ed7))

### Documentation

- Document commit convention in CLAUDE.md ([`85038ea`](https://github.com/DiegoGuidaF/PulseWeaver/commit/85038ea127b49e82e54e5e6a85de9649f93f49f2))
- Improve documentation for AI agents and properly refresh/replace stale docs ([`4ae09f8`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4ae09f865593280f4fcf658eebd83e853e4c4698))
- Improve readme, update screenshots and improve readability by properly organizing it ([`a861e3a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a861e3a175d50c35984733d6eb70d01fafbe60a9)) ([`d84536a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d84536acb5497afc2964d4874cc2ebe98f4e97ef))

### Features

- *(ui)* lead device pairing tab with link status ([`4d240fb`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4d240fb56b2e1027ff0ca0fcd8f5812fb8c6bb3a))
- *(docker)* Publish multi-arch images (aarch was missing) ([`1b1fbce`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1b1fbce03c1e98d2fcaee2865dbbcb55a5dc1601))

### Under the Hood

- DenyReason list to openapi as single source of truth and source it from both backend and frontend ([`e41d513`](https://github.com/DiegoGuidaF/PulseWeaver/commit/e41d513865dce9afb07807aaeba2ef2701ed6433))
- Improve dashboard security posture query performance by sourcing from db instead of cache ([`7571b26`](https://github.com/DiegoGuidaF/PulseWeaver/commit/7571b269ddbbb25e34b52f7a2b39118ada889ced))
- Restructure Makefile with back-/front- command prefixes to make it easier to use ([`48929e5`](https://github.com/DiegoGuidaF/PulseWeaver/commit/48929e5dc307d3aa4ca21ba717355750b8306b62))
- *(ui)* Improve frontend main flows test coverage ([`92bb838`](https://github.com/DiegoGuidaF/PulseWeaver/commit/92bb8389680a5274501423496b39f3a3244f842d))
- *(ci)* Add Go benchmark flow to test how this could look as part of a github runner action ([`bf46821`](https://github.com/DiegoGuidaF/PulseWeaver/commit/bf46821895a23010050744ed6257f7beb54d4ef3))

## [0.4.0] - 2026-06-23

### Documentation

- Add per-feature documentation and restructure README with latest changes
- Improve setup documentation and fix Caddy devices endpoint missing X-Real-IP header ([`93c32ef`](https://github.com/DiegoGuidaF/PulseWeaver/commit/93c32effae92e0604e218305607418e297054a41))

### Features

- Improve user flows in all pages based on use-case based analysis
- Implement network policies management for CIDR addresses
- Add more information to the dashboard and improve performance of queries.
- Dashboard info tooltips, clickable attribution rows, reorder ([`22cc7c4`](https://github.com/DiegoGuidaF/PulseWeaver/commit/22cc7c4ce74e56640e474fd22da484b51d4623bf))
- Make device list user centered and improve device creation flow
- Add cleanup job for access log and address event log data. Defaults to 1month configurable via ENV_VARIABLE ([`d267edf`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d267edf4e4ac1c2ec64891bc2874473e53f2aa7e))
- Improve device recommendations by warning the user when a device with an API_KEY present has no rules
- IPv6-native engine on canonical netip.Addr; fix 4-in-6 asymmetry ([`adb6dc3`](https://github.com/DiegoGuidaF/PulseWeaver/commit/adb6dc31be35f08fd005f2c7bddb80009d3fcb8a))
- Improve access verification page (previously named policy cache audit)
- Allow disabling a device so that no addresses can be added and current ones are disabled. Meant as a temporary measure
- Change provisioning to a device pairing concept. Allowing easy pairing and re-pairing of existing devices.
- Device pairing - Show QR code to easily copy the pairing code ([`236ed50`](https://github.com/DiegoGuidaF/PulseWeaver/commit/236ed5045c0067f61e007199216691bb1dea5286))
- Add more filtering capabilities to existing tables as well as pages listing devices, addresses.. and such
- Improve onboarding. On missing entities (missing hosts, groups, devices...) guide user on how to create them. ([`13ea68e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/13ea68e68001552dc3596f55d05f1e5196249fa9))
- Add geoip enrichment on all IP fields shown in the frontend
- Host assignation can only happen via groups to simplify the model and avoid confusion. Individual host assigment
  together with group based can easily result in complex UI and difficulty knowing which hosts are actually allowed
  on each user if user list grows
- Improve group badges color and contrast to facilitate visual discrimination and accessibility.
- Add new branding ([`af5baed`](https://github.com/DiegoGuidaF/PulseWeaver/commit/af5baed9a17b8390ff09ca2559d13894454b032e))
- Improve feedback when trying to add trusted_proxy as a device address ([`051ab8b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/051ab8b230cb3166583c76f25b9bd08c54ea522b))
- Do not allow sending IP on heartbeat endpoints. There's no current valid usecase and it can be a security issue if misused ([`05f52af`](https://github.com/DiegoGuidaF/PulseWeaver/commit/05f52afd82098f9c427086d1e4a66c9791537595))
- Remove device type concept from devices, if needed will be added later with more context and how that should look like
- Address log - Add information regarding time gap and ttl to each row in order to facilitate spotting users with short TTL or device misconfiguration ([`dee6b76`](https://github.com/DiegoGuidaF/PulseWeaver/commit/dee6b766b8a69f192b4e9593e10de09ada2b1686))
- Allow users with null email. Email is just metadata for now ([`7419768`](https://github.com/DiegoGuidaF/PulseWeaver/commit/7419768965eff69937e7a3fa97ff52221bb27e95))

### Under the Hood

- Add sample database seeding to facilitate local development, testing and showcasing (such as taking screenshots for the README)
- Allow compiling a pprof exposing binary under a build tag on a loopback listener to facilitate analyzing CPU and HEAP usage on critical flows ([`aa925fa`](https://github.com/DiegoGuidaF/PulseWeaver/commit/aa925fa7d52d4232ac65e37bc67829243ab8f66a))
- Improve performance of access log query ([`4c0bb2c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4c0bb2cbc624ac0e9fdac8472086f5d5070b725c))
- Reorganize the frontend pages into folders matching routing to facilitate navigating them ([`29997f3`](https://github.com/DiegoGuidaF/PulseWeaver/commit/29997f3935f97980c605ab4938ae07813ca6dbbc))
- *(ci)* Run backend and frontend test concurrently to speedup ci/cd ([`c7c220d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c7c220d14bbdfe0a6f97c6af7f55389a61a3e0a2))
- Add cross-domain integration tests to fully validate critical flows
- Add GO benchmarks to critical flows and improve them based on that (access log query, policy evaluation...)
- *(backend)* Generate openapi test client to help with handler tests by using it instead of implementing the http client code ([`1d3eddf`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1d3eddfe962a6a7e6d13ae3c82cc2a66a524f73b))
- *(backend)* Do not do time.sleep for cross-domain integration testing when waiting for policy refresh. Do a poll-based approach ([`99496cc`](https://github.com/DiegoGuidaF/PulseWeaver/commit/99496ccd772fb5e5f41d04ebcff5e106a3b37d95))
- *(backend)* Delete devices and addresses when deleting a user ([`c1f91a5`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c1f91a50cf78847adbcf19d07ea0c9fb951b84d7))
- *(backend)* Add Squirrel Go library to facilitate building dynamic queries ([`b4a5a36`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b4a5a3646f6f13d9943b9458f59ba2da6744dfd6)) ([`a7c6290`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a7c6290e640b093f8e9dbe10bb227c41180a684d)) ([`bb79213`](https://github.com/DiegoGuidaF/PulseWeaver/commit/bb79213aaf2727ca827075ed1839aabcd42e44fb))
- *(backend)* Improve logging output by reducing noise (such as moving some from error to debug where applicable) and ensuring UI actions resulting in changes have proper logging
- *(backend)* Remove device_type from database since it no longer makes sense as a stored type (it will be calculated depending on conditions such as rules) ([`b55eb6d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b55eb6d441cdbcebcfbf59db5f131490aa2e447b))
- Greatly improve frontend test speed by removing CSS imports from Mantine and improving general setup ([`68b685d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/68b685dabae1d24c80c11182e6bf46773b2e66f3)) ([`332133e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/332133e105e6618ee4524afd91742ff7bfe50fed))
- *(backend)* Bound host suggestions scan to last 7 days to reduce old noise and improve performance ([`27513c7`](https://github.com/DiegoGuidaF/PulseWeaver/commit/27513c7ddd1269d8707205073ad09a4e02c0f3b2))

## [0.3.0] - 2026-05-06

### Bug Fixes

- Properly set default address updated_at value for devices (was 1970) ([`6b60ee4`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6b60ee4c70b7978394e296fbd00e30bb77027cfa))
- Show owner name not id on device provisioning invite creation ([`6bc3236`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6bc3236bee01ec4489e6696f2a332636ee16889e))
- On device delete ensure addresses are disabled and API keys removed ([`e7614c7`](https://github.com/DiegoGuidaF/PulseWeaver/commit/e7614c754c949c4881702d034c9050dca82f247c))

### Features

- Add device ownership with per-user scoping
- Add Policy Cache Auditing: Allows reviewing active IPs by User. Source of truth is same one as the authorization endpoint used by the proxy
- Allow simulating a request to check which response the proxy would see. Added to Policy Cache page (the one added by the feature above)
- Add device provisioning page: Allows creating a device registration code for use by the heartbeat-client application.
- Remove non-admin user login. Allow only admin user login. Simplify flows accordingly and ask for password on user -> admin promotion ([`5d7d577`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5d7d5775d4ff904878c3b0bfaa8c41c5c6cd203c))
    - Only bootstrapped admin (the admin account created on first initialization with superadmin role) can create/promote/demote/delete users ([`bd78411`](https://github.com/DiegoGuidaF/PulseWeaver/commit/bd7841140241d95a16aaa8b39b1f150bd1c08921))
- Add per user host based authorization: Allows restricting users to only specific hosts either individually and/or via host groups.

### Miscellaneous

- Devices List - Visual improvements
- AccessLog - The user can now copy the headers shown on the item details side panel
- Dashboard - Hide country map section when no geo data is available ([`12182b9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/12182b9e0db417f54b68cb7a4207d0e513f1f7a8))
- Navbar - Add GitHub docs and feedback links ([`53230db`](https://github.com/DiegoGuidaF/PulseWeaver/commit/53230dbb1a1d1c55a3ef3d8471cff6604c56a46a))
- Requests Traffic charts - Add toggle buttons for Allowed/Denied lines on traffic chart ([`1557e41`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1557e41aec4766ce8406366323bc5480021bb935))
- AccessLog | AddressLog - Make device name clickable ([`67802a6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/67802a63a25b91ce54f5a98f149e11bb976cf614))
- Devices - Make device api-key optional. Do not create it by default. ([`70dbd63`](https://github.com/DiegoGuidaF/PulseWeaver/commit/70dbd63d863340d139c4771ab71dddecbf31230c))
- Device Settings Tab - Split the rules into a separate Rules tab ([`bce29cb`](https://github.com/DiegoGuidaF/PulseWeaver/commit/bce29cb172c65514644ded4ac6c6bdc8ae66787d))

### Under the Hood

- Split openapi.yaml into smaller files to reduce size and context needed for changes. Bundle it into a single file using redocly before sending it to backend or frontend ([`708d78a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/708d78a52c9f93bdb4a0e985ba4c3cc84bc3c47d))
- Auto-merge dependabot. Pending enable automerge at github repo level, needs public repo. ([`8bf0a31`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8bf0a3106ab0474842ac8263c5b67098b5e13260))
- DB - Remove unneeded created_at and id columns for access_log_contributors to reduce size since a lot of records will be there usually ([`88bd113`](https://github.com/DiegoGuidaF/PulseWeaver/commit/88bd113d3dda92ee67553e6f9cc3fc6c9c641453))
- DB - Allow multiple open connections on SQLite DB so that slow queries don't block quick ones. WAL mode already enabled. ([`8ed847e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8ed847ecb3132ccee4b789e6b145d373268d10b5))

## [0.2.1] - 2026-04-02

### Bug Fixes

- *(ci)* Fix release flow. Remove PR enrichment from cliff since it wasn't needed ([`dc143d0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/dc143d0418ec0822f1628c141c7c4c3df85eeac4))
- *(lint)* Fix linting ([`6f525db`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6f525dbda9ac97987c22be82056e0c92019504a1))
- *(backend)* Fix failing test ([`ca7db6c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/ca7db6ce1ddc43a1a78f8ffd584a1a41a0a4bb1a))
- *(ui)* Fix failing test due to API generated code changes. Make tests less flaky by limiting vitest workers to 50% ([`182cb15`](https://github.com/DiegoGuidaF/PulseWeaver/commit/182cb15bd54851dd61b18f48c0d6053d912b6337))
- *(backend)* Make sure "generic" devicetype is replaced everywhere with "static" ([`154073a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/154073ac5c02d257199a199ee87107b3ef28e7a2))

### Features

- Track avg response time in hourly aggregates and dashboard ([`1b4727b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1b4727b22865528fc13f5fd08ad2c825eb7baecc))
- *(backend)* Reject UDP Early-data=1 traffic properly with StatusTooEarly (425). This removes noise from traffic log (it was being seen as if coming from the trusted proxy) and provides a better response than a 403. ([`54b1c90`](https://github.com/DiegoGuidaF/PulseWeaver/commit/54b1c90d78083a8e6b8bdcff529d1e558ed7ae15))
- *(backend)* Add max number of active IPs rule per device ([`be0aa2a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/be0aa2a744dba4ec7259e71587e3ee82e2147690))
- *(ui)* Add rule config for max number of active addresses per device ([`b74185a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b74185ae5d89cf4c5291c7fac2108e448b9d5912))
- *(backend)* Enforce max active address on rule change ([`0a85186`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0a85186fae963edbc83f6b86741e5da6fd880919))
- *(backend)* Add more properties to device and allow changing the name ([`fe2da19`](https://github.com/DiegoGuidaF/PulseWeaver/commit/fe2da19a170a5df768e90aeead20830a15425d65))

### Miscellaneous

- *(ui)* Improve traffic map UI ([`70801b9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/70801b905f951113e6a113013c4f367f270c858b))
- *(backend)* Add verify request duration in microseconds to have metrics on performance ([`d017fd7`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d017fd7e5bdb13dd037223df27c52f3a570d1af7))
- *(ci)* Put tests on separate workflow so it can be sourced by ci and release. General improvements to it ([`0fa8f08`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0fa8f089a3ace5e15a16a57df9049b601943224a))
- *(backend)* Improve migration testing by testing latest migration with fake seed data ([`af05cee`](https://github.com/DiegoGuidaF/PulseWeaver/commit/af05ceeefd3f51af173b02d1a85b4db16c2ff972))
- *(ai)* Ensure test migration seed file is kept up to sync whenever a new migration is added ([`b6d25c5`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b6d25c507862e42a20837e1124a6a0ad3337f645))

## [0.1.0] - 2026-04-01

### Backend

- Fully implement openapi server code ([`8e22068`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8e22068cae73d63b9df54f28de85abcadd6dc0e2))
- Remove no longer needed code ([`ef4ef73`](https://github.com/DiegoGuidaF/PulseWeaver/commit/ef4ef73a223857a036dedf1a7cfe8144f581cb31))
- Add RequestId along with basic middleware settings ([`c08cb45`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c08cb45a7ccd803bfb5a70b4dca58b1e2f78edd0))
- Simplify disableIp logic since there is no need for so many status codes ([`0a1e78c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0a1e78cdf5e0209f593ca81bb828b9b48cacfb13))
- Fix, DisableDevice returns 200 (not 204) ([`9f75bb8`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9f75bb893a07dc46c0f4351b43ef3e78cc5e373e))
- Improve entity naming for DeviceIP to Address ([`4b1a57f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4b1a57f980d2720807accba271c5b3bae91196a4))
- Add ping feature to auto-update device IP based on the request ip ([`3efa415`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3efa415479002d004f55925c2db7cd14513b8356))
- Refactor code to better separate domain layers, add ipv6 support ([`3a24a28`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3a24a28f4ec294cd761d4977e940048a969040a3))
- Refactor ipv4/6 parsing and validation ([`d8aeb2d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d8aeb2d940078c41c78d2ecf93d28703fdde5eba))
- Add address_status table and work on returning the latest status via a DB view ([`1dfc0dc`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1dfc0dcc8db610b77823a8fb7e577125f0a82c3e))
- Ensure address ownership before disable/enable ([`3a755d6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3a755d6e12a6d678937a087015f31f86304495b1))
- Update tests and fix issues ([`50bfce7`](https://github.com/DiegoGuidaF/PulseWeaver/commit/50bfce7223499e060bcc80fffe34b85ed03c39ea))
- Minor improvements to REST contract and response ([`0715cf9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0715cf9d8484b653d63cae01b8701d1c7bd95780))
- Share business logic between assigning an address and the device heartbeat ([`8027676`](https://github.com/DiegoGuidaF/PulseWeaver/commit/80276760b680c1397767cf94e8cfd5b0dab5a1d4))
- Add authentication layer and users ([`165faa4`](https://github.com/DiegoGuidaF/PulseWeaver/commit/165faa4c0dc1e6c75323d4179447f7341067b9dc))
- Add user signup ([`6ffd0b2`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6ffd0b226f9cefda17e6ec40b6eddc337ad4fb05))
- Validate authorization and inject it. Improve middlewares code and segregation ([`8c99c0f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8c99c0f308fbd2e4bff05f3e2b07d7d4e51cb133))
- Add logout endpoint logic and improve cookie creation ([`abd058e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/abd058eb1ab04657bfcc7723846c7c66125a70c6))
- Only allow admin users to register other users. Create admin user on first init ([`f0b7702`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f0b770215078cbf607ed6ab44bbf02c7e730bc54))
- Refactor - Improve layer separation and quality ([`de554ef`](https://github.com/DiegoGuidaF/PulseWeaver/commit/de554ef48eafe8254cf7b5b4f46d4541eb5d59aa))
- Test cleanup. Remove non-happy-path tests and add missing ones for authentication layer. ([`cd4ef57`](https://github.com/DiegoGuidaF/PulseWeaver/commit/cd4ef5729cf2ec36eea87ef49e75cbda72b93ffa))
- Refactor device repository and authentication repository to unify naming conventions and improve type consistency across the codebase. ([`900a90a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/900a90a58b74cdb045e5f4aa90e6b2fdbf7d3b5a))
- Improve test structure and add missing service unit test and domain entities validation tests ([`6bd10be`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6bd10beb3fb43006dab177c7edf1e1e460f83497))
- Add "me" endpoint to retrieve current user details ([`5610c92`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5610c92c4cdb3747c0567364e24f562353eab70c))
- Update Go to 1.26 ([`2b351dd`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2b351dd023d2aafd9bbf346aa7233a40bd3182dc))
- Add concept of device api key ([`8af459f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8af459f4589def482611b8c3f04352ee434a6f8d))
- Add tests to the device heartbeat via api-key feature ([`645286d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/645286d5503ce31d3f19c372622debd778f691e7))
- Improve security by better handling of contextKeys and trusted proxy for ClientIP retrieval ([`2e3fbec`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2e3fbec4131fb2a95c10a096045bf0693658aa1f))
- Refactor main to reduce logic and split it into proper domains ([`0d4d0df`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0d4d0dfbfb0472aa346afe9d103b34b378a4f9ed))
- Allow sending the IP on the heartbeat endpoint and fallback to request IP if not provided ([`b8cae45`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b8cae459948d549b622c239b9d41aec14508187e))
- Improve test logic and use services for Given ([`d67f11a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d67f11aff8a44090e15eeaed401035b81c04baf1))
- Allow not setting TRUSTED_PROXY and do not parse XFF header in that case ([`552724a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/552724ae12bc76d3e2adafcafd04534b2402187f))
- Add and improve logging for devices module ([`d34716b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d34716b66edb4106cec6e5f967fcdd2f85802e53))
- Add auth logging ([`2a98e69`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2a98e6980546c1e9c60fe1d1636bb907d97e3d1b))
- Improve naming ([`e406e28`](https://github.com/DiegoGuidaF/PulseWeaver/commit/e406e2819701ea562a237f256a61d912e904a990))
- Improve logging output and setup via env variables ([`a344bb7`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a344bb72bc83c9bc24a78220dc494632dde9f5a0))
- Improve package visibility by having the repository return a struct and the service define a private interface ([`1a2834d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1a2834d1a383c3bd9fae5df4cfbb72e8e6c0e179))
- First proposal of whitelist generation feature ([`80e58b4`](https://github.com/DiegoGuidaF/PulseWeaver/commit/80e58b44b1f5a76b515fe6a1f177dcf4b4e6b86f))
- Only write whitelist if there are changes ([`d6cc363`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d6cc363157a1b9442c1147ea99ff9e4bbb7aadfe))
- Replace sqlite library with one non-dependant on CGO so that no cross-compilation is required and app is easier to dockerize ([`d9a07b2`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d9a07b2f91b788987b30b1bcba3317947b34af40))

### Bug Fixes

- *(backend)* Extract go tools to separate module in order to not polute project dependencies. Lint and openapigen ([`9397ee0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9397ee06d5fa3db1bda24b0850b7a05c8d0a348d))
- *(backend)* Lint - Apply and use golanglint ([`2c1e894`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2c1e89447ea43bad4dcfe9248017b3fb963a40e8))
- *(CI)* Try to fix pipeline linting ([`2dea1f6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2dea1f671503cb7c841811a17012534a0fbd6681))
- *(CI)* Try to fix pipeline linting V2 ([`413a95d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/413a95dcc5ae05f795dc196a2161a9f165b3d939))
- *(CI)* Do not lint ui-prod since it depends on build-time folder ([`8ff7049`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8ff7049a823ae6461c91a39dd2d285ca8955f9ea))
- *(backend)* Whitelist should be generated on first signal and rate limited to once after RATE_LIMIT time. ([`11dadc0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/11dadc002fb2aee192b46064803b93a2ef465692))
- *(backend)* Improve process interrupt handling. Main listens to interrupt and context forwards cancellation ([`2ae8d50`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2ae8d505f53492a24468c85d3614ee0760ebe38c))
- *(test)* Use temporary directory for whitelist creation so there are no leftovers after testing ([`f2550cd`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f2550cd483436a2d24e0fee87ab3107847c3a0b6))
- *(lint)* Linter fixes ([`ab91d7a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/ab91d7a117e2117fbeb36efa832b01444efdfea4))
- *(ui)* DeviceDetailPage now uses proper types for queries ([`48be539`](https://github.com/DiegoGuidaF/PulseWeaver/commit/48be539f4b7a8e9031889c94e5333f60879c2ed6))
- *(backend)* Fix failing tests ([`f49f7f0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f49f7f04013527ed091239938dfde9e2e54af024))
- *(ui)* Remove unused parameter ([`832bec1`](https://github.com/DiegoGuidaF/PulseWeaver/commit/832bec11f6c5064f6add4b4edfdb6ce0f7762b52))
- *(ui)* Renamed response model address "status" to "is_enabled". ([`db915c2`](https://github.com/DiegoGuidaF/PulseWeaver/commit/db915c201ecb45b2758035b4f698c560a7ac5cf4))
- *(backend)* Use UTCTime for the address lease expires_at return via API ([`f40ff3a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f40ff3ae10ac934c32a07d82cb8a2e8b7bc2f913))
- *(ui)* Pass localstorage to vitest else it fails with node v25+. Also cleanup the tmp localstorage files. ([`1626e93`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1626e93492ea896a2bcbde622bb5a23fa327efdf))
- *(ui)* Reset update user form on success ([`d40ee25`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d40ee2504af0f146bef2dd787ee7c0aa38f10f3a))
- *(backend)* Linter fixes ([`904df25`](https://github.com/DiegoGuidaF/PulseWeaver/commit/904df257a53e78db1d32aa9d3bf283f691231ae1))
- *(backend)* Properly record heartbeat event source when creating an address (bug was that it set it to manual) ([`9567422`](https://github.com/DiegoGuidaF/PulseWeaver/commit/95674224d6d361ecda0f1887ca05a8e90f5dbdfd))
- *(api)* Include disabled addresses in last_seen_at ([`3e178cd`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3e178cdde2baf7bdfe3e768227e1e368ce5755e2))
- *(ui)* Linter complaints ([`3f3045f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3f3045fac1d3fb8058e16cce1d43f496ea209ce6))
- Set geoip folder for docker image ([`a756bf9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a756bf99884227c8672fc36c05c0d7f46da09425))
- *(tests)* Increase defaults timeouts to reduce flakyness ([`bd7f9ba`](https://github.com/DiegoGuidaF/PulseWeaver/commit/bd7f9bab921766a4c89d19be17d4b4ecf341d9c2))

### Features

- *(logging)* Add conf variable to enable/disable LOG_COLOR ([`953a3ff`](https://github.com/DiegoGuidaF/PulseWeaver/commit/953a3ffa74467abfc607b7190f0514e77a8cf8be))
- *(ci)* Add basic CI for test and build ([`3cee973`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3cee9734c5d8d608864c158ca37076dd58a02ae9))
- *(ci)* Push latest main image to docker registry with "dev" tag ([`38046d9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/38046d97b8c085c455df35fb4f2414ee456523c2))
- *(ci)* Fix push latest dev image needs lowercase repo name ([`c4d2816`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c4d2816626ef12296174bf0d09aa7327886977f6))
- *(ci)* Also push dev to a sha-{commit_id} tag ([`6cbfd74`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6cbfd74b5490a3c12c970e8931ee9834cd485f31))
- *(backend)* Allow (soft-)deleting a device. Hide it on endpoints ([`d124df1`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d124df130abbfc89e1dd5e0b9f03fe496342cf5e))
- *(ui)* Button to delete device along with improvements on duplicate device name handling ([`3df17d8`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3df17d8854781bd5badf34fe216ec9fc52e3ddf6))
- *(backend)* WIP - Add rule engine and address lease (auto-expiry) system ([`aef34f9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/aef34f91ae179cbadf1bd1036caf0ba331fa690e))
- *(backend)* API to add/retrieve rules for address lease ([`611c743`](https://github.com/DiegoGuidaF/PulseWeaver/commit/611c743f43465fa6796eaf31bbbb83b14261beca))
- *(ui)* Add rule management in UI. First proposal - WIP ([`8b652dc`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8b652dcee0c1d1317c1cfe410406cd5540020d4e))
- *(backend)* Allow retrieving a device via API Get call ([`0bac98c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0bac98c9e7c651017d0df4a71c4e438d680009d7))
- *(ui)* Add DeviceDetailsPage and improve device address list rule UX ([`9913e17`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9913e1737621ca4c50f3073e68a9dec063dba035))
- *(backend)* Add caddy reloader to send api call on whitelist regeneration ([`1edbe40`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1edbe40d88545b412556057801e0c4b275de9ec8))
- *(backend)* Whitelist regeneration now has a small debounce so that if we disable multiple addresses we wait a bit until the last one before regeneration ([`af88be7`](https://github.com/DiegoGuidaF/PulseWeaver/commit/af88be7266a82ce2c576d93c5f5d01e3ef43acfb))
- *(backend)* Generate a whitelist.txt compatible with Caddy import - WIP ([`5ef2efd`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5ef2efd17f16d70158006b90995f2134a3d7c617))
- *(backend)* Empty caddy whitelist has non-routable dummy IP (to ensure Caddy forbids anything) ([`1ce7e81`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1ce7e8118622add21f693ab0f66590351054244a))
- *(backend)* Add authz verify-ip endpoint and service. Allows much easier integration with proxies. ([`aec7dc6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/aec7dc66418a3490eb53015693363dca7c8a7892))
- *(backend)* Parse clientIP from X-Real-IP header if TRUSTED_PROXY is set. Add extra checks to the IP when creating an address ([`c1649af`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c1649af91912f4e3e4a3afa7145e47725c177b30))
- *(ui)* Auto-refresh address list and allow quick heartbeat to device via browser ([`03531f6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/03531f6ebff01d47dfb08ed7cf45e4cffeceeaf2))
- *(backend)* Add CQRS lite abstraction with GetAddresses to return address with expires_at from lease domain. ([`c36c15a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c36c15ac54a88eff125c262c4ad8d10fe51b7a31))
- *(ui)* Add Expires info to device address list ([`560b407`](https://github.com/DiegoGuidaF/PulseWeaver/commit/560b4079fe51d002e0ec26efa6a228d623ff8bea))
- *(backend)* Update device address leases on rule config update ([`0813564`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0813564ed63555c14f789f9744f79c14ea14cbe2))
- *(backend)* Add device list enabled addresses count ([`6b86444`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6b86444ee403344245a5b78350594ea9fd62ac00))
- *(ui)* Add device list enabled addresses count ([`9d72858`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9d7285868137b5ad72fd04dd3b02ddd614d0263f))
- *(backend)* Add user management API calls ([`48a0883`](https://github.com/DiegoGuidaF/PulseWeaver/commit/48a0883b5e99f132356b749dc3bbb35f11c6cd62))
- *(backend)* Improve login api rate limit and add basic one to heartbeat ([`93cce24`](https://github.com/DiegoGuidaF/PulseWeaver/commit/93cce2478bbe0af8fefdecfb3c589302c51c926a))
- *(ui)* Add user management section allowing to update profile and create new users ([`5f6757b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5f6757b04df214bfefff3e8236b21d3da9a8f37a))
- *(ui)* Allow setting a device to auto-register the browser IP on it periodically ([`24a99a0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/24a99a0c289fed576039c45c4c4ddfbef2d1a0b3))
- *(backend)* Allow regenerating a device's API key ([`ea7a7cd`](https://github.com/DiegoGuidaF/PulseWeaver/commit/ea7a7cdcfd58868539f565107876a5892f691e3f))
- *(ui)* Allow regenerating a device's API key ([`df3b012`](https://github.com/DiegoGuidaF/PulseWeaver/commit/df3b01206a42a7fcd32988b29e38a14648bccb83))
- *(backend)* Store policy audit of each request (DB write in batch) ([`38c16e0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/38c16e0810b58e628f598691eaf033b59d99259f))
- *(general)* Rename application to PulseWeaver ([`03022d0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/03022d0fa5090ae443bd8a3194ee295026e70c47))
- *(backend)* Improvements to the audit flow ([`700d573`](https://github.com/DiegoGuidaF/PulseWeaver/commit/700d5737c040364119ff3b5db3c12fa540e29819))
- Add requests audit log UI along with fixes of backend so that it doesn't return nulls but empty/valid data ([`f9658d5`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f9658d544a7273b0694d960d013c889a08840d89))
- Show address history as global page allowing device filters ([`082e2dc`](https://github.com/DiegoGuidaF/PulseWeaver/commit/082e2dcb7db5cca35c123ae747e0bab8afce9d7e))
- *(ui)* Active table filters are visible via chips ([`91d4901`](https://github.com/DiegoGuidaF/PulseWeaver/commit/91d4901b60e225007d0a4720b55be6872f414507))
- Address list allows showing only events that resulted in state change ([`bbaef98`](https://github.com/DiegoGuidaF/PulseWeaver/commit/bbaef98c9590429bdc96e419e0f8bd342e753b61))
- *(backend)* Add geoip to the access log. ([`8c8c15b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8c8c15bf35c1dfcb77f47fdfb79f0ea0164450b1))
- *(ui)* Improve devices list and device detail header ([`6c47882`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6c4788268630b1a45fd1cf4545d4ab7876f45e57))
- *(ui)* Show country code data for the requests in the Access Log ([`728ce09`](https://github.com/DiegoGuidaF/PulseWeaver/commit/728ce096fce83c96f002bdbbc76c64d9ea420144))
- Add requests traffic map by country code ([`839982c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/839982c137fb915b58e229621bf19a05b86841af))

### Front

- Generate typing from server openapi api.yaml ([`9c10234`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9c1023425fdda6f42c10a5aa96c2883f511ac8ff))
- Easy back/front dev run via makefile. ([`ef53453`](https://github.com/DiegoGuidaF/PulseWeaver/commit/ef534531e5d9c6d88dd118005a0a892b3a98c19f))
- Basic skeleton UI with all features working. List, add & disable ([`7358ab6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/7358ab6192e1a0346249939e2997ca395b0d6d30))
- Add toasts on succ/err, manage loading state and skeleton ([`b03a577`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b03a57734df14861f0beab53a9097f77813552b1))
- Fix css. Add more modern UI with AppShell ([`103d5eb`](https://github.com/DiegoGuidaF/PulseWeaver/commit/103d5eb9dccc491b06a3aeac490c1157a46138c3))
- Allow dark/light mode ([`6ba07b2`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6ba07b240fcc57bd714fe2440843b88f966b4e18))

### Frontend

- Update latest changes, not tested ([`6cdb11c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6cdb11ceef60f21703ba2bdfd2d4b0bafedc0bda))
- Update api schema typing ([`dccb647`](https://github.com/DiegoGuidaF/PulseWeaver/commit/dccb647131464402fc3ac11c33737289a35883eb))
- Implement user authentication (Login & Logout) ([`82d7fe5`](https://github.com/DiegoGuidaF/PulseWeaver/commit/82d7fe524e8f7905b9322904889da63624e3c980))
- Improve typing usage from api generated types ([`7cbe9f9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/7cbe9f9c5cfa0b3910d6fe2b579091f32e4ffdf3))
- Replace typescript generation with heyapi one and validate inputs/outputs from the API via zod autogeneration. ([`df1106b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/df1106b1d5834e37a1926effceb768a35cfa848d))
- Use the generated client for most http functionality. Reduce unneeded code and try to only have UI related logic. ([`b100d63`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b100d638b5349c722ed212ee678645e56238fba0))
- Improve login/out logic and simplify it. ([`79e509a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/79e509a52d95d9e37de7bf0a4fad1338b78b6bb7))
- Add test structure with a first test for DeviceList ([`5d0ffff`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5d0ffff0e8ed856456637912015d92f1b87bf6fb))
- Add testing along with simple API mock handling (handlers.ts) ([`487de10`](https://github.com/DiegoGuidaF/PulseWeaver/commit/487de10990a0869e5f3ba3933c3e5f15b36f05ec))
- Update API generated code ([`6b7e07a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6b7e07ada7eafe5c12cf1399c4c0eece49fba453))

### General

- Add documentation to improve AI based suggestions and tips ([`58bcf97`](https://github.com/DiegoGuidaF/PulseWeaver/commit/58bcf978f73c8948467e210356377bb76a1dffdf))
- Change from .cursorrules to the new format in .cursor/rules/*.mdc ([`168282e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/168282ec58d034033123d388f36e5d84cf1ba5d7))
- Remove unused bearerAuth from openapi spec ([`6e5fa03`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6e5fa03438b072e90487c54a7eac8a92bbb44a50))
- Goland ignore frontend dist directory ([`8a3dd5f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8a3dd5fc57b88bb67973a27be9282e86e01f69ca))
- Move openapi spec and configurations to root and keep backend specifics to internal Go package ([`ab3be3a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/ab3be3af2359d21a545d95b7fd2443d4db8a8788))
- Add dockerfile along with compose for easier development ([`492c64e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/492c64ec7492130b577b6fe2b76c5a32decc54c6))

### Miscellaneous

- *(lint)* Improve linting ([`1d538dd`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1d538ddfb997d35d5d197c0e03e5c2e9cd182591))
- *(backend)* Replace Go "dev" tag with "prod" ([`742854f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/742854f58f5cd76146bb4105272b27f03b274ec8))
- *(backend)* Add debug logging to request XFFF header ([`b3ab039`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b3ab039f24ef16859fd5dd573cfeb36bd11b3b18))
- *(ui)* Show api-key on device creation and show prefix on device list ([`2a20c1b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2a20c1b455134c5a55d506b98d8e0b532dc3ecdc))
- *(ui)* Ensure CreateDevice api key shown dialog can only be closed by clicking the right button ([`046c1af`](https://github.com/DiegoGuidaF/PulseWeaver/commit/046c1afc3fe68ab6b4f061d1c40a99dfd049d372))
- *(backend)* Refactor. Abstract domain entities from DB ones, separate signals for whitelist regeneration ([`573eb1e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/573eb1ecc1f53f1325d3a5cf33834dc499d2a4d2))
- *(backend)* Squad DB migrations to reduce noise ([`80939c0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/80939c0cfe99acda87061a336feed3e5649acbb9))
- *(general)* Add make api command that builds both front and back APIs via openapi ([`eb3c0ed`](https://github.com/DiegoGuidaF/PulseWeaver/commit/eb3c0edcbc5e3118ee2bd340c9fdcad86e9f99f3))
- *(ai)* Remove Cursor rule to fix plans since it is still not working ([`35c4759`](https://github.com/DiegoGuidaF/PulseWeaver/commit/35c47598b552f0b8ef0bd5a01af9c96cafa29e75))
- *(general)* Minor visual fixes to makefile and CI ([`3103e00`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3103e000e7c555e148ec032278576f47c628cd3b))
- *(backend)* Refactor service orchestrator from channels to an interface with observers. Channels are for internal service processing ([`5d1c23d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5d1c23db53acf8d92e640c4a23c180cf04c4c70e))
- *(backend)* Add tests for lease service ([`9ae86a8`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9ae86a86a3db7e5575796ac9627559a238f88257))
- *(backend)* Remove sleep calls from caddy client test ([`65db93b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/65db93b222e1e9a1e4a256bfa96a288e7d869847))
- *(ai)* Add CLAUDE.md ([`8b13d47`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8b13d47053a2f921d9a0784a64a674ff9fc14955))
- *(backend)* Improve domain separation. Service as repository gate-keeper ([`0a7f877`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0a7f87756543e6ab86449f7504df62557c4cc507))
- *(docs)* Add architecture overview documentation ([`b2a545a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b2a545a96ef5f0f37afbf360953c9e05263aa14d))
- *(backend)* Simplify and reduce logging ([`01d6a8a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/01d6a8a8a03b82b0abad5664f4bcc88093cef68e))
- *(backend)* Do not store logger in context. Use slog native Handle interface ([`1f0ca62`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1f0ca62d2cd8f0dfd34cd26ed7634a3a0a8e9348))
- *(backend)* Ensure caddy reloader env variables are valid if defined ([`b014133`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b014133747bd73b8f94f377e23dedef975e405cc))
- *(backend)* Improve cancel context handling and keep track of subroutine cancellation ([`21f1aae`](https://github.com/DiegoGuidaF/PulseWeaver/commit/21f1aae4acf59cde3eb19a3f87c9221ad1b0b9cd))
- *(ui)* Make dev-front ensures npm packages are installed ([`c82c380`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c82c380c65119d7f63b9c534fdd206cf20dd4278))
- *(ui)* Separate DeviceDetailsPage into smaller components and other minor improvements ([`e817c7a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/e817c7a15d7390a6e25f657df732fe66707d2ab4))
- *(ui)* Reduce Login component into component and restructure it. ([`b78ebcc`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b78ebcc8c71e02fbd4fcfd94c9eca8421b48add3))
- *(ci)* Fix cache usage for Go tools ([`13c79c1`](https://github.com/DiegoGuidaF/PulseWeaver/commit/13c79c12c6e3b0680ca5ecddd6e8772b3e2fb992))
- *(backend)* Simplify building the caddy whitelist by just concatenating the IPs into a single remote_ip matcher ([`4c9f8c5`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4c9f8c53745a74d55e3fb99780e092b9cf92f452))
- *(backend)* Remove no longer needed whitelist + caddy reload ([`3ec6624`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3ec6624454747876bff9ec89bc749532d4a0cfa0))
- *(general)* Update README to reflect latest changes on features and application usage. Make it ([`3cf33a3`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3cf33a3f3b05fcd12ef8cd63b59367d80b706eb1))
- *(backend)* Remove mentions of initial Forgejo deployment with the new one in github ([`4eeb9f9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4eeb9f96bfec3662840c47f78c7f9e283925cc74))
- *(backend)* Default to LOG_COLORS on for text log output ([`20cc217`](https://github.com/DiegoGuidaF/PulseWeaver/commit/20cc21797a6d1700355d28f88632b6b6ab6ca910))
- *(backend)* Minor comment fix ([`8dd71fd`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8dd71fd9c32c8969a81c3447f84118820314efb9))
- *(backend)* Improve authz inner domain separation. Handler for HTTP, service for business logic. ([`29c1aca`](https://github.com/DiegoGuidaF/PulseWeaver/commit/29c1aca17e856caa5f13ac57c9a37be2a7802d0b))
- *(backend)* Improve lifecycle management and context cancellation ([`a9e50a2`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a9e50a2178db0b92cf0582d3f9c59954eab8be62))
- *(backend)* Do not update authz address cache if address status didn't change (ie. a heartbeat was received but address was still enabled) ([`01e82cb`](https://github.com/DiegoGuidaF/PulseWeaver/commit/01e82cb8725f253eec7829c14f0f96363e67c978))
- *(backend)* Log only mutations at handler level, improve log colors and centralize common categories ([`30ec364`](https://github.com/DiegoGuidaF/PulseWeaver/commit/30ec364c4ebc7d02f290c381b0a22e969849a18e))
- *(backend)* Remove leftover mentions of the whitelist ([`548b318`](https://github.com/DiegoGuidaF/PulseWeaver/commit/548b31855ec4975a9aa8d2473809458e6e1120d1))
- *(backend)* Improve TZ management. API returns UTC RFC3339. Build wrapper API method to ensure this happens. ([`1a6259c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1a6259c0e4142e6e02f82a8ff100642daa7fb6ee))
- *(backend)* Set DB_DIR instead of file. Default to /data in docker but ./data for local ([`fa7dbca`](https://github.com/DiegoGuidaF/PulseWeaver/commit/fa7dbca1d1da913947772176798a04e7e9aeb75d))
- *(backend)* Remove deprecated expiry concept from codebase (leftovers not used) ([`f6e00ea`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f6e00ea84cd30353bc3164ab2e3aa0e1afb5d55d))
- *(ui)* Add missing tests and reorganize existing ones ([`868f590`](https://github.com/DiegoGuidaF/PulseWeaver/commit/868f590f8ee54ca3433a5c0a8ceffc6998260ef6))
- *(general)* Remove IDE folder to reduce noise ([`eff80c8`](https://github.com/DiegoGuidaF/PulseWeaver/commit/eff80c88d407c388f6f8c495b64ae14a090a727e))
- *(general)* Rename authz new domain to policy to better reflect what it is ([`9705422`](https://github.com/DiegoGuidaF/PulseWeaver/commit/97054227f506330c14b74fac48f51c6d868ce681))
- *(backend)* Rename address_status to address_events to improve the domain naming ([`70fc65c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/70fc65c34848faddb9e4d15bb3548b217f246b2b))
- *(ui)* Use latest node and set in .nvmrc ([`5a6b220`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5a6b220f2b6619c0665a5f9cf0660672e6c836d7))
- *(general)* Improve & update cursor rules ([`02db85c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/02db85c0a6b84e27e6822823ce7b67a3b2fcf9ed))
- *(backend)* Ensure all tests are not built for prod (reduce binary size). Minor fixes to policy domain test naming ([`0d866fe`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0d866feea241d6ff714568c5bd39d089c47888f0))
- *(backend)* Update dependency versions to latest ([`fd422fa`](https://github.com/DiegoGuidaF/PulseWeaver/commit/fd422fafa10b0fa84914b1a2d66813e689dede4d))
- *(ui)* Remove unused dependencies and update current ones to latest ([`1c2bcc0`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1c2bcc0f6df63028f957aa999155bcc48f2e2ce7))
- *(backend)* Add missing testing for the new user management - WIP ([`47e515c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/47e515c2d4386066ea27bf6d3dcfb0ccbdc21cde))
- *(ui)* Add missing testing for the new user management - WIP ([`9c63fac`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9c63facea3e62e56fa14941d00dedb7691884c46))
- *(ui)* Improvements to browser heartbeat ([`ef5fb67`](https://github.com/DiegoGuidaF/PulseWeaver/commit/ef5fb6738f8ff8c957e9623e0b918636dad8f1de))
- *(ui)* Small fixes. Username must be lowercase. ([`6e1a72b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6e1a72bc15d3590511700666295e1dcc2b024919))
- *(ui)* Improve device api key regeneration flow. Pending other imprrovements ([`3efe923`](https://github.com/DiegoGuidaF/PulseWeaver/commit/3efe923f918f4ca17c114c6e0573557094937dbf))
- *(ui)* Improve test handler setup and preload happy-path responses ([`b67b69f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b67b69fc371efbc4b187053bf02650ba7997b08f))
- *(ui)* Migrate to Mantain - Step 1 ([`195f742`](https://github.com/DiegoGuidaF/PulseWeaver/commit/195f742358b8e23739fab0900075178751819a60))
- *(ui)* Migrate to Mantain - Step 2 ([`0debae6`](https://github.com/DiegoGuidaF/PulseWeaver/commit/0debae637c62ae0e014ad50833628231badac910))
- *(ui)* Migrate to Mantain - Step 3 (replace Shadcn components) ([`b1680a2`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b1680a2365912af73e4e7a97979cf4bd5e62a1a4))
- *(ui)* Migrate to Mantain - Step 4 - Replace form usage with Mantain forms ([`e02921a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/e02921ae3f2248742ec5d95859e06054347982ae))
- *(ui)* Migrate to Mantain - Step 5 & 6 - Notifications into component instead of mutation ([`5b860ea`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5b860eab72fd668376dafde2c6bf0e6402a1f3b2))
- *(ui)* Migrate to Mantain - Step 7 & 8 - Dark/light mode and final cleanup ([`17af77d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/17af77dd63691ce05dc4719d9df433250e651cbd))
- *(ui)* Minor review fixes ([`42d7493`](https://github.com/DiegoGuidaF/PulseWeaver/commit/42d749384d8f0fabf911d5242b6e8690ce69b534))
- *(backend)* Write test logs to console and remove verbose flag on make test (only show console output for failed tests) ([`a175200`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a1752006b1b87c5668aaddc0ae0c46edb001df7e))
- *(backend)* Improve user management flow and separate promote and demote user actions ([`fb8353e`](https://github.com/DiegoGuidaF/PulseWeaver/commit/fb8353e23e337f15c1a8c42c4db0a3fd1c8c0d19))
- *(ui)* Improve user promotion demotion flow by using the new separate API mutations ([`666da5f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/666da5fb4fbf7f71eca5a76e14da7723bf31cb6f))
- *(backend)* Make lease init explicit on app launch. Better structuring of queries package ([`18573ba`](https://github.com/DiegoGuidaF/PulseWeaver/commit/18573baa6fcd6750900243071c267cf7140c2f49))
- *(backend)* Simplify DB by removing address_current_state and embedding it in the address table. Now it is not immutable but we still have an address_events that should fix that requirement. ([`11fca67`](https://github.com/DiegoGuidaF/PulseWeaver/commit/11fca673c512bedf3edc2bfbe8dc5342f21c52e8))
- *(backend)* Improve auth ip verification endpoint debug logs ([`860bcc9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/860bcc9d0303180ff073423f80f8e5bcbfcfd9c7))
- *(backend)* Obtain the denyReasons dynamically via query instead of hardcoded list ([`e4f08c9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/e4f08c939d478afa15849289e3a1eadf92d6cd62))
- *(ai)* Improve CODEBASE docs and ensure CLAUDE reads it more often. Intention is to improve quality of Claude proposals ([`f74ca52`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f74ca52898b795b00a6bffdfbff5d75cd851cf8b))
- *(backend)* Refactor on general code quality (no design changes). WIP ([`5b61be3`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5b61be3779c2dcd6b4d34f72d85cabe0452c6bfc))
- *(ui)* Add auto-refresh button to request audit log page ([`1a08484`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1a084842ddf1b6842baaf7b70f530972f60c7d6a))
- *(backend)* Simplify header management. Store all except of typical headers with tokens. Start managing migrations from now on ([`1914a94`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1914a94d41695b64d04a38d6477acb8cc0096e84))
- *(backend)* Improve http error logging ([`fee1932`](https://github.com/DiegoGuidaF/PulseWeaver/commit/fee19322fe5f33f899bd8eaddbe4ceec821b5816))
- *(ui)* Add preset to URL query parameters. Persist it in local storage and allow quickly clearing filters ([`8d8b712`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8d8b712540013854bc4a02c777cff728f16a771f))
- *(ui)* Improve audit log date range filter UX ([`837ea67`](https://github.com/DiegoGuidaF/PulseWeaver/commit/837ea6735e6638403b22e0573bd97f9bed332dcf))
- *(ui)* Fix linter errors ([`1b4cf45`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1b4cf45860eb8b70e3c0b9206cc7f902a13366d1))
- *(backend)* Add address lease history endpoint ([`86ced42`](https://github.com/DiegoGuidaF/PulseWeaver/commit/86ced424c978656b5a041c3696e6e2c99e05d6be))
- *(ui)* Add address lease history page on device detail page ([`2efb77b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2efb77be38b78cb4a60934cd5c2843bcd8cf3916))
- *(frontend)* Make address history global and per device pages show the same and allow filters ([`b2178e7`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b2178e7627fddaf9d49ffc4ce7d13136d621cc59))
- *(ui)* Improve settings page ([`5b22bbc`](https://github.com/DiegoGuidaF/PulseWeaver/commit/5b22bbcb66334aa5c773b373c2b131fba625f944))
- *(backend)* Dashboard queries by doing a traffic aggregate via scheduler on 1h ranges ([`9e870a1`](https://github.com/DiegoGuidaF/PulseWeaver/commit/9e870a17db4a4c7e29c9ab5aafb0ed898eb6462a))
- *(ui)* Add dashboard with overview of traffic statistics ([`d67447d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d67447d06d74db46bc4010078a9d92839b18d07d))
- *(general)* Ignore playwright files ([`7edb770`](https://github.com/DiegoGuidaF/PulseWeaver/commit/7edb77063f53bb63e069f766d6ca2580836dab36))
- *(ui)* Update packages to latest versions ([`b0a8649`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b0a864954554f0e5a676e812ca4d6684588ca89d))
- *(ui)* Move AreaCharts to LineCharts for address and traffic ([`8d143d4`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8d143d44f8d00ec7670594fdc63806d6057c6467))
- *(ui)* Time range preset is not cleared by "Clear Filters", that's only for table column filters ([`d14fc17`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d14fc1720880dffde030efcd5105c1a3c893c683))
- *(ci)* Ensure frontend linting happens on testing stage ([`cf5a165`](https://github.com/DiegoGuidaF/PulseWeaver/commit/cf5a16554ab95cc7da8b6249836c9dc89db591ee))
- *(ui)* Improve mobile sidebar UX ([`d25238f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d25238f15cea789c79a9c731a31262670fc28adc))
- *(ui)* LineChart remove point labels since they add too much noise ([`f451434`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f45143481cf5af2fc189effa8febbc73ebb6372e))
- *(ui)* Improve account tab UX ([`1e1960a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/1e1960ae7f2cf20cc4497c2e7fa8c1ae2a4c80ec))
- *(backend)* Order address query by updated_at. Minor test placement improvement ([`a260c1b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a260c1be768a589e8ff02befcfe4893e7c133977))
- *(ui)* Improve UI color scheme ([`5705283`](https://github.com/DiegoGuidaF/PulseWeaver/commit/57052830b6cbe6418661091b2d51ebf6aefbbaf7))
- *(ui)* Add shared components and polish dashboard ([`6c25308`](https://github.com/DiegoGuidaF/PulseWeaver/commit/6c25308faaa04fb332906a9d48bf7f9d588d3b13))
- *(ui)* Redesign device detail tabs ([`241989b`](https://github.com/DiegoGuidaF/PulseWeaver/commit/241989bf7f1b5a9afd75aaccb557ca91aab93bc5))
- *(ui)* Polish access log and settings pages ([`8855356`](https://github.com/DiegoGuidaF/PulseWeaver/commit/88553564a5cbfd5c89f5a2a8707c075b721d1a32))
- *(ui)* Auto-refresh device header every 10s ([`42e192d`](https://github.com/DiegoGuidaF/PulseWeaver/commit/42e192d3038786466676d5b38690148fe65b6043))
- *(ui)* Make TTL value more prominent in settings ([`b1c7419`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b1c74195405639a42074c180333e2cf2907adbf3))
- *(ui)* Add BrandName component with amber Pulse ([`23f0c82`](https://github.com/DiegoGuidaF/PulseWeaver/commit/23f0c82e8d7c6b05ceb761531360dbbb5d1d311a))
- *(ci)* Add release pipeline with git-cliff and conventional commit hook ([`c407248`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c40724845ab5d13a356072ded1763b0e26eee1d7))
- *(ui)* Use Space Grotesk font for brand name ([`8debeee`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8debeee6d5af7e72ed7125e76fa1d2a35814118b))
- Add community and contributor files for public release ([`4ace6db`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4ace6db18c73fd76013c581abae662666967bd14))
- *(ci)* Add Dependabot configuration ([`855a4c9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/855a4c9c69055980dc7c16e1326c78e0b484e968))
- *(ui)* Improvements after reviewing latest UI/UX changes ([`4873e47`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4873e4724f492504ee2ed6063a8fdc19ae9ada55))
- Improve documentation with screenshots and split into smaller for more specific details ([`2ca4dfb`](https://github.com/DiegoGuidaF/PulseWeaver/commit/2ca4dfb070a9931ba138bb77d4eacca79c83c1af))
- *(ci)* Group @mantine/* Dependabot updates together ([`d3b2cd1`](https://github.com/DiegoGuidaF/PulseWeaver/commit/d3b2cd12e6364285a5ab7b5eec348f9bd32b5a75))
- *(deps)* Bump flatted, @vitest/coverage-v8, @vitest/ui and vitest ([`93a5919`](https://github.com/DiegoGuidaF/PulseWeaver/commit/93a59190c77a812601ac616249cd5dd9d058693c))
- *(deps)* Bump picomatch from 4.0.3 to 4.0.4 in /frontend ([`425179f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/425179f76eb667e9f4118b765d5a6417382d8fad))
- *(deps-dev)* Bump brace-expansion from 1.1.12 to 1.1.13 in /frontend ([`be5d728`](https://github.com/DiegoGuidaF/PulseWeaver/commit/be5d7286af1829c7c71d0899eefe1c6735b60be1))
- *(deps-dev)* Bump happy-dom from 20.8.8 to 20.8.9 in /frontend ([`b208f3c`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b208f3c2495bcee70ff5f4ebe760670c9462c45f))
- *(deps)* Bump actions/checkout from 5.0.1 to 6.0.2 ([`b1fe8ea`](https://github.com/DiegoGuidaF/PulseWeaver/commit/b1fe8ea7e996a8c68f6522a3e4662c4ffc977f5e))
- *(deps)* Bump golang.org/x/crypto from 0.48.0 to 0.49.0 ([`c453b0a`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c453b0ac9c5561dc92396f1fa203731480e3ed14))
- *(deps)* Bump actions/setup-node from 5.0.0 to 6.3.0 ([`a388a83`](https://github.com/DiegoGuidaF/PulseWeaver/commit/a388a83ad86450d301de049f85fc2698cb2dcf46))
- *(deps)* Bump github.com/getkin/kin-openapi from 0.133.0 to 0.134.0 ([`e35d18f`](https://github.com/DiegoGuidaF/PulseWeaver/commit/e35d18f39c51280f06cf754c7a958cb21ae5a07f))
- *(ci)* Group Go and Actions Dependabot updates ([`af2af99`](https://github.com/DiegoGuidaF/PulseWeaver/commit/af2af99b7e1a4467e4a783073f9c53a7fc19f3cf))
- *(ui)* Update to latest Mantine 9.0.0 ([`54ea945`](https://github.com/DiegoGuidaF/PulseWeaver/commit/54ea94507cd2f1b93dc3d2d8fef1456e2c13d6c5))
- *(deps)* Bump actions/setup-go from 5.6.0 to 6.4.0 ([`c6199d4`](https://github.com/DiegoGuidaF/PulseWeaver/commit/c6199d4c3c6edbe0ecce96cb80593d9c27a8ee15))
- *(deps)* Bump the frontend-minor-patch group across 1 directory with 5 updates ([`bfb1651`](https://github.com/DiegoGuidaF/PulseWeaver/commit/bfb1651d4a0014847fc7d63b9fb430d2ec907a34))
- *(deps)* Bump the go-minor-patch group with 3 updates ([`53390f9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/53390f956140f46e153e1cca91730df446b1ebef))
- *(deps)* Replace _loc=auto with _timezone=UTC for modernc.org/sqlite v1.48 ([`4909b36`](https://github.com/DiegoGuidaF/PulseWeaver/commit/4909b36a5f7360a6dc16fda87163f9c26bf2b729))
- *(ci)* Set gh actions tags to latest version (not pin to sha) to reduce PR noise and always use latest. Keep dependabot PR for major changes on them ([`f1c7ed9`](https://github.com/DiegoGuidaF/PulseWeaver/commit/f1c7ed946b57e58a57446dd8d65124b7d51680ba))
- Rename RequestAuditLog concept to AccessLog to better represent it ([`fd4f1f2`](https://github.com/DiegoGuidaF/PulseWeaver/commit/fd4f1f283308c6ddf74ae0deb6417791b1abf331))
- Rename request audit to access log leftovers ([`19df503`](https://github.com/DiegoGuidaF/PulseWeaver/commit/19df503347ab4cd0f0228e559b6bb65e76b12385))

### Refactor

- Improve separation between domain models and http models ([`8e20eda`](https://github.com/DiegoGuidaF/PulseWeaver/commit/8e20eda77bea89c8956df66c05c06463f1842a4c))


