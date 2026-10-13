// Orbit Flow — visual flow editor for industrial automation on Orbit OS.
//
// Orbit Flow is based on LOOPZE by Dennis Bleul
// (https://github.com/loopzedev/loopze-edge, AGPL-3.0-or-later): its HTTP
// server, flow engine and Vue 3 editor live under internal/ and frontend/.
// What this program adds or changes for Orbit OS:
//
//   - main() connects to Orbit OS through the SDK and, once the app's own
//     server is listening, registers it with the Launcher in redirect mode
//     (see registerWithAppHub): the Launcher tile sends the browser straight
//     to <device-ip>:<port>/ instead of serving the page under a path of the
//     Launcher, which broke the editor's WebSocket log and debug stream and
//     the code editor's worker loading.
//   - The page listens on every interface, on a port in the 50000-60000
//     range reserved for app pages, because the browser connects to it
//     directly. The app has its own login.
//   - The Modbus RTU node (internal/nodes/modbus/server.go) does not open
//     serial devices itself: an Orbit OS app reaches the serial ports through
//     the UART API, so RTU runs over client.UartPort through the adapter in
//     internal/nodes/modbus/orbit_rtu.go. Modbus TCP, MQTT, OPC UA and S7 are
//     plain network I/O and needed no changes.
//   - filesystem.SetWorkdir keeps the File Read/Out/Watch and CSV Out nodes
//     inside the app's data folder, the only place an Orbit OS app writes to.
//     Upstream leaves that limit optional.
//   - Nodes for the device's own hardware and services (GPIO, I2C, SPI,
//     system metrics, device identity, mobile notifications) under
//     internal/nodes/{gpio,i2c,spi,orbit}.
//   - The embedded NATS server opens no network port by default.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "embed"

	apphubv26 "github.com/OrbitOS-org/orbit-os-sdk-go/v26/api/app_hub_service/v26"
	"github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
	"github.com/OrbitOS-org/orbit-os-sdk-go/v26/logger"
	"github.com/OrbitOS-org/orbit-os-sdk-go/v26/metadata"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/config"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/logbuffer"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes/filesystem"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/server"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/ws"
)

const logTag = "main"

// appHubRoute identifies LOOPZE's tile in the OrbitOS launcher. With
// ExposureMode_EXPOSURE_MODE_REDIRECT (see registerWithAppHub below) it is
// not a proxied path prefix — the launcher sends the browser straight to
// <device-ip>:<port>/ instead of proxying through /loopze/... on the
// portal's own origin. Some things (the flow editor's WebSocket log/debug
// stream, Monaco's dynamic worker loading) didn't tolerate the proxied path
// prefix well, so this app opts out of that in favour of a direct link.
const appHubRoute = "/loopze"

//go:embed metadata.json
var metadataJSON []byte

var appManifest = metadata.MustParseAppManifestJSON(metadataJSON)

const banner = `
██╗      ██████╗  ██████╗ ██████╗ ███████╗███████╗
██║     ██╔═══██╗██╔═══██╗██╔══██╗╚══███╔╝██╔════╝
██║     ██║   ██║██║   ██║██████╔╝  ███╔╝ █████╗
██║     ██║   ██║██║   ██║██╔═══╝  ███╔╝  ██╔══╝
███████╗╚██████╔╝╚██████╔╝██║     ███████╗███████╗
╚══════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚══════╝╚══════╝
   Industrial Flow Automation — %s (%s) — on OrbitOS

`

