package browser

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"browseforge/internal/config"
	"browseforge/internal/fingerprint"
	"browseforge/internal/profile"

	"github.com/playwright-community/playwright-go"
)

func (m *Manager) launchChromium(p *profile.Profile) (*Session, error) {
	chromiumPath := m.cfg.CloakBrowserPath
	if chromiumPath == "" {
		return nil, fmt.Errorf("cloakbrowser_path not configured")
	}

	userDataDir, err := filepath.Abs(filepath.Join(p.ProfileDir, "browser-data"))
	if err != nil {
		return nil, fmt.Errorf("browser data path: %w", err)
	}
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return nil, fmt.Errorf("create browser data dir: %w", err)
	}
	cleanProfileLocks(userDataDir)

	absChromiumPath, err := filepath.Abs(chromiumPath)
	if err != nil {
		return nil, fmt.Errorf("cloakbrowser path: %w", err)
	}

	args := []string{
		"--no-first-run",
		"--test-type",
	}
	if p.FingerprintSeed > 0 {
		args = append(args, fmt.Sprintf("--fingerprint=%d", p.FingerprintSeed))
	}

	var tz, locale string
	effectiveProxy := m.effectiveProxy(p)
	if effectiveProxy.Proxy != nil {
		proxy := effectiveProxy.Proxy
		tz, locale = fingerprint.DetectProxyGeoResult(proxy.Type, proxy.Host, proxy.Port, proxy.Username, proxy.Password)
		args = append(args, "--fingerprint-webrtc-ip=auto")
	} else {
		tz, locale = fingerprint.DetectLocalGeoResult()
	}
	args = append(args,
		"--fingerprint-timezone="+tz,
		"--fingerprint-locale="+locale,
	)

	if runtime.GOOS == "linux" {
		args = append(args, "--fingerprint-platform=windows")
	}

	if p.Fingerprint != nil {
		if v, ok := p.Fingerprint["navigator.hardwareConcurrency"]; ok {
			args = append(args, fmt.Sprintf("--fingerprint-hardware-concurrency=%v", v))
		}
		if v, ok := p.Fingerprint["screen.width"]; ok {
			args = append(args, fmt.Sprintf("--fingerprint-screen-width=%v", v))
		}
		if v, ok := p.Fingerprint["screen.height"]; ok {
			args = append(args, fmt.Sprintf("--fingerprint-screen-height=%v", v))
		}
	}

	if m.cfg.NoSandbox {
		args = append(args, "--no-sandbox")
	}

	if _, err := os.Stat("/usr/share/fonts"); err == nil {
		args = append(args, "--fingerprint-fonts-dir=/usr/share/fonts")
	}

	baseArgs := append([]string(nil), args...)
	args, err = applyCloakBrowserLaunchPolicy(baseArgs, userDataDir, m.cfg.CloakBrowser, false)
	if err != nil {
		return nil, err
	}

	downloadsDir, err := filepath.Abs(filepath.Join(p.ProfileDir, "downloads"))
	if err != nil {
		return nil, fmt.Errorf("downloads path: %w", err)
	}
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("create downloads dir: %w", err)
	}

	prefsDir := filepath.Join(userDataDir, "Default")
	if err := os.MkdirAll(prefsDir, 0755); err != nil {
		return nil, fmt.Errorf("create chromium prefs dir: %w", err)
	}
	prefsPath := filepath.Join(prefsDir, "Preferences")
	prefs := map[string]any{}
	if data, err := os.ReadFile(prefsPath); err == nil {
		if err := json.Unmarshal(data, &prefs); err != nil {
			return nil, fmt.Errorf("decode chromium preferences: %w", err)
		}
	}
	prefs["savefile"] = map[string]any{"default_directory": downloadsDir}
	prefs["download"] = map[string]any{"default_directory": downloadsDir, "prompt_for_download": false}
	out, err := json.Marshal(prefs)
	if err != nil {
		return nil, fmt.Errorf("encode chromium preferences: %w", err)
	}
	if err := os.WriteFile(prefsPath, out, 0644); err != nil {
		return nil, fmt.Errorf("write chromium preferences: %w", err)
	}

	ignoreArgs := []string{
		"--enable-automation",
		"--disable-blink-features=AutomationControlled",
		"--host-resolver-rules=MAP * ~NOTFOUND , EXCLUDE 127.0.0.1",
	}
	if !m.cfg.NoSandbox {
		ignoreArgs = append(ignoreArgs, "--no-sandbox")
	}

	launch := func(launchArgs []string) (playwright.BrowserContext, *SOCKS5Relay, error) {
		opts := playwright.BrowserTypeLaunchPersistentContextOptions{
			ExecutablePath:    playwright.String(absChromiumPath),
			Headless:          playwright.Bool(false),
			AcceptDownloads:   playwright.Bool(true),
			Args:              launchArgs,
			Viewport:          &playwright.Size{Width: 1280, Height: 800},
			IgnoreDefaultArgs: ignoreArgs,
		}

		var relay *SOCKS5Relay
		if effectiveProxy.Proxy != nil {
			proxy := effectiveProxy.Proxy
			needsRelay := proxy.Type == "socks5" && proxy.Username != ""
			if needsRelay {
				upstream := fmt.Sprintf("%s:%d", proxy.Host, proxy.Port)
				var localAddr string
				relay, localAddr, err = StartSOCKS5Relay(upstream, proxy.Username, proxy.Password)
				if err != nil {
					return nil, nil, fmt.Errorf("socks5 relay: %w", err)
				}
				opts.Proxy = &playwright.Proxy{Server: "socks5://" + localAddr}
			} else {
				server := fmt.Sprintf("%s://%s:%d", proxy.Type, proxy.Host, proxy.Port)
				opts.Proxy = &playwright.Proxy{
					Server:   server,
					Username: playwright.String(proxy.Username),
					Password: playwright.String(proxy.Password),
				}
			}
		}

		ctx, err := m.pw.Chromium.LaunchPersistentContext(userDataDir, opts)
		if err != nil {
			if relay != nil {
				relay.Close()
			}
			return nil, nil, err
		}
		return ctx, relay, nil
	}

	ctx, relay, err := launch(args)
	fallbackAttempted := false
	if err != nil {
		if m.cfg.CloakBrowser != nil &&
			(m.cfg.CloakBrowser.RepairTransientCacheOnLaunchFailure || m.cfg.CloakBrowser.AutoSafeGPUFallback) &&
			isChromiumGPUOrCacheLaunchFailure(err) {
			slog.Warn("repairing transient chromium cache after launch failure", "profile", p.ID, "userDataDir", userDataDir, "error", err)
			repairTransientChromiumData(userDataDir)
		}
		if shouldAutoFallbackCloakBrowserLaunch(m.cfg.CloakBrowser, err) {
			fallbackAttempted = true
			slog.Warn("retrying CloakBrowser launch with safe GPU fallback", "profile", p.ID, "userDataDir", userDataDir, "error", err)
			if len(m.sessions) > 0 {
				m.dropSessionsLocked("playwright driver restart before CloakBrowser safe GPU fallback")
			}
			if restartErr := m.restartPlaywright(); restartErr != nil {
				return nil, fmt.Errorf("launch chromium: %w; safe GPU fallback playwright restart failed: %v", humanizeError(err), restartErr)
			}
			fallbackArgs, fallbackArgErr := applyCloakBrowserLaunchPolicy(baseArgs, userDataDir, m.cfg.CloakBrowser, true)
			if fallbackArgErr != nil {
				return nil, fallbackArgErr
			}
			ctx, relay, err = launch(fallbackArgs)
			if err == nil {
				slog.Info("CloakBrowser launch recovered with safe GPU fallback", "profile", p.ID)
			}
		}
	}
	if err != nil {
		if fallbackAttempted {
			return nil, fmt.Errorf("launch chromium: %w", noManagerRetryError{err: humanizeError(err)})
		}
		return nil, fmt.Errorf("launch chromium: %w", humanizeError(err))
	}

	dlDir := downloadsDir
	onDl := func(d playwright.Download) { go d.SaveAs(filepath.Join(dlDir, d.SuggestedFilename())) }
	for _, pg := range ctx.Pages() {
		pg.OnDownload(onDl)
	}
	ctx.OnPage(func(pg playwright.Page) { pg.OnDownload(onDl) })

	pages := ctx.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = ctx.NewPage()
		if err != nil {
			if relay != nil {
				relay.Close()
			}
			ctx.Close()
			return nil, fmt.Errorf("new page: %w", err)
		}
	}

	return &Session{
		ID:             fmt.Sprintf("sess_%s", p.ID),
		ProfileID:      p.ID,
		Engine:         "chromium",
		Context:        ctx,
		Page:           page,
		relay:          relay,
		ProfileDir:     p.ProfileDir,
		UserDataDir:    userDataDir,
		ExecutablePath: absChromiumPath,
	}, nil
}

