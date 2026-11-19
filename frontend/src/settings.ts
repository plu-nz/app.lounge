import { createSignal } from 'solid-js';
import { player, settingsApi, type Settings } from './api';
import { setLanguage } from './i18n';

const [settings, setSignal] = createSignal<Settings>({
  language: '',
  audioLanguages: [],
  subtitleLanguages: [],
  subtitleScale: 1,
  autoplayNext: true,
  accent: 'blue',
});
export { settings };

/** apply pushes preferences to the UI and to mpv. */
function apply(s: Settings) {
  setLanguage(s.language);
  document.querySelector('.app')?.setAttribute('data-plz-accent', s.accent || 'blue');
  void player.set('alang', (s.audioLanguages ?? []).join(',')).catch(() => undefined);
  void player.set('slang', (s.subtitleLanguages ?? []).join(',')).catch(() => undefined);
  void player.set('sub-scale', s.subtitleScale || 1).catch(() => undefined);
}

export async function loadSettings() {
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
