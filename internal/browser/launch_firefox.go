package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"browseforge/internal/fingerprint"
	"browseforge/internal/profile"

	"github.com/playwright-community/playwright-go"
)

func (m *Manager) launchFirefox(p *profile.Profile) (*Session, error) {
	camoufoxPath := m.cfg.CamoufoxPath
	if _, err := os.Stat(camoufoxPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("camoufox not found: %s", camoufoxPath)
	}

	// Build CAMOU_CONFIG: start from profile fingerprint, then overlay GeoIP.
	// WebGL strategy: only pass WebGL fields if we have a complete profile.
	config := make(map[string]any)
	hasFullWebGL := false
	for k, v := range p.Fingerprint {
		config[k] = v
		if k == "webGl:supportedExtensions" {
			hasFullWebGL = true
		}
	}
	if !hasFullWebGL {
		delete(config, "webGl:renderer")
		delete(config, "webGl:vendor")
	}

	effectiveProxy := m.effectiveProxy(p)
	if effectiveProxy.Proxy != nil {
		proxy := effectiveProxy.Proxy
		tz, locale := fingerprint.DetectProxyGeoResult(proxy.Type, proxy.Host, proxy.Port, proxy.Username, proxy.Password)
		config["timezone"] = tz
		parts := splitLocale(locale)
		config["locale:language"] = parts[0]
		config["locale:region"] = parts[1]
	} else {
		tz, locale := fingerprint.DetectLocalGeoResult()
		config["timezone"] = tz
		parts := splitLocale(locale)
		config["locale:language"] = parts[0]
		config["locale:region"] = parts[1]
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("encode camoufox config: %w", err)
	}

	userDataDir, err := filepath.Abs(filepath.Join(p.ProfileDir, "browser-data"))
	if err != nil {
		return nil, fmt.Errorf("browser data path: %w", err)
	}
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return nil, fmt.Errorf("create browser data dir: %w", err)
	}
	cleanProfileLocks(userDataDir)

	absPath, err := filepath.Abs(camoufoxPath)
	if err != nil {
		return nil, fmt.Errorf("camoufox path: %w", err)
	}

	downloadsDir, err := filepath.Abs(filepath.Join(p.ProfileDir, "downloads"))
	if err != nil {
		return nil, fmt.Errorf("downloads path: %w", err)
	}
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("create downloads dir: %w", err)
	}

	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		ExecutablePath:  playwright.String(absPath),
		Headless:        playwright.Bool(false),
		AcceptDownloads: playwright.Bool(true),
		Env: map[string]string{
			"CAMOU_CONFIG":          string(configJSON),
			"DISPLAY":               os.Getenv("DISPLAY"),
			"HOME":                  os.Getenv("HOME"),
			"LIBGL_ALWAYS_SOFTWARE": os.Getenv("LIBGL_ALWAYS_SOFTWARE"),
		},
		FirefoxUserPrefs: map[string]any{
			"xpinstall.signatures.required":          false,
			"browser.download.folderList":            2,
			"browser.download.dir":                   downloadsDir,
			"browser.download.useDownloadDir":        true,
			"browser.helperApps.neverAsk.saveToDisk": "application/octet-stream,image/jpeg,image/png,application/pdf,application/zip",
			"webgl.disabled":                         false,
			"webgl.force-enabled":                    true,
			"webgl.forbid-software":                  false,
		},
		Viewport: &playwright.Size{Width: 1280, Height: 800},
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
				return nil, fmt.Errorf("socks5 relay: %w", err)
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

	ctx, err := m.pw.Firefox.LaunchPersistentContext(userDataDir, opts)
	if err != nil {
		if relay != nil {
			relay.Close()
		}
		return nil, fmt.Errorf("launch firefox: %w", humanizeError(err))
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
		Engine:         "firefox",
		Context:        ctx,
		Page:           page,
		relay:          relay,
		ProfileDir:     p.ProfileDir,
		UserDataDir:    userDataDir,
		ExecutablePath: absPath,
	}, nil
}