func applyCloakBrowserLaunchPolicy(args []string, userDataDir string, policy *config.CloakBrowserConfig, fallback bool) ([]string, error) {
	out := append([]string(nil), args...)
	if policy == nil {
		return out, nil
	}

	if policy.SafeGPU || fallback {
		out = appendUniqueChromiumArgs(out,
			"--disable-gpu",
			"--disable-gpu-compositing",
			"--disable-gpu-sandbox",
			"--disable-gpu-shader-disk-cache",
			"--in-process-gpu",
		)
	}
	if policy.IsolatedRuntimeCache || fallback {
		cacheDir := filepath.Join(userDataDir, "BrowseForgeRuntimeCache", fmt.Sprintf("cache-%d-%d", os.Getpid(), time.Now().UnixNano()))
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return nil, fmt.Errorf("create chromium runtime cache dir: %w", err)
		}
		out = append(out, "--disk-cache-dir="+cacheDir)
	}
	out = appendUniqueChromiumArgs(out, sanitizeExtraChromiumArgs(policy.ExtraArgs)...)
	return out, nil
}

func shouldAutoFallbackCloakBrowserLaunch(policy *config.CloakBrowserConfig, err error) bool {
	return policy != nil &&
		policy.AutoSafeGPUFallback &&
		isChromiumGPUOrCacheLaunchFailure(err) &&
		(!policy.SafeGPU || !policy.IsolatedRuntimeCache)
}

func appendUniqueChromiumArgs(args []string, extra ...string) []string {
	seen := make(map[string]bool, len(args)+len(extra))
	for _, arg := range args {
		seen[arg] = true
	}
	for _, arg := range extra {
		if arg == "" || seen[arg] {
			continue
		}
		seen[arg] = true
		args = append(args, arg)
	}
	return args
}
