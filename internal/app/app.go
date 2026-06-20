package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/y08lin4/proxy-cat/internal/configgen"
	"github.com/y08lin4/proxy-cat/internal/core"
	"github.com/y08lin4/proxy-cat/internal/profile"
)

var ErrNoActiveProfile = errors.New("no active profile is loaded")

type App struct {
	mu         sync.RWMutex
	ctx        context.Context
	httpClient *http.Client
	launcher   *core.MihomoLauncher
	system     *core.WindowsSystemProxy
	autoStable AutoStableRunner

	status           AppStatus
	autoStableStatus AutoStableStatus
	autoStableTickAt time.Time
	logs             []LogLine
	active           profile.Profile
	activeConfig     string
	dataDir          string
	mihomoHome       string
	mihomoBinary     string
}

func New() *App {
	dataDir := defaultDataDir()
	return &App{
		httpClient:   http.DefaultClient,
		launcher:     core.NewMihomoLauncher(),
		system:       core.NewWindowsSystemProxy(),
		dataDir:      dataDir,
		mihomoHome:   filepath.Join(dataDir, "mihomo"),
		mihomoBinary: "mihomo.exe",
		status: AppStatus{
			CoreRunning:        false,
			SystemProxyEnabled: false,
			AutoStableEnabled:  false,
			ControllerAddress:  "127.0.0.1:9090",
		},
		autoStableStatus: AutoStableStatus{
			Available: false,
			Enabled:   false,
			Running:   false,
		},
	}
}

func (a *App) GetAppStatus() AppStatus {
	if a.launcher != nil {
		recovered, err := a.launcher.RecoverIfNeeded(a.context())
		a.mu.Lock()
		if err != nil {
			a.status.LastError = fmt.Sprintf("recover mihomo core: %v", err)
			a.appendLogLocked("error", a.status.LastError)
		} else if recovered {
			a.status.CoreRunning = true
			a.status.LastError = ""
			a.appendLogLocked("info", "Mihomo core recovered")
		}
		a.mu.Unlock()
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.refreshCoreStatusLocked()
	return a.status
}

func (a *App) StartCore() error {
	a.mu.RLock()
	configPath := a.activeConfig
	binaryPath := a.mihomoBinary
	homeDir := a.mihomoHome
	a.mu.RUnlock()

	if configPath == "" {
		return a.fail(ErrNoActiveProfile)
	}
	ctx := a.context()
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		return a.fail(fmt.Errorf("create mihomo home: %w", err))
	}
	if err := a.launcher.Start(ctx, core.MihomoLaunchConfig{
		BinaryPath: binaryPath,
		ConfigPath: configPath,
		HomeDir:    homeDir,
	}); err != nil {
		return a.fail(err)
	}

	a.mu.Lock()
	a.status.CoreRunning = true
	a.status.LastError = ""
	a.appendLogLocked("info", "Mihomo core started")
	a.mu.Unlock()
	return nil
}

func (a *App) StopCore() error {
	if err := a.launcher.Stop(a.context()); err != nil {
		return a.fail(err)
	}
	a.mu.Lock()
	a.status.CoreRunning = false
	a.appendLogLocked("info", "Mihomo core stopped")
	a.mu.Unlock()
	return nil
}

func (a *App) RestartCore() error {
	if err := a.StopCore(); err != nil {
		return err
	}
	return a.StartCore()
}

func (a *App) RecoverCoreIfNeeded() (bool, error) {
	if a.launcher == nil {
		return false, nil
	}
	recovered, err := a.launcher.RecoverIfNeeded(a.context())
	if err != nil {
		return false, a.fail(fmt.Errorf("recover mihomo core: %w", err))
	}
	if !recovered {
		a.mu.Lock()
		a.refreshCoreStatusLocked()
		a.mu.Unlock()
		return false, nil
	}

	a.mu.Lock()
	a.status.CoreRunning = true
	a.status.LastError = ""
	a.appendLogLocked("info", "Mihomo core recovered")
	a.mu.Unlock()
	return true, nil
}

func (a *App) SetSystemProxy(enabled bool) error {
	a.mu.RLock()
	mixedPort := a.active.Settings.MixedPort
	a.mu.RUnlock()
	if mixedPort <= 0 {
		mixedPort = profile.DefaultSettings().MixedPort
	}

	var err error
	if enabled {
		server := fmt.Sprintf("127.0.0.1:%d", mixedPort)
		err = a.system.Enable(a.context(), server, "localhost;127.*;<local>")
	} else {
		err = a.system.Disable(a.context())
	}
	if err != nil {
		return a.fail(err)
	}

	a.mu.Lock()
	a.status.SystemProxyEnabled = enabled
	a.status.LastError = ""
	if enabled {
		a.appendLogLocked("info", "Windows system proxy enabled")
	} else {
		a.appendLogLocked("info", "Windows system proxy disabled")
	}
	a.mu.Unlock()
	return nil
}

