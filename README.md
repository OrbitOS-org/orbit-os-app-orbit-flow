<p align="center">
  <img src="https://www.orbit-os.org/images/vscode/orbit-os-logo.png" width="300" alt="Orbit OS">
</p>

<h1 align="center">Orbit Flow for Orbit OS</h1>

<p align="center"><b>Build industrial automation by wiring blocks in your browser — Modbus, OPC UA, S7 and MQTT, running on the device itself.</b></p>

Orbit Flow is a visual flow editor for [Orbit OS](https://www.orbit-os.org/?ref=github-orbit-flow). You drag nodes onto a canvas, connect them and deploy: read a Modbus register every 30 seconds, switch a relay when an input closes, publish a value over MQTT, send a notification to your phone. The flows run on the device, with no cloud service.

It is also an example of a complete Orbit OS app in Go: it uses the [Orbit OS Go SDK](https://github.com/OrbitOS-org/orbit-os-sdk-go) for the serial ports, GPIO, I²C, SPI, system metrics and mobile notifications.

Runs on Raspberry Pi, Arduino UNO Q and other ARM64 devices with Orbit OS (free Community Edition).

> [!NOTE]
> Orbit Flow is based on [LOOPZE](https://github.com/loopzedev/loopze-edge) by Dennis Bleul and is licensed under the AGPL-3.0. This is an early version.

## Features

- **Visual editor** — drag, connect and deploy; several flows in tabs, with a live debug panel and a log view; light by default, with a dark theme
- **Works like Node-RED** — Deploy with *Full*, *Modified Flows* or *Modified Nodes*, and *Restart Flows*; right-click a tab to edit its properties, hide it (it keeps running), disable it (applied on the next Deploy), export or delete it; the list at the end of the tabs shows every flow, hidden ones included
- **Industrial protocols built in** — Modbus TCP and RTU, OPC UA (read, subscribe, write), Siemens S7 and MQTT
- **The device's own I/O** — GPIO, I²C, SPI and the serial ports, through the Orbit OS API; *Serial In*, *Serial Out* and *Serial Request* work like Node-RED's serial nodes, on the board's UARTs, RS-232/RS-485 ports and USB-serial adapters
- **Logic without code, or with it** — switch, change, delay, template and state-machine nodes; JavaScript, expressions or Go when you need them
- **Data in and out** — HTTP, TCP and UDP nodes; JSON, XML and CSV; files in the app's own files folder
- **Notifications on your phone** — a node that sends a push notification through the Orbit OS mobile app
- **Users and roles** — admin, editor and viewer accounts; credentials stored encrypted
- **Examples and saved flows** — the **+** next to the flow tabs → *Open…* adds a ready-made example (for instance, switching the relays of an EBYTE RS-485 relay board, or reading a GPS on a serial port) or a flow you exported with *Export…* from a tab's menu; exported flows are kept in the app's data folder

## Install

**From source, with [Orbit Studio](https://marketplace.visualstudio.com/items?itemName=orbit-os.orbit-studio) (VS Code):**

You need [VS Code](https://code.visualstudio.com/) with the Orbit Studio extension, **[Go](https://go.dev/dl/) 1.25 or newer** (`go` on your PATH) and, to build the editor, **[Node.js](https://nodejs.org/) 20 or newer**.

1. Clone the repository and build the editor:
   ```bash
   git clone https://github.com/OrbitOS-org/orbit-os-app-orbit-flow
   cd orbit-os-app-orbit-flow/frontend
   npm install && npm run build
   ```
2. Open the `orbit-os-app-orbit-flow` folder in VS Code. In the Orbit sidebar, run **Add / Update SDK** and set your device's IP.
3. Use **Build + Deploy** to install the signed `.orb` on the device.

## Getting started

1. Open **Orbit Flow** from the Launcher on your device.
2. Create the administrator account on the first visit.
3. Drag an **Inject** node and a **Debug** node onto the canvas, connect them and press **Deploy**.
4. Click the button on the Inject node and watch the message arrive in the debug panel.

## Development (Orbit Studio)

| Path | What |
|---|---|
| `cmd/orbit_flow/` | app entry point (`main.go`), the node groups built into the app (`groups.go`), `metadata.json` (manifest and permissions) |
| `cmd/orbit_flow/orb/icon.svg` | Launcher and Store icon |
| `internal/flow/`, `internal/server/`, `internal/api/` | flow engine, HTTP server and API |
| `internal/nodes/` | the nodes, one folder per group; `gpio`, `i2c`, `spi`, `serial` and `orbit` use the Orbit OS API |
| `internal/auth/`, `internal/credentials/` | users, sessions and encrypted credentials |
| `frontend/` | the editor (Vue 3); `npm run build` writes it to `web/dist/`, which is embedded in the app |
| `examples/` | example flows shipped inside the app, one JSON file each (see `examples/examples.go`) |
| `orbit-os-sdk-go/` | [Orbit OS Go SDK](https://github.com/OrbitOS-org/orbit-os-sdk-go) — not in the repository; added by Orbit Studio |
| `orbit.project.json` | Orbit Studio project settings |

- Run the tests with `go test ./...`.
- Development certificates live in `cmd/certs/grpc/` and are never committed.

## Security

- **The page is reached directly on the device's network address**, on a port between 50000 and 60000 (52880, or the next free one), not through the Launcher login: the Launcher tile sends the browser there. Orbit Flow has its own login; create the administrator account right after installing, and use it only on networks you trust.
- The app talks to the device's hardware only through the Orbit OS API, with the permissions listed in `metadata.json`: system information, Launcher registration, serial ports, GPIO, I²C, SPI and mobile notifications.
- The file nodes can only read and write inside the `files` folder of the app's data folder, never the app's keys, users or saved flows.
- The message broker inside the app opens no network port.
- Flows can open network connections and listen on ports (HTTP, TCP, UDP, MQTT). Whoever can edit flows can do that; give the editor role only to people you trust.

## Acknowledgments

Orbit Flow is based on **[LOOPZE](https://github.com/loopzedev/loopze-edge)** by **Dennis Bleul** — an industrial flow automation tool written in Go (AGPL-3.0-or-later). The flow engine, the editor and the industrial protocol nodes come from LOOPZE. [NOTICE](NOTICE) lists what Orbit Flow changed.

## Links

[Orbit OS](https://www.orbit-os.org/?ref=github-orbit-flow) · [Getting started](https://www.orbit-os.org/getting_started.html?ref=github-orbit-flow) · [SDK reference](https://www.orbit-os.org/api-reference.html?ref=github-orbit-flow) · [Forum](https://forum.orbit-os.org/?ref=github-orbit-flow) · info@orbit-os.org

## License

AGPL-3.0-or-later — see [LICENSE](LICENSE) and [NOTICE](NOTICE). Orbit Flow is a modified version of LOOPZE, which sets this license: the source code of every version must stay available, and so must the source of any modified version that people use over a network.
