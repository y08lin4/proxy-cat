package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/proxy-cat/internal/autostable"
	"github.com/y08lin4/proxy-cat/internal/domain/configgen"
	"github.com/y08lin4/proxy-cat/internal/domain/persistence"
	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
	"github.com/y08lin4/proxy-cat/internal/domain/ruleengine"
	"github.com/y08lin4/proxy-cat/internal/domain/subscription"
	"github.com/y08lin4/proxy-cat/internal/platform/mihomo"
	"github.com/y08lin4/proxy-cat/internal/platform/system"
)

// ErrNoActiveProfile indicates that no subscription has been loaded yet.
var ErrNoActiveProfile = errors.New("no active profile is loaded")

// ErrAutoStableUnavailable indicates the auto-stable manager has not been initialised.
var ErrAutoStableUnavailable = errors.New("auto-stable service is not available")

const autoStableTickCooldown = 5 * time.Second

const autoStableDelayURL = "https://www.gstatic.com/generate_204"

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// Config holds the startup configuration for the Service.
type Config struct {
	DataDir       string
	MihomoBinary  string
	MihomoHome    string
	Headless      bool
	NoSystemProxy bool
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	root, err := os.UserConfigDir()
	if err != nil || root == "" {
		root = "."
	}
	return Config{
		DataDir:      filepath.Join(root, "Proxy-Cat"),
		MihomoBinary: "mihomo.exe",
		MihomoHome:   filepath.Join(root, "Proxy-Cat", "mihomo"),
	}
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service is the headless service layer that manages the proxy core, system
// proxy, subscription lifecycle, and auto-stable selection.  It is the
// single-writer owner of all mutable domain and UI state.
type Service struct {
	mu sync.RWMutex

	// Platform dependencies
	launcher    *mihomo.Launcher
	systemProxy system.SystemProxy
	httpClient  *http.Client

	// Domain state
	active        proxy.Profile
	activeConfig  string
	autoStable    *autostable.Manager
	autoStableCfg autostable.Config
	store         *persistence.Store
	ruleEngine    *ruleengine.Engine

	// UI state
	status           AppStatus
	autoStableStatus AutoStableStatus
	logs             []LogLine

	// Config
	cfg Config
}

// New creates a Service with the supplied Config.  A zero-value Config causes
// sensible defaults to be used.
func New(cfg Config) *Service {
	if cfg.DataDir == "" {
		cfg = DefaultConfig()
	}
	if cfg.MihomoHome == "" {
		cfg.MihomoHome = filepath.Join(cfg.DataDir, "mihomo")
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}

	var sysProxy system.SystemProxy
	if !cfg.NoSystemProxy {
		sysProxy = system.NewDefault()
	}

	return &Service{
		httpClient:  httpClient,
		launcher:    mihomo.NewLauncher(),
		systemProxy: sysProxy,
		autoStableCfg: autostable.Config{
			SampleLimit:          10,
			MinHoldTime:          1 * time.Minute,
			SwitchThreshold:      100,
			CooldownAfterFailure: 1 * time.Minute,
			ConsecutiveFailLimit: 2,
		},
		store:       persistence.NewStore(cfg.DataDir),
		ruleEngine:  ruleengine.New(),
		cfg:         cfg,
	}
}

// Shutdown gracefully stops all running services.
func (s *Service) Shutdown(ctx context.Context) error {
	var errs []error
	if err := s.launcher.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("stop mihomo: %w", err))
	}
	if s.systemProxy != nil {
		if err := s.systemProxy.Disable(ctx); err != nil {
			errs = append(errs, fmt.Errorf("disable system proxy: %w", err))
		}
	}
	return errors.Join(errs...)
}

// ---------------------------------------------------------------------------
// Context helper
// ---------------------------------------------------------------------------

// context returns a context.Background() — the service layer does not carry
// a Wails lifecycle context.
func (s *Service) context() context.Context {
	return context.Background()
}

// ---------------------------------------------------------------------------
// App status
// ---------------------------------------------------------------------------

