import { createSignal } from 'solid-js';
import { player, settingsApi, themeApi, type GtkTheme, type Settings } from './api';
import { setLanguage } from './i18n';

const [settings, setSignal] = createSignal<Settings>({
  language: '',
  audioLanguages: [],
  subtitleLanguages: [],
  subtitleScale: 1,
  autoplayNext: true,
  theme: 'gtk',
  accent: 'theme',
  sourceMode: 'ask',
  sourceMaxHeight: 0,
  sourceMaxSizeGb: 0,
  sourcePrefer: '',
  sourceMinSeeders: 0,
});
export { settings };

const [gtkTheme, setGtkTheme] = createSignal<GtkTheme>();
export { gtkTheme };

/** followsGtk reports whether the UI takes its colors from the GTK theme. */
export function followsGtk(s: Settings = settings()): boolean {
  return (s.theme || 'gtk') === 'gtk' && Boolean(gtkTheme()?.background);
}

const mix = (a: string, pct: number, b: string) => `color-mix(in srgb, ${a} ${pct}%, ${b})`;

/**
 * gtkTokens maps a GTK theme onto the Tinta tokens. Colors a theme
 * does not define are mixed from its background and text, and the named
 * accents keep Lounge's hues.
 */
function gtkTokens(g: GtkTheme): Record<string, string> {
  const bg = g.background;
  const text = g.text || (g.dark ? '#ffffff' : '#000000');
  const tokens: Record<string, string> = {
    '--plz-bg': bg,
    '--plz-surface': g.surface && g.surface !== bg ? g.surface : mix(text, 6, bg),
    '--plz-raised': mix(text, 14, bg),
    '--plz-line': mix(text, 18, bg),
    '--plz-text': text,
    '--plz-muted': mix(text, 82, bg),
    '--plz-subtle': mix(text, 55, bg),
    '--plz-theme-accent': g.accent,
    '--plz-on-accent': g.onAccent,
  };
  if (g.error) tokens['--plz-red'] = g.error;
  if (g.success) tokens['--plz-green'] = g.success;
  if (g.warning) tokens['--plz-orange'] = g.warning;
  return tokens;
}

let applied: string[] = [];

function applyTheme(s: Settings) {
  const app = document.querySelector<HTMLElement>('.app');
  if (!app) return;
  for (const name of applied) app.style.removeProperty(name);
  applied = [];
  const g = gtkTheme();
  if (followsGtk(s) && g) {
    for (const [name, value] of Object.entries(gtkTokens(g))) {
      if (!value) continue;
      app.style.setProperty(name, value);
      applied.push(name);
    }
    app.style.colorScheme = g.dark ? 'dark' : 'light';
  } else {
    app.style.colorScheme = '';
  }
  app.dataset.plzMode = followsGtk(s) && g && !g.dark ? 'light' : 'dark';
  app.setAttribute('data-plz-accent', s.accent || 'blue');
}

async function loadGtkTheme() {
  const theme = await themeApi.gtk().catch(() => undefined);
  setGtkTheme(theme);
  applyTheme(settings());
}

/** apply pushes preferences to the UI and to mpv. */
function apply(s: Settings) {
  setLanguage(s.language);
  applyTheme(s);
  void player.set('alang', (s.audioLanguages ?? []).join(',')).catch(() => undefined);
  void player.set('slang', (s.subtitleLanguages ?? []).join(',')).catch(() => undefined);
  void player.set('sub-scale', s.subtitleScale || 1).catch(() => undefined);
}

export async function loadSettings() {
  themeApi.onChange(() => void loadGtkTheme());
  await loadGtkTheme();
  const s = await settingsApi.get();
  s.audioLanguages ??= [];
  s.subtitleLanguages ??= [];
  setSignal(s);
  apply(s);
}

export async function updateSettings(patch: Partial<Settings>) {
  const next = { ...settings(), ...patch };
  setSignal(next);
  apply(next);
  await settingsApi.set(next);
}