func main() {
	// Device IP, used only when the app runs on a development machine against
	// a device on the network. On the device, NewClientAuto connects locally
	// and ignores it. Named "--host" like in every other Orbit OS app; the
	// address the page listens on is the separate "--bind-host" flag
	// (internal/config), so the two cannot be mistaken for one another.
	deviceHost := flag.String("host", "192.168.1.100", "Orbit OS device IP address (development machine only)")

	// config.Load() defines LOOPZE's own flags (--bind-host, --port,
	// --data-dir, ...) and calls flag.Parse() itself, so the device --host
	// flag must be registered above, before this call.
	cfg := config.Load()

	if cfg.ShowVersion {
		fmt.Fprintf(os.Stdout, "orbit-flow %s (commit %s, built %s)\n",
			config.Version, config.Commit, config.BuildTime)
		return
	}

	fmt.Fprintf(os.Stdout, banner, config.Version, config.Commit)

	// LOOPZE's own structured logger: stdout + an in-memory ring buffer
	// that feeds the editor's Terminal Log panel over WebSocket. Left as
	// upstream designed it — this is independent of the OrbitOS logger
	// below, which only mirrors a few OrbitOS-specific milestones to the
	// device-wide log daemon.
	logLevel := cfg.ParseLogLevel()
	logBuf := logbuffer.New(cfg.LogBufferSize)
	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logHandler := logbuffer.NewHandler(textHandler, logBuf)
	slog.SetDefault(slog.New(logHandler))

	slog.Info("starting LOOPZE",
		"version", config.Version,
		"commit", config.Commit,
		"log_level", cfg.LogLevel,
	)
	slog.Info("configuration loaded",
		"host", cfg.Host,
		"port", cfg.Port,
		"data_dir", cfg.DataDir,
	)

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		slog.Error("failed to create data directory", "path", cfg.DataDir, "error", err)
		os.Exit(1)
	}

	// OrbitOS apps may only touch their own workdir — install it as the
	// mandatory jail for the File/CSV nodes' filesystem access (see
	// internal/nodes/filesystem's effectiveJail) before the flow engine can
	// deploy any flow that uses them. This overrides whatever rootJail a
	// flow configures if it points outside cfg.DataDir; an empty rootJail
	// no longer means "no restriction" the way it does in upstream
	// loopze-edge run outside OrbitOS.
	absDataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		slog.Error("failed to resolve data directory to an absolute path", "path", cfg.DataDir, "error", err)
		os.Exit(1)
	}
	filesystem.SetWorkdir(absDataDir)

	// OrbitOS logger mirrors a handful of milestones to logd; stdout stays
	// owned by the slog handler above (enableStdout=false avoids duplicate
	// lines in two different formats).
	meta := metadata.Build(appManifest)
	logger.Init(meta.Name, "INFO", false)
	appManifest.PrintInfo()

	// Connect to Orbit OS (locally on the device, over the network to --host
	// from a development machine).
	orbit, err := client.NewClientAuto(*deviceHost)
	if err != nil {
		slog.Error("failed to connect to Orbit OS", "error", err)
		os.Exit(1)
	}
	defer orbit.Close()

	hwModel, _ := orbit.SystemManager.GetHardwareModel()
	logger.Infof(logTag, "HW Model: %s", hwModel)
	slog.Info("connected to OrbitOS device", "hw_model", hwModel)

	// Make the SDK client reachable from node packages that use the device's
	// hardware through the API (for example Modbus RTU over UartManager — see
	// internal/nodes/modbus/orbit_rtu.go). Must happen before the flow
	// engine can deploy any flow that references a serial device.
	orbitsdk.SetClient(orbit)

	// Build and start LOOPZE's own server (flow engine, REST API, WS hub,
	// embedded frontend) exactly as upstream does.
	srv, err := server.New(cfg, logBuf)
	if err != nil {
		slog.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	logHandler.SetNotify(func(e logbuffer.LogEntry) {
		srv.Hub().Broadcast(ws.EventLog, e)
	})

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	// Give the listener a moment to bind before announcing it to AppHub —
	// RegisterWebUI's TCP health check would otherwise flap on the first
	// probe. srv.Start() logs "listening" itself once bound; select below
	// also catches an immediate bind failure instead of registering a dead
	// address.
	select {
	case err := <-serverErr:
		slog.Error("server failed", "error", err)
		os.Exit(1)
	case <-time.After(300 * time.Millisecond):
	}

	if err := registerWithAppHub(orbit, cfg.Port); err != nil {
		slog.Warn("AppHub WebUI registration failed — LOOPZE stays reachable directly on its bind address", "error", err)
	}

	// Wait for interrupt signal for graceful shutdown, or an async server
	// failure after startup.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}

	if err := orbit.AppHubManager.UnregisterService(); err != nil {
		slog.Warn("AppHub unregister failed", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("LOOPZE stopped gracefully")
}

// registerWithAppHub announces LOOPZE to the OrbitOS launcher with
// ExposureMode_EXPOSURE_MODE_REDIRECT: clicking the tile sends the browser
// straight to http://<lan-ip>:<port>/ instead of proxying through the
// portal's own origin at /loopze/... (ExposureMode_EXPOSURE_MODE_PROXY, the
// default RegisterWebUI uses). The redirect target must be an address the
// browser can actually dial, so — unlike the proxy case, where 127.0.0.1 is
// fine because the daemon dials it locally — this needs the device's real
// LAN-facing IP, not cfg.Host (which is 0.0.0.0, not a dialable address).
func registerWithAppHub(orbit *client.Client, port int) error {
	ip, err := detectLANIP()
	if err != nil {
		return fmt.Errorf("detect LAN IP: %w", err)
	}

	err = orbit.AppHubManager.RegisterService(&apphubv26.RegisterServiceRequest{
		Host:         ip,
		Port:         int32(port),
		Routes:       []*apphubv26.Route{{Path: appHubRoute}},
		Health:       &apphubv26.HealthCheck{Type: apphubv26.HealthCheckType_HEALTH_CHECK_TCP},
		ExposureMode: apphubv26.ExposureMode_EXPOSURE_MODE_REDIRECT,
	})
	if err != nil {
		return err
	}
	slog.Info("registered WebUI with OrbitOS AppHub (direct redirect)", "route", appHubRoute, "addr", fmt.Sprintf("%s:%d", ip, port))
	return nil
}

// detectLANIP returns this device's outbound-facing IP without sending any
// packet (a UDP "connect" only resolves the local routing table) — the same
// trick used by other Orbit apps (e.g. app-mcp-server) to find an address
// reachable from other machines on the LAN, as opposed to 127.0.0.1 or
// 0.0.0.0 which aren't valid redirect targets for a remote browser.
func detectLANIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}