// GetAppStatus returns the current application-level status snapshot.
func (s *Service) GetAppStatus() AppStatus {
	if s.launcher != nil {
		recovered, err := s.launcher.RecoverIfNeeded(s.context())
		s.mu.Lock()
		if err != nil {
			s.status.LastError = fmt.Sprintf("recover mihomo core: %v", err)
			s.appendLogLocked("error", s.status.LastError)
		} else if recovered {
			s.status.CoreRunning = true
			s.status.LastError = ""
			s.appendLogLocked("info", "Mihomo core recovered")
		}
		s.mu.Unlock()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshCoreStatusLocked()
	return s.status
}

// ---------------------------------------------------------------------------
// Core lifecycle
// ---------------------------------------------------------------------------

// StartCore launches the Mihomo core process with the active configuration.
func (s *Service) StartCore() error {
	s.mu.RLock()
	configPath := s.activeConfig
	binaryPath := s.cfg.MihomoBinary
	homeDir := s.cfg.MihomoHome
	s.mu.RUnlock()

	if configPath == "" {
		return s.fail(ErrNoActiveProfile)
	}
	ctx := s.context()
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		return s.fail(fmt.Errorf("create mihomo home: %w", err))
	}
	if err := s.launcher.Start(ctx, mihomo.LaunchConfig{
		BinaryPath: binaryPath,
		ConfigPath: configPath,
		HomeDir:    homeDir,
	}); err != nil {
		return s.fail(err)
	}

	s.mu.Lock()
	s.status.CoreRunning = true
	s.status.LastError = ""
	s.appendLogLocked("info", "Mihomo core started")
	s.mu.Unlock()
	return nil
}

// StopCore stops the Mihomo core process.
func (s *Service) StopCore() error {
	if err := s.launcher.Stop(s.context()); err != nil {
		return s.fail(err)
	}
	s.mu.Lock()
	s.status.CoreRunning = false
	s.appendLogLocked("info", "Mihomo core stopped")
	s.mu.Unlock()
	return nil
}

// RestartCore stops and then restarts the Mihomo core process.
func (s *Service) RestartCore() error {
	if err := s.StopCore(); err != nil {
		return err
	}
	return s.StartCore()
}

// RecoverCoreIfNeeded checks whether the core process exited unexpectedly and
// restarts it if so.  It returns whether a recovery was performed.
func (s *Service) RecoverCoreIfNeeded() (bool, error) {
	if s.launcher == nil {
		return false, nil
	}
	recovered, err := s.launcher.RecoverIfNeeded(s.context())
	if err != nil {
		return false, s.fail(fmt.Errorf("recover mihomo core: %w", err))
	}
	if !recovered {
		s.mu.Lock()
		s.refreshCoreStatusLocked()
		s.mu.Unlock()
		return false, nil
	}

	s.mu.Lock()
	s.status.CoreRunning = true
	s.status.LastError = ""
	s.appendLogLocked("info", "Mihomo core recovered")
	s.mu.Unlock()
	return true, nil
}

// ---------------------------------------------------------------------------
// System proxy
// ---------------------------------------------------------------------------

// SetSystemProxy enables or disables the OS-level system proxy.  When
// headless mode is active (systemProxy is nil) the call is a no-op that
// returns a descriptive error.
func (s *Service) SetSystemProxy(enabled bool) error {
	if s.systemProxy == nil {
		return s.fail(errors.New("system proxy is not available (headless mode)"))
	}

	s.mu.RLock()
	mixedPort := s.active.Settings.MixedPort
	s.mu.RUnlock()
	if mixedPort <= 0 {
		mixedPort = proxy.DefaultSettings().MixedPort
	}

	var err error
	if enabled {
		server := fmt.Sprintf("127.0.0.1:%d", mixedPort)
		err = s.systemProxy.Enable(s.context(), server, "localhost;127.*;<local>")
	} else {
		err = s.systemProxy.Disable(s.context())
	}
	if err != nil {
		return s.fail(err)
	}

	s.mu.Lock()
	s.status.SystemProxyEnabled = enabled
	s.status.LastError = ""
	if enabled {
		s.appendLogLocked("info", "System proxy enabled")
	} else {
		s.appendLogLocked("info", "System proxy disabled")
	}
	s.mu.Unlock()
	return nil
}

// ---------------------------------------------------------------------------
// Subscription
// ---------------------------------------------------------------------------

