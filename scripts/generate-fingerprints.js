#!/usr/bin/env node
// Generate fingerprint pool using fingerprint-suite's Bayesian network
// Usage: node generate-fingerprints.js --browser firefox --os windows --count 500

const { FingerprintGenerator } = require('fingerprint-generator');
const fs = require('fs');
const path = require('path');

const args = process.argv.slice(2);
const getArg = (name, def) => {
  const i = args.indexOf(`--${name}`);
  return i >= 0 && args[i + 1] ? args[i + 1] : def;
};

const browser = getArg('browser', 'firefox');
const os = getArg('os', 'windows');
const count = parseInt(getArg('count', '500'));
const outputDir = getArg('output', 'data');

// Load WebGL profiles for complete fingerprinting
const webglProfilesPath = path.join(__dirname, '..', 'data', 'webgl-profiles', 'webgl-profiles.json');
let webglProfiles = [];
try { webglProfiles = JSON.parse(fs.readFileSync(webglProfilesPath, 'utf8')); } catch {}

function findWebGLProfile(renderer) {
  if (!renderer) return null;
  return webglProfiles.find(p => renderer.includes(p.match)) || null;
}

// Font subsets per OS (randomly select 60-80% to simulate real users)
const macFonts = ["Arial","Arial Black","Comic Sans MS","Courier New","Georgia","Helvetica","Helvetica Neue","Impact","Lucida Grande","Menlo","Monaco","Palatino","Times New Roman","Trebuchet MS","Verdana","Gill Sans","Futura","Optima","Baskerville","Didot","American Typewriter","Avenir","Avenir Next","Hoefler Text","Apple SD Gothic Neo","Hiragino Sans","PingFang SC","PingFang TC","Songti SC"];
const winFonts = ["Arial","Arial Black","Calibri","Cambria","Comic Sans MS","Consolas","Constantia","Corbel","Courier New","Ebrima","Franklin Gothic Medium","Gabriola","Georgia","Impact","Lucida Console","Lucida Sans Unicode","MS Gothic","MS PGothic","Malgun Gothic","Microsoft YaHei","Palatino Linotype","Segoe UI","SimSun","Tahoma","Times New Roman","Trebuchet MS","Verdana","Yu Gothic"];
const linuxFonts = ["Arimo","Cousine","Tinos","Noto Sans","Noto Sans SC","Noto Sans TC","Noto Sans JP","Noto Sans KR","Noto Serif","Noto Sans Symbols","Noto Sans Symbols 2","Twemoji Mozilla"];

function randomFontSubset(osFonts) {
  const ratio = 0.6 + Math.random() * 0.2; // 60-80%
  const shuffled = [...osFonts].sort(() => Math.random() - 0.5);
  return shuffled.slice(0, Math.floor(shuffled.length * ratio));
}

console.log(`Generating ${count} ${browser}/${os} fingerprints (${webglProfiles.length} WebGL profiles loaded)...`);

const generator = new FingerprintGenerator({ browsers: [browser], operatingSystems: [os] });
const fingerprints = [];

for (let i = 0; i < count; i++) {
  const { fingerprint } = generator.getFingerprint();

  // Select font subset based on target OS
  const osFonts = os === 'macos' ? macFonts : os === 'windows' ? winFonts : linuxFonts;

  // Build the BrowseForge fingerprint-pool format.
  // Firefox/Camoufox consumes these flat keys through CAMOU_CONFIG. Chromium/CloakBrowser
  // primarily uses fingerprint_seed, with selected fields available as explicit overrides.
  const fingerprintConfig = {
    // Navigator
    'navigator.userAgent': fingerprint.navigator.userAgent,
    'navigator.platform': fingerprint.navigator.platform,
    'navigator.hardwareConcurrency': fingerprint.navigator.hardwareConcurrency,
    'navigator.language': fingerprint.navigator.language,
    'navigator.languages': fingerprint.navigator.languages,
    'navigator.oscpu': fingerprint.navigator.oscpu || '',
    // Screen
    'screen.width': fingerprint.screen.width,
    'screen.height': fingerprint.screen.height,
    'screen.availWidth': fingerprint.screen.availWidth,
    'screen.availHeight': fingerprint.screen.availHeight,
    'screen.colorDepth': fingerprint.screen.colorDepth,
    // Window
    'window.outerWidth': fingerprint.screen.outerWidth,
    'window.outerHeight': fingerprint.screen.outerHeight,
    'window.innerWidth': fingerprint.screen.innerWidth,
    'window.innerHeight': fingerprint.screen.innerHeight,
    'window.devicePixelRatio': fingerprint.screen.devicePixelRatio,
    // WebGL (renderer/vendor — may be overridden by full profile below)
    'webGl:vendor': fingerprint.videoCard?.vendor || '',
    'webGl:renderer': fingerprint.videoCard?.renderer || '',
    // Canvas & Font anti-fingerprinting seeds
    'canvas:seed': Math.floor(Math.random() * 4294967295),
    'fonts:spacing_seed': Math.floor(Math.random() * 4294967295),
    // Fonts (random subset to avoid full-list detection)
    'fonts': randomFontSubset(osFonts),
  };

  // Never ship a renderer/vendor-only identity. Camoufox requires the complete
  // WebGL profile to keep getParameter() consistent with the claimed GPU.
  const webglProfile = findWebGLProfile(fingerprintConfig['webGl:renderer']);
  if (webglProfile) {
    const { match, ...webglData } = webglProfile;
    Object.assign(fingerprintConfig, webglData);
  } else {
    delete fingerprintConfig['webGl:renderer'];
    delete fingerprintConfig['webGl:vendor'];
  }

  fingerprints.push(fingerprintConfig);
}

fs.mkdirSync(outputDir, { recursive: true });
const outPath = path.join(outputDir, `fingerprints-${browser}-${os}.json`);
fs.writeFileSync(outPath, JSON.stringify(fingerprints, null, 2));
console.log(`Written ${fingerprints.length} fingerprints to ${outPath}`);
