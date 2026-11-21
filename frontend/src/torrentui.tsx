import { Layers, RotateCw } from 'lucide-solid';
import { Show } from 'solid-js';
import type { TorrentStats } from './api';
import { t } from './i18n';
import { focusEl } from './nav';

function bytes(n: number): string {
  if (n >= 1e9) return `${(n / 1e9).toFixed(2)} GB`;
  if (n >= 1e6) return `${(n / 1e6).toFixed(1)} MB`;
  return `${Math.round(n / 1e3)} kB`;
}

/** loadingText says what the torrent is doing while the video is not playing yet. */
export function loadingText(s: TorrentStats | null): string {
  if (!s) return t('player.connecting');
  if (s.phase === 'connecting') return t('player.connecting');
  if (s.phase === 'metadata') return t('player.metadata', { p: s.peers });
  return t('player.buffering', { n: s.bufferedPercent });
}

/**
 * StatsPanel is a read-only overlay. It is never focusable and ignores the
 * pointer, so it cannot disturb the controls' navigation.
 */
export function StatsPanel(props: { stats: TorrentStats | null; duration: number }) {
  const ahead = () => {
    const s = props.stats;
    if (!s) return '';
    const base = bytes(s.bufferedBytes);
    if (s.fileSize > 0 && props.duration > 0) return `${base} · ${Math.round((s.bufferedBytes / s.fileSize) * props.duration)}s`;
    return base;
  };
  return (
    <div class="torrent-stats" aria-hidden="true">
      <Show when={props.stats} fallback={<p>{t('player.connecting')}</p>}>
        {(s) => (
          <dl>
            <dt>{t('player.statsPeers')}</dt>
            <dd>{t('player.statsPeersValue', { p: s().peers, t: s().totalPeers, s: s().seeders })}</dd>
            <dt>{t('player.statsDown')}</dt>
            <dd>{bytes(s().downloadRate)}/s</dd>
            <dt>{t('player.statsUp')}</dt>
            <dd>{bytes(s().uploadRate)}/s</dd>
            <dt>{t('player.statsDone')}</dt>
            <dd>
              {bytes(s().downloaded)} / {bytes(s().size)} · {Math.round(s().progress * 100)}%
            </dd>
            <dt>{t('player.statsBuffer')}</dt>
            <dd>{ahead()}</dd>
          </dl>
        )}
      </Show>
    </div>
  );
}

/** NoSeeders replaces the loading spinner when no peer answered for too long. */
export function NoSeeders(props: { onOtherSource: () => void; onKeepWaiting: () => void }) {
  return (
    <div class="center" style={{ 'pointer-events': 'auto' }}>
      <p class="error" style={{ 'font-size': '1.3rem' }}>
        {t('player.noSeeders')}
      </p>
      <p>{t('player.noSeedersHint')}</p>
      <div class="btn-row">
        <button class="btn primary" data-nav data-nav-default ref={(el) => queueMicrotask(() => focusEl(el))} onClick={props.onOtherSource}>
          <Layers />
          {t('player.anotherSource')}
        </button>
        <button class="btn" data-nav onClick={props.onKeepWaiting}>
          <RotateCw />
          {t('player.keepWaiting')}
        </button>
      </div>
    </div>
  );
}
