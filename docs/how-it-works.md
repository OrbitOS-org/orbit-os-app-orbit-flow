# How Orbit Flow works

Technical notes for developers: how the page is reached, how flows run, what the app keeps on the device, and where things are in the code. For what the app does and how to install it, see the [README](../README.md).

## The page

The app's web server listens on `127.0.0.1` only. The page is reached through the device Launcher, which forwards `/orbit-flow/` to the app (WebSocket included), so the app is never open to the network directly. The app has its own accounts on top of that: admin, editor and viewer.

| Address (under `/orbit-flow/`) | What |
|---|---|
| `/` | the editor |
| `/dashboard` | the dashboard (every account, viewers included) |
| `/settings` | users, certificates, files, appearance, about |
| `/api/v1/…` | the JSON API used by the page (CSRF-protected) |
| `/ws` | live events: debug messages, node status, deploys, logs, dashboard values |
| `/endpoint/…` | the addresses of HTTP In nodes |

## Flows

- **Deploy** saves the workspace (`workspace.json`) and applies it in one of three ways, as in Node-RED: *Full* (restart everything), *Modified Flows* or *Modified Nodes*. A deploy is refused when two connections would use the same serial port.
- **Engine:** every node runs in its own goroutine with an input queue of 64 messages. When a queue is full, a message waits up to 1 s for room before it is dropped, so a Split of a long list is not lost.
- **Messages:** maps with `payload`, `topic` and any other field; passed to the next node as a copy-on-write clone. Numbers from JavaScript are `float64`, as JSON gives them.
- **Stop / Start Flows** stops every node without changing the workspace (`runtime-state.json` remembers it across restarts).
- **Safe outputs:** nodes that drive a physical output (GPIO Write, PWM Out) put it in their "when flows stop" state (off by default) when the flows stop, the app stops or updates, the node is deleted or its flow disabled. A deploy that keeps the node leaves the output as it is.
- **Errors:** a node error reaches the Catch nodes of its flow (`msg.error`) and the debug panel.

## Hardware

The `gpio`, `pwm`, `i2c`, `spi`, `serial` and `orbit` nodes use the Orbit OS API through the Go SDK, with the permissions in `cmd/orbit_flow/metadata.json`. Pin, channel, bus and port lists in the editor come from the device. A serial port is opened by the first node that uses it and closed with the last.

## Dashboard

The `ui-*` nodes register their widgets in `internal/dashboard` while they run; `ui-page` and `ui-group` connections place them. The last value of each widget and the points of each chart are kept in memory (lost when the app restarts). Changes go to the page in one `dashboard` WebSocket event every 100 ms; switches, sliders and buttons send their use with `POST /api/v1/dashboard/widgets/{id}/action`.

## What Orbit Flow keeps on the device

In the app's data folder:

| File | What |
|---|---|
| `workspace.json` | the deployed flows and connections |
| `credentials.json` | secrets and certificates of the connections, encrypted with `orbit-flow.key` |
| `users.json` | accounts and password hashes |
| `orbit-flow.key`, `orbit-flow.session.key` | the encryption key and the session signing key |
| `runtime-state.json` | whether the flows are stopped |
| `library/` | flows saved with *Export…* |
| `jetstream/` | the internal store: persistent context, open sessions, file-reading positions |
| `files/` | the files of the flows: the only folder Write File, Read File and Watch can reach |

They are kept across app and device restarts, app updates and Orbit OS updates, and removed when the app is uninstalled.

## Code map

```
cmd/orbit_flow/        entry point, node groups built in, metadata.json, orb/icon.svg
internal/server/       HTTP server, static page, Launcher registration
internal/api/          JSON API
internal/flow/         engine: deploy modes, wiring, messages, safe outputs
internal/nodes/        the nodes, one folder per group (core, network, mqtt, modbus,
                       opcua, s7, filesystem, gpio, pwm, i2c, spi, serial, orbit, dashboard)
internal/dashboard/    dashboard state
internal/scripting/    JavaScript (goja), expressions and Go (yaegi) for the function nodes
internal/auth/         accounts, sessions, roles, CSRF
internal/ws/           WebSocket hub
internal/nats/         embedded message server and its store
examples/              the examples of + → Open…
frontend/              the editor and the dashboard (Vue 3), built into web/dist
```
