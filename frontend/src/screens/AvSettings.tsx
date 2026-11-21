import { createResource, Show } from 'solid-js';
import { player } from '../api';
import { t } from '../i18n';
import { settings, updateSettings } from '../settings';

const audioModes = ['stereo', 'auto', 'passthrough'];
const hdrModes = ['tonemap', 'off'];

function cycle<T>(list: T[], current: T): T {
  return list[(list.indexOf(current) + 1) % list.length];
}

const noteStyle = {
  margin: '-0.2rem 1.4rem 0.2rem',
  'font-size': '0.9rem',
  color: 'var(--plz-text-muted, var(--plz-text))',
  opacity: '0.8',
};

/** AvSettings lists the audio output and HDR options of the settings page. */
export function AvSettings() {
  const s = settings;
  const [devices] = createResource(() => player.audioDevices().catch(() => []));
  const mode = () => (audioModes.includes(s().audioOutput) ? s().audioOutput : 'stereo');
  const hdr = () => (hdrModes.includes(s().hdr) ? s().hdr : 'tonemap');
  const modeLabel = () =>
    ({ stereo: t('settings.audioStereo'), auto: t('settings.audioAuto'), passthrough: t('settings.audioPassthrough') })[mode()];
  const deviceLabel = () => {
    const name = s().audioDevice;
    if (!name || name === 'auto') return t('settings.audioDefault');
    return devices()?.find((d) => d.name === name)?.description ?? name;
  };
  // The list is cycled with Enter, like the other options; "auto" is the system default.
  const nextDevice = () => {
    const names = (devices() ?? []).map((d) => d.name);
    if (!names.includes('auto')) names.unshift('auto');
    void updateSettings({ audioDevice: cycle(names, s().audioDevice || 'auto') });
  };
  return (
    <>
      <button class="setting" data-nav onClick={() => void updateSettings({ audioOutput: cycle(audioModes, mode()) })}>
        <span>{t('settings.audioOutput')}</span>
        <span>{modeLabel()}</span>
      </button>
      <Show when={mode() === 'passthrough'}>
        <p style={noteStyle}>{t('settings.audioPassthroughNote')}</p>
      </Show>
      <button class="setting" data-nav onClick={nextDevice}>
        <span>{t('settings.audioDevice')}</span>
        <span>{deviceLabel()}</span>
      </button>
      <button class="setting" data-nav onClick={() => void updateSettings({ nightMode: !s().nightMode })}>
        <span>{t('settings.nightMode')}</span>
        <span>{s().nightMode ? t('settings.on') : t('settings.off')}</span>
      </button>
      <button class="setting" data-nav onClick={() => void updateSettings({ hdr: cycle(hdrModes, hdr()) })}>
        <span>{t('settings.hdr')}</span>
        <span>{hdr() === 'tonemap' ? t('settings.hdrTonemap') : t('settings.hdrOff')}</span>
      </button>
      <p style={noteStyle}>{t('settings.hdrNote')}</p>
    </>
  );
}