func (a *App) LoadSubscription(subscriptionURL string) error {
	if subscriptionURL == "" {
		return errors.New("subscription url is required")
	}

	req, err := http.NewRequestWithContext(a.context(), http.MethodGet, subscriptionURL, nil)
	if err != nil {
		return a.fail(err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return a.fail(fmt.Errorf("load subscription: %w", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return a.fail(fmt.Errorf("load subscription: status %d", resp.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return a.fail(err)
	}
	return a.LoadSubscriptionData(subscriptionURL, body)
}

func (a *App) LoadSubscriptionData(subscriptionURL string, data []byte) error {
	p, err := profile.ParseSubscription(data, profile.ParseOptions{
		ProfileName:      "Default",
		SubscriptionName: "Default",
		SubscriptionURL:  subscriptionURL,
	})
	if err != nil {
		return a.fail(err)
	}
	yamlData, err := configgen.GenerateMihomoYAML(p, configgen.Options{})
	if err != nil {
		return a.fail(err)
	}
	configPath, err := a.writeActiveConfig(yamlData)
	if err != nil {
		return a.fail(err)
	}
	autoStableRunner, err := newMihomoAutoStableRunner(p, a.httpClient)
	if err != nil {
		return a.fail(err)
	}

	a.mu.Lock()
	a.active = p
	a.activeConfig = configPath
	a.autoStable = autoStableRunner
	a.autoStableStatus = AutoStableStatus{
		Available: true,
		Enabled:   a.status.AutoStableEnabled,
		Running:   a.status.AutoStableEnabled,
		Health:    autoStableRunner.health(time.Now()),
	}
	a.status.ActiveProfileName = p.Name
	a.status.ControllerAddress = p.Settings.ExternalController
	a.status.LastError = ""
	a.appendLogLocked("info", "Subscription loaded and Mihomo config generated")
	a.mu.Unlock()
	return nil
}

func (a *App) GetProxyGroups() []ProxyGroupView {
	a.mu.RLock()
	defer a.mu.RUnlock()
	nodeByName := make(map[string]profile.ProxyNode, len(a.active.Proxies))
	for _, node := range a.active.Proxies {
		nodeByName[node.Name] = node
	}
	views := make([]ProxyGroupView, 0, len(a.active.ProxyGroups))
	for _, group := range a.active.ProxyGroups {
		view := ProxyGroupView{
			Name:     group.Name,
			Type:     group.Type,
			Selected: group.SelectedProxy,
			Proxies:  make([]ProxyView, 0, len(group.Proxies)),
		}
		for _, name := range group.Proxies {
			node := nodeByName[name]
			proxyType := node.Type
			alive := node.Name != "" || name == "DIRECT" || name == "AUTO"
			view.Proxies = append(view.Proxies, ProxyView{
				Name:  name,
				Type:  proxyType,
				Alive: alive,
			})
		}
		views = append(views, view)
	}
	return views
}

func (a *App) SelectProxy(groupName string, proxyName string) error {
	if groupName == "" {
		return errors.New("proxy group name is required")
	}
	if proxyName == "" {
		return errors.New("proxy name is required")
	}
	a.mu.RLock()
	controller := a.status.ControllerAddress
	secret := a.active.Settings.Secret
	running := a.status.CoreRunning
	a.mu.RUnlock()

	if running {
		client, err := core.NewMihomoClient(controller, secret, a.httpClient)
		if err != nil {
			return a.fail(err)
		}
		if err := client.SelectProxy(a.context(), groupName, proxyName); err != nil {
			return a.fail(err)
		}
	}

	a.mu.Lock()
	for i := range a.active.ProxyGroups {
		if a.active.ProxyGroups[i].Name == groupName {
			a.active.ProxyGroups[i].SelectedProxy = proxyName
			break
		}
	}
	a.status.LastError = ""
	a.appendLogLocked("info", fmt.Sprintf("Proxy group %s selected %s", groupName, proxyName))
	a.mu.Unlock()
	return nil
}

func (a *App) GetConnectionStatus() (ConnectionStatus, error) {
	a.mu.Lock()
	a.refreshCoreStatusLocked()
	controller := a.status.ControllerAddress
	secret := a.active.Settings.Secret
	running := a.status.CoreRunning
	httpClient := a.httpClient
	a.mu.Unlock()

	status := ConnectionStatus{CoreRunning: running}
	if !running {
		return status, nil
	}

	client, err := core.NewMihomoClient(controller, secret, httpClient)
	if err != nil {
		return status, a.fail(err)
	}
	connections, err := client.GetConnections(a.context())
	if err != nil {
		return status, a.fail(err)
	}

	status.UploadTotal = connections.UploadTotal
	status.DownloadTotal = connections.DownloadTotal
	status.ConnectionCount = len(connections.Connections)
	return status, nil
}

func (a *App) GetLogs(limit int) []LogLine {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.logs) == 0 {
		return []LogLine{}
	}
	if limit <= 0 || limit >= len(a.logs) {
		return append([]LogLine(nil), a.logs...)
	}
	return append([]LogLine(nil), a.logs[len(a.logs)-limit:]...)
}

func (a *App) context() context.Context {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) writeActiveConfig(data []byte) (string, error) {
	configDir := filepath.Join(a.dataDir, "profiles", "active")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, bytes.TrimSpace(data), 0o600); err != nil {
		return "", err
	}
	return configPath, nil
}

func (a *App) refreshCoreStatusLocked() {
	if a.launcher == nil {
		return
	}
	status := a.launcher.Status()
	a.status.CoreRunning = status.Running
	if status.LastExit.Exited && !status.LastExit.Expected && !status.Running {
		a.status.LastError = status.LastExit.Error
		if a.status.LastError == "" {
			a.status.LastError = fmt.Sprintf("mihomo exited with code %d", status.LastExit.ExitCode)
		}
	}
}

func (a *App) fail(err error) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status.LastError = err.Error()
	a.appendLogLocked("error", err.Error())
	return err
}

func (a *App) appendLogLocked(level string, message string) {
	a.logs = append(a.logs, LogLine{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	})
	if len(a.logs) > 500 {
		a.logs = append([]LogLine(nil), a.logs[len(a.logs)-500:]...)
	}
}

func defaultDataDir() string {
	root, err := os.UserConfigDir()
	if err != nil || root == "" {
		root = "."
	}
	return filepath.Join(root, "Proxy-Cat")
}
