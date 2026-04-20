# How-to guides

Task-oriented walkthroughs that coordinate across multiple files and patterns. Read the relevant one before starting any multi-file change.

| Guide | When to use |
|-------|-------------|
| [Add an HTTP endpoint end-to-end](add-http-endpoint.md) | Adding any new API route: OpenAPI spec → codegen → handler → routes → app wiring |
| [Wire cross-domain events](wire-cross-domain-events.md) | One service needs to react when another domain mutates state (observer + channel) |
| [Add a cross-domain read query](add-cross-domain-query.md) | Read joining two or more domain tables (e.g. hosts + access_log) |

For the mechanics of individual layers, see `../patterns/_index.md`.