// validateSubscriptionURL performs security checks on a subscription URL to
// prevent SSRF attacks.  It rejects file://, ftp://, and other non-HTTP schemes,
// loopback addresses, private network addresses, link-local addresses, and the
// unspecified address.
func validateSubscriptionURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid subscription URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("subscription URL has no host")
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsUnspecified() {
			return fmt.Errorf("subscription URL host %q is not allowed (unspecified address)", host)
		}
		if ip.IsLoopback() {
			return fmt.Errorf("subscription URL host %q is not allowed (loopback address)", host)
		}
		if ip.IsPrivate() {
			return fmt.Errorf("subscription URL host %q is not allowed (private network)", host)
		}
		if ip.IsLinkLocalUnicast() {
			return fmt.Errorf("subscription URL host %q is not allowed (link-local address)", host)
		}
		return nil
	}
	// Host is not a raw IP — check textual loopback/unspecified.
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "0.0.0.0" {
		return fmt.Errorf("subscription URL host %q is not allowed (loopback or unspecified)", host)
	}
	return nil
}

// LoadSubscription fetches the subscription from the given URL, parses it,
// writes the generated Mihomo configuration, and initialises the auto-stable
// manager.
func (s *Service) LoadSubscription(subscriptionURL string) error {
	if subscriptionURL == "" {
		return errors.New("subscription url is required")
	}
	if err := validateSubscriptionURL(subscriptionURL); err != nil {
		return s.fail(err)
	}

	req, err := http.NewRequestWithContext(s.context(), http.MethodGet, subscriptionURL, nil)
	if err != nil {
		return s.fail(err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return s.fail(fmt.Errorf("load subscription: %w", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.fail(fmt.Errorf("load subscription: status %d", resp.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return s.fail(err)
	}
	return s.LoadSubscriptionData(subscriptionURL, body)
}

// LoadSubscriptionData parses raw subscription data, writes the generated
// Mihomo configuration, and initialises the auto-stable manager.
func (s *Service) LoadSubscriptionData(subscriptionURL string, data []byte) error {
	p, err := subscription.ParseSubscription(data, subscription.ParseOptions{
		ProfileName:      "Default",
		SubscriptionName: "Default",
		SubscriptionURL:  subscriptionURL,
	})
	if err != nil {
		return s.fail(err)
	}

	yamlData, err := configgen.GenerateMihomoYAML(p, configgen.Options{})
	if err != nil {
		return s.fail(err)
	}

	configPath, err := s.writeActiveConfig(yamlData)
	if err != nil {
		return s.fail(err)
	}

	manager, err := s.initAutoStableManager(p)
	if err != nil {
		return s.fail(err)
	}

	s.mu.Lock()
	s.active = p
	s.activeConfig = configPath
	s.autoStable = manager
	s.autoStableStatus = AutoStableStatus{
		Available: true,
		Enabled:   s.status.AutoStableEnabled,
		Running:   s.status.AutoStableEnabled,
		Health:    s.buildAutoStableHealthLocked(),
	}
	s.status.ActiveProfileName = p.Name
	s.status.ControllerAddress = p.Settings.ExternalController
	s.status.LastError = ""
	s.appendLogLocked("info", "Subscription loaded and Mihomo config generated")
	s.mu.Unlock()
	return nil
}

// initAutoStableManager creates a new autostable.Manager, registers every
// proxy node, and seeds the current selection from the active profile's
// auto-stable groups.
func (s *Service) initAutoStableManager(p proxy.Profile) (*autostable.Manager, error) {
	manager, err := autostable.NewManager(s.autoStableCfg)
	if err != nil {
		return nil, err
	}
	for _, node := range p.Proxies {
		if err := manager.Register(node.Name); err != nil {
			return nil, err
		}
	}
	// Seed the current selection from the first auto-stable group that
	// already has a selected proxy.
	for _, group := range p.ProxyGroups {
		if group.Type != "auto-stable" {
			continue
		}
		if group.SelectedProxy != "" {
			_ = manager.SetCurrent(group.SelectedProxy, time.Now())
			break
		}
	}
	return manager, nil
}

// ---------------------------------------------------------------------------
// Proxy groups
// ---------------------------------------------------------------------------

// GetProxyGroups returns a frontend-ready snapshot of all proxy groups.
func (s *Service) GetProxyGroups() []ProxyGroupView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodeByName := make(map[string]proxy.ProxyNode, len(s.active.Proxies))
	for _, node := range s.active.Proxies {
		nodeByName[node.Name] = node
	}
	views := make([]ProxyGroupView, 0, len(s.active.ProxyGroups))
	for _, group := range s.active.ProxyGroups {
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

// SelectProxy tells the Mihomo core to switch a proxy group to the given
// proxy.  The in-memory state is only updated after a successful API call
// (fixing a bug in the old App layer where state was mutated even on
// failure).
func (s *Service) SelectProxy(groupName string, proxyName string) error {
	if groupName == "" {
		return errors.New("proxy group name is required")
	}
	if proxyName == "" {
		return errors.New("proxy name is required")
	}

	s.mu.RLock()
	controller := s.status.ControllerAddress
	secret := s.active.Settings.Secret
	running := s.status.CoreRunning
	s.mu.RUnlock()

	if running {
		client, err := mihomo.NewClient(controller, secret, s.httpClient)
		if err != nil {
			return s.fail(err)
		}
		if err := client.SelectProxy(s.context(), groupName, proxyName); err != nil {
			return s.fail(err)
		}
	}

	// Only mutate in-memory state after the API call succeeded (or was
	// skipped because the core is not running).
	s.mu.Lock()
	for i := range s.active.ProxyGroups {
		if s.active.ProxyGroups[i].Name == groupName {
			s.active.ProxyGroups[i].SelectedProxy = proxyName
			break
		}
	}
	s.status.LastError = ""
	s.appendLogLocked("info", fmt.Sprintf("Proxy group %s selected %s", groupName, proxyName))
	s.mu.Unlock()
	return nil
}

// ---------------------------------------------------------------------------
// Connection status
// ---------------------------------------------------------------------------

// GetConnectionStatus returns a snapshot of current connection statistics from
// the Mihomo core.
func (s *Service) GetConnectionStatus() (ConnectionStatus, error) {
	s.mu.Lock()
	s.refreshCoreStatusLocked()
	controller := s.status.ControllerAddress
	secret := s.active.Settings.Secret
	running := s.status.CoreRunning
	httpClient := s.httpClient
	s.mu.Unlock()

	status := ConnectionStatus{CoreRunning: running}
	if !running {
		return status, nil
	}

	client, err := mihomo.NewClient(controller, secret, httpClient)
	if err != nil {
		return status, s.fail(err)
	}
	connections, err := client.GetConnections(s.context())
	if err != nil {
		return status, s.fail(err)
	}

	status.UploadTotal = connections.UploadTotal
	status.DownloadTotal = connections.DownloadTotal
	status.ConnectionCount = len(connections.Connections)
	return status, nil
}

// ---------------------------------------------------------------------------
// Logs
// ---------------------------------------------------------------------------

// GetLogs returns the most recent application log lines.  If limit is zero or
// exceeds the number of stored lines, all lines are returned.
func (s *Service) GetLogs(limit int) []LogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit >= len(s.logs) {
		return append([]LogLine(nil), s.logs...)
	}
	return append([]LogLine(nil), s.logs[len(s.logs)-limit:]...)
}

// ---------------------------------------------------------------------------
// Auto-stable status
// ---------------------------------------------------------------------------

// GetAutoStableStatus returns the current Phase 2 health/control snapshot.
func (s *Service) GetAutoStableStatus() AutoStableStatus {
	s.mu.RLock()
	manager := s.autoStable
	s.mu.RUnlock()

	if manager != nil {
		s.mu.Lock()
		s.autoStableStatus.Health = s.buildAutoStableHealthLocked()
		status := s.autoStableSnapshotLocked()
		s.mu.Unlock()
		return status
	}

	return s.autoStableSnapshot()
}

// SetAutoStableEnabled toggles manual/interval auto-stable behaviour.
func (s *Service) SetAutoStableEnabled(enabled bool) error {
	s.mu.Lock()
	s.status.AutoStableEnabled = enabled
	s.autoStableStatus.Enabled = enabled
	s.autoStableStatus.Available = s.autoStable != nil
	s.autoStableStatus.LastError = ""
	if enabled {
		s.appendLogLocked("info", "Auto-stable enabled")
	} else {
		s.appendLogLocked("info", "Auto-stable disabled")
	}
	s.mu.Unlock()
	return nil
}

// RunAutoStableTick runs one controllable auto-stable tick against the
// "AUTO-STABLE" group.  A cooldown window is enforced to avoid spamming the
// Mihomo API.  Only autoStableStatus.LastTickAt is consulted (fixing the
// dual-tick-at bug present in the App layer).
func (s *Service) RunAutoStableTick() (AutoStableActionResult, error) {
	s.mu.RLock()
	manager := s.autoStable
	lastTickAt := s.autoStableStatus.LastTickAt
	s.mu.RUnlock()

	if manager == nil {
		result := s.unavailableAutoStableResult("tick", "")
		s.mu.Lock()
		s.autoStableStatus.LastTickAt = result.CompletedAt
		s.autoStableStatus.LastAction = result.Action
		s.autoStableStatus.LastError = result.Message
		s.appendLogLocked("warn", result.Message)
		s.mu.Unlock()
		return result, ErrAutoStableUnavailable
	}

	if time.Since(lastTickAt) < autoStableTickCooldown {
		result := AutoStableActionResult{
			Action:      "tick",
			Changed:     false,
			Message:     "Auto-stable tick skipped by cooldown",
			CompletedAt: time.Now(),
		}
		s.mu.Lock()
		result.Health = s.buildAutoStableHealthLocked()
		s.autoStableStatus.LastAction = result.Action
		s.autoStableStatus.LastError = result.Message
		s.appendLogLocked("info", result.Message)
		s.mu.Unlock()
		return result, nil
	}

	result, err := s.autoStableTickLocked("AUTO-STABLE")
	if err != nil {
		return result, s.fail(fmt.Errorf("auto-stable tick: %w", err))
	}
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now()
	}
	s.mu.Lock()
	s.autoStableStatus.LastTickAt = result.CompletedAt
	s.mu.Unlock()
	s.recordAutoStableAction(result)
	return result, nil
}

// SelectAutoStableProxy asks the auto-stable engine to choose one proxy for a
// group now.
func (s *Service) SelectAutoStableProxy(groupName string) (AutoStableActionResult, error) {
	if groupName == "" {
		return AutoStableActionResult{}, errors.New("proxy group name is required")
	}

	s.mu.RLock()
	manager := s.autoStable
	s.mu.RUnlock()

	if manager == nil {
		result := s.unavailableAutoStableResult("select", groupName)
		s.mu.Lock()
		s.autoStableStatus.LastAction = result.Action
		s.autoStableStatus.LastError = result.Message
		s.appendLogLocked("warn", result.Message)
		s.mu.Unlock()
		return result, ErrAutoStableUnavailable
	}

	result, err := s.autoStableTickLocked(groupName)
	if err != nil {
		return result, s.fail(fmt.Errorf("auto-stable select: %w", err))
	}
	s.recordAutoStableAction(result)
	return result, nil
}

// ---------------------------------------------------------------------------
// Auto-stable runtime — performs the actual health checks and selection I/O.
// ---------------------------------------------------------------------------

// autoStableTickLocked runs one health-check + selection round for the named
// group.  The caller must already hold the write lock.
func (s *Service) autoStableTickLocked(groupName string) (AutoStableActionResult, error) {
	now := time.Now()
	controller := s.active.Settings.ExternalController
	secret := s.active.Settings.Secret

	client, err := mihomo.NewClient(controller, secret, s.httpClient)
	if err != nil {
		return AutoStableActionResult{}, err
	}

	nodes := s.nodesForGroupLocked(groupName)
	for _, node := range nodes {
		delay, err := client.TestProxyDelay(s.context(), node, autoStableDelayURL, 5000)
		if err != nil {
			_ = s.autoStable.Record(autostable.Sample{
				NodeID:    node,
				Success:   false,
				CheckedAt: now,
			})
			continue
		}
		_ = s.autoStable.Record(autostable.Sample{
			NodeID:    node,
			Latency:   time.Duration(delay.Delay) * time.Millisecond,
			Success:   true,
			CheckedAt: now,
		})
	}

	decision := s.autoStable.Select(now)
	result := AutoStableActionResult{
		Action:      "select",
		GroupName:   groupName,
		Selected:    decision.SelectedID,
		Changed:     decision.Switched,
		Message:     fmt.Sprintf("Auto-stable kept %s", decision.SelectedID),
		CompletedAt: now,
		Health:      s.buildAutoStableHealthLocked(),
	}
	if decision.Switched {
		result.Message = fmt.Sprintf("Auto-stable selected %s", decision.SelectedID)
		if decision.SelectedID != "" {
			if err := client.SelectProxy(s.context(), groupName, decision.SelectedID); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

// nodesForGroupLocked returns the proxy names that belong to the named proxy
// group.  Only proxies that exist in the active profile are included.
func (s *Service) nodesForGroupLocked(groupName string) []string {
	nodeSet := make(map[string]struct{}, len(s.active.Proxies))
	for _, node := range s.active.Proxies {
		nodeSet[node.Name] = struct{}{}
	}
	for _, group := range s.active.ProxyGroups {
		if group.Name != groupName {
			continue
		}
		var nodes []string
		for _, name := range group.Proxies {
			if _, ok := nodeSet[name]; ok {
				nodes = append(nodes, name)
			}
		}
		return nodes
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// fail records the error in the status and log, then returns it unchanged.
func (s *Service) fail(err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.LastError = err.Error()
	s.appendLogLocked("error", err.Error())
	return err
}

// appendLogLocked adds a log line and trims the buffer to a maximum of 500
// entries.  The caller must hold s.mu.
func (s *Service) appendLogLocked(level string, message string) {
	s.logs = append(s.logs, LogLine{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	})
	if len(s.logs) > 500 {
		s.logs = append([]LogLine(nil), s.logs[len(s.logs)-500:]...)
	}
}

// refreshCoreStatusLocked synchronises the in-memory core-running flag with
// the launcher's observed status.  The caller must hold s.mu.
func (s *Service) refreshCoreStatusLocked() {
	if s.launcher == nil {
		return
	}
	status := s.launcher.Status()
	s.status.CoreRunning = status.Running
	if status.LastExit.Exited && !status.LastExit.Expected && !status.Running {
		s.status.LastError = status.LastExit.Error
		if s.status.LastError == "" {
			s.status.LastError = fmt.Sprintf("mihomo exited with code %d", status.LastExit.ExitCode)
		}
	}
}

// writeActiveConfig writes the generated Mihomo YAML to the active profile
// directory and returns the path to the written file.
func (s *Service) writeActiveConfig(data []byte) (string, error) {
	configDir := filepath.Join(s.cfg.DataDir, "profiles", "active")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, bytes.TrimSpace(data), 0o600); err != nil {
		return "", err
	}
	return configPath, nil
}

// ---------------------------------------------------------------------------
// Auto-stable snapshot helpers
// ---------------------------------------------------------------------------

func (s *Service) autoStableSnapshot() AutoStableStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.autoStableSnapshotLocked()
}

func (s *Service) autoStableSnapshotLocked() AutoStableStatus {
	status := s.autoStableStatus
	status.Enabled = s.status.AutoStableEnabled
	if len(status.Health) == 0 {
		status.Health = s.buildAutoStableHealthLocked()
	}
	return status
}

func (s *Service) mergeAutoStableStatusLocked(status AutoStableStatus) {
	lastTickAt := s.autoStableStatus.LastTickAt
	lastAction := s.autoStableStatus.LastAction
	lastSelected := s.autoStableStatus.LastSelected
	lastError := s.autoStableStatus.LastError
	s.autoStableStatus = status
	if s.autoStableStatus.LastTickAt.IsZero() {
		s.autoStableStatus.LastTickAt = lastTickAt
	}
	if s.autoStableStatus.LastAction == "" {
		s.autoStableStatus.LastAction = lastAction
	}
	if s.autoStableStatus.LastSelected == "" {
		s.autoStableStatus.LastSelected = lastSelected
	}
	if s.autoStableStatus.LastError == "" {
		s.autoStableStatus.LastError = lastError
	}
	s.status.AutoStableEnabled = status.Enabled
}

func (s *Service) recordAutoStableAction(result AutoStableActionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now()
	}
	s.autoStableStatus.LastTickAt = result.CompletedAt
	s.autoStableStatus.LastAction = result.Action
	s.autoStableStatus.LastSelected = result.Selected
	s.autoStableStatus.LastError = ""
	if len(result.Health) > 0 {
		s.autoStableStatus.Health = append([]AutoStableGroupHealth(nil), result.Health...)
	}
	if result.GroupName != "" && result.Selected != "" {
		for i := range s.active.ProxyGroups {
			if s.active.ProxyGroups[i].Name == result.GroupName {
				s.active.ProxyGroups[i].SelectedProxy = result.Selected
				break
			}
		}
	}
	message := result.Message
	if message == "" {
		message = fmt.Sprintf("Auto-stable %s completed", result.Action)
	}
	s.appendLogLocked("info", message)
}

func (s *Service) unavailableAutoStableResult(action string, groupName string) AutoStableActionResult {
	message := "Auto-stable service is not available yet"
	return AutoStableActionResult{
		Action:      action,
		GroupName:   groupName,
		Changed:     false,
		Message:     message,
		CompletedAt: time.Now(),
		Health:      s.autoStableSnapshot().Health,
	}
}

// buildAutoStableHealthLocked builds a frontend-ready health snapshot for
// every auto-stable proxy group using the current manager state and profile
// data.  The caller must hold s.mu.
func (s *Service) buildAutoStableHealthLocked() []AutoStableGroupHealth {
	if s.autoStable == nil {
		return s.buildAutoStableHealthFallbackLocked()
	}

	now := time.Now()
	snapshots := make(map[string]autostable.NodeSnapshot)
	for _, snapshot := range s.autoStable.Snapshots(now) {
		snapshots[snapshot.NodeID] = snapshot
	}
	nodeByName := make(map[string]proxy.ProxyNode, len(s.active.Proxies))
	for _, node := range s.active.Proxies {
		nodeByName[node.Name] = node
	}

	var groups []AutoStableGroupHealth
	for _, group := range s.active.ProxyGroups {
		if group.Type != "auto-stable" {
			continue
		}
		view := AutoStableGroupHealth{
			Name:     group.Name,
			Type:     group.Type,
			Selected: group.SelectedProxy,
			Proxies:  make([]AutoStableNodeHealth, 0, len(group.Proxies)),
		}
		for _, name := range group.Proxies {
			node, isNode := nodeByName[name]
			if !isNode {
				continue
			}
			snapshot := snapshots[name]
			view.Proxies = append(view.Proxies, AutoStableNodeHealth{
				Name:          name,
				Type:          node.Type,
				LatencyMS:     int(snapshot.LatencyMS),
				Alive:         snapshot.Available,
				Score:         snapshot.Score,
				SuccessCount:  snapshot.Successes,
				FailureCount:  snapshot.Failures,
				TotalChecks:   snapshot.Samples,
				FailureRate:   snapshot.FailureRate,
				CooldownUntil: snapshot.CooldownUntil,
			})
		}
		groups = append(groups, view)
	}
	return groups
}

// buildAutoStableHealthFallbackLocked returns a lightweight health view built
// solely from profile data — used when the auto-stable manager has not yet
// been initialised.
func (s *Service) buildAutoStableHealthFallbackLocked() []AutoStableGroupHealth {
	nodeByName := make(map[string]string, len(s.active.Proxies))
	for _, node := range s.active.Proxies {
		nodeByName[node.Name] = node.Type
	}

	views := make([]AutoStableGroupHealth, 0, len(s.active.ProxyGroups))
	for _, group := range s.active.ProxyGroups {
		view := AutoStableGroupHealth{
			Name:     group.Name,
			Type:     group.Type,
			Selected: group.SelectedProxy,
			Proxies:  make([]AutoStableNodeHealth, 0, len(group.Proxies)),
		}
		for _, name := range group.Proxies {
			proxyType := nodeByName[name]
			view.Proxies = append(view.Proxies, AutoStableNodeHealth{
				Name:  name,
				Type:  proxyType,
				Alive: proxyType != "" || name == "DIRECT" || name == "AUTO",
			})
		}
		views = append(views, view)
	}
	return views
}
