import { createMemo, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { AudioLines, Gauge, Layers, Pause, Play, RotateCcw, RotateCw, SkipForward, Subtitles } from 'lucide-solid';
import { content, library, player, tracksApi, type TorrentStatus, type TrackInfo, type Video } from '../api';
import { Panel, Spinner, errorText } from '../components';
import { t } from '../i18n';
import { focusEl, onBack } from '../nav';
import { pop, replaceTop, type PlayContext } from '../router';
import { settings } from '../settings';
import { findStream, streamKey, StreamPicker, toRequest } from '../streams';
import { episodeLabel } from './Details';

type Track = { id: number; type: 'video' | 'audio' | 'sub'; lang?: string; title?: string; codec?: string; external?: boolean; selected?: boolean; default?: boolean; forced?: boolean; 'demux-channel-count'?: number };

const hideAfter = 4000;
const mouseRevealDistance = 24;
const mouseRevealWindow = 750;
// Remote "menu" buttons arrive as the ContextMenu key (XKB Menu).
const menuKeys = new Set(['ContextMenu', 'Menu']);
const seekStep = 10;

export function formatTime(s: number): string {
  if (!Number.isFinite(s) || s < 0) s = 0;
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = Math.floor(s % 60);
  const mm = h ? String(m).padStart(2, '0') : String(m);
  return `${h ? `${h}:` : ''}${mm}:${String(sec).padStart(2, '0')}`;
}

function formatBytes(n: number): string {
  if (n > 1e6) return `${(n / 1e6).toFixed(1)} MB`;
  return `${Math.round(n / 1e3)} kB`;
}

const langNames = new Intl.DisplayNames(undefined, { type: 'language' });
function trackLabel(tr: Track): string {
  let lang = tr.lang ?? '';
  try {
    if (lang) lang = langNames.of(lang) ?? lang;
  } catch {
    /* keep the code */
  }
  const parts = [lang, tr.title && tr.title !== tr.lang ? tr.title : '', tr.codec?.toUpperCase() ?? ''];
  if (tr['demux-channel-count']) parts.push(`${tr['demux-channel-count']}ch`);
  return parts.filter(Boolean).join(' · ') || t('player.track', { n: tr.id });
}

export function Player(props: { play: PlayContext; active: boolean }) {
  const ctx = props.play;
  const [time, setTime] = createSignal(0);
  const [duration, setDuration] = createSignal(0);
  const [paused, setPaused] = createSignal(false);
  const [buffering, setBuffering] = createSignal(true);
  const [cacheTime, setCacheTime] = createSignal(0);
  const [started, setStarted] = createSignal(false);
  const [tracks, setTracks] = createSignal<Track[]>([]);
  const [speed, setSpeed] = createSignal(1);
  const [error, setError] = createSignal('');
  const [visible, setVisible] = createSignal(true);
  const [menu, setMenu] = createSignal<'audio' | 'sub' | 'speed' | 'sources'>();
  const [osd, setOsd] = createSignal('');
  const [torrent, setTorrent] = createSignal<TorrentStatus | null>(null);
  const [countdown, setCountdown] = createSignal<number>();

  const nextVideo = createMemo((): Video | undefined => {
    if (!ctx.video) return undefined;
    const i = ctx.videos.findIndex((v) => v.id === ctx.video!.id);
    return i >= 0 ? ctx.videos[i + 1] : undefined;
  });

  let hideTimer: number | undefined;
  let osdTimer: number | undefined;
  let saveTimer: number | undefined;
  let ended = false;
  let closing = false;
  let mousePosition: { x: number; y: number } | undefined;
  let mouseMoveStart: { x: number; y: number; at: number } | undefined;

  const showControls = (focus = true) => {
    setVisible(true);
    document.body.classList.remove('mouse-hidden');
    clearTimeout(hideTimer);
    hideTimer = window.setTimeout(() => {
      if (!paused() && !menu() && started()) {
        setVisible(false);
        document.body.classList.add('mouse-hidden');
        mouseMoveStart = undefined;
      }
    }, hideAfter);
    if (focus && !document.activeElement?.closest('.player .controls')) {
      const btn = document.querySelector<HTMLElement>('.player [data-nav-default]');
      if (btn) focusEl(btn);
    }
  };

  const hideControls = () => {
    clearTimeout(hideTimer);
    setVisible(false);
    document.body.classList.add('mouse-hidden');
    mouseMoveStart = undefined;
  };

  // The Menu key can arrive as a keydown and then a contextmenu event; act once.
  let lastMenuToggle = 0;
  const toggleFromMenu = () => {
    if (performance.now() - lastMenuToggle < 400) return;
    lastMenuToggle = performance.now();
    if (visible() && started() && !paused() && !error()) hideControls();
    else showControls();
  };

  const flash = (text: string) => {
    setOsd(text);
    clearTimeout(osdTimer);
    osdTimer = window.setTimeout(() => setOsd(''), 1200);
  };

  const save = () => {
    if (!started() || duration() <= 0) return;
    void library.save({
      id: ctx.meta.id,
      type: ctx.meta.type,
      name: ctx.meta.name,
      poster: ctx.meta.poster,
      background: ctx.meta.background,
      logo: ctx.meta.logo,
      videoId: ctx.video?.id ?? ctx.meta.id,
      season: ctx.video?.season,
      episode: ctx.video?.episode ?? ctx.video?.number,
      episodeTitle: ctx.video?.name ?? ctx.video?.title,
      position: time(),
      duration: duration(),
      streamKey: ctx.streamKey,
    });
  };

  const exit = () => {
    if (closing) return;
    closing = true;
    save();
    void player.stop();
    pop();
  };

  const seek = (delta: number) => {
    void player.command('seek', delta, 'relative');
    flash(`${delta > 0 ? '+' : ''}${delta}s`);
  };

  const togglePause = () => void player.command('cycle', 'pause');

  const playNext = async () => {
    const v = nextVideo();
    if (!v) return exit();
    save();
    closing = true;
    setCountdown(undefined);
    setError('');
    setBuffering(true);
    setStarted(false);
    try {
      const groups = await content.streams(ctx.meta.type, v.id);
      const found = findStream(groups, { bingeGroup: ctx.bingeGroup, addonId: ctx.addonId });
      if (!found) throw new Error(t('streams.none'));
      replaceTop({
        name: 'player',
        play: {
          ...ctx,
          request: { ...toRequest(found.stream), start: 0, title: `${ctx.meta.name} · ${episodeLabel(v)}` },
          video: v,
          streamKey: streamKey(found.stream),
          bingeGroup: found.stream.behaviorHints?.bingeGroup,
          addonId: found.group.addonId,
        },
      });
    } catch (err) {
      closing = false;
      setError(errorText(err));
    }
  };

  // Adds the addon subtitles, then picks the audio and subtitle tracks from
  // this title's remembered choices and the preferences.
  const addExternalSubtitles = async () => {
    const videoId = ctx.video?.id ?? ctx.meta.id;
    const subs = await content.subtitles(ctx.meta.type, videoId).catch(() => []);
    for (const s of subs.slice(0, 40)) {
      await player.command('sub-add', s.url, 'cached', s.lang, s.lang).catch(() => undefined);
    }
    const list = await player.get<Track[]>('track-list').catch(() => [] as Track[]);
    const info: TrackInfo[] = list
      .filter((tr) => tr.type !== 'video')
      .map((tr) => ({ id: tr.id, type: tr.type, lang: tr.lang ?? '', title: tr.title ?? '', default: Boolean(tr.default), forced: Boolean(tr.forced), external: Boolean(tr.external) }));
    const pick = await tracksApi.select(ctx.meta.id, info).catch(() => undefined);
    if (!pick) return;
    if (pick.audioId && !list.some((tr) => tr.type === 'audio' && tr.selected && tr.id === pick.audioId)) void player.set('aid', pick.audioId);
    if (pick.subOff) void player.set('sid', 'no');
    else if (pick.subId) void player.set('sid', pick.subId);
  };

  // Remembers a manual choice for this series (or movie).
  const remember = (kind: 'audio' | 'sub', tr?: Track) => {
    const choice = tr ? { lang: tr.lang ?? '' } : { off: true };
    if (tr && !choice.lang) return;
    void tracksApi.remember(ctx.meta.id, kind, choice).catch(() => undefined);
  };

  onMount(() => {
    const off = player.on((e) => {
      if (closing && e.kind !== 'end') return;
      if (e.kind === 'prop') {
        switch (e.name) {
          case 'time-pos':
            if (typeof e.value === 'number') {
              setTime(e.value);
              if (!started() && e.value >= 0) {
                setStarted(true);
                setBuffering(false);
                showControls(false);
              }
            }
            break;
          case 'duration':
            if (typeof e.value === 'number') setDuration(e.value);
            break;
          case 'pause':
            setPaused(Boolean(e.value));
            if (e.value) showControls(false);
            break;
          case 'paused-for-cache':
            setBuffering(Boolean(e.value));
            break;
          case 'demuxer-cache-time':
            if (typeof e.value === 'number') setCacheTime(e.value);
            break;
          case 'track-list':
            setTracks(Array.isArray(e.value) ? e.value : []);
            break;
          case 'speed':
            if (typeof e.value === 'number') setSpeed(e.value);
            break;
        }
      } else if (e.kind === 'loaded') {
        void addExternalSubtitles();
      } else if (e.kind === 'end') {
        const reason = e.value?.reason;
        if (reason === 'error') {
          setError(e.value?.error || t('player.failed'));
          setBuffering(false);
        } else if (reason === 'eof' && !closing) {
          ended = true;
          setTime(duration());
          save();
          if (nextVideo() && settings().autoplayNext) startCountdown();
          else if (nextVideo()) showControls();
          else exit();
        }
      }
    });
    onCleanup(off);

    player.load(ctx.request).catch((err) => {
      setError(errorText(err));
      setBuffering(false);
    });

    saveTimer = window.setInterval(() => !paused() && save(), 15000);

    const hash = ctx.request.infoHash;
    let statusTimer: number | undefined;
    if (hash) {
      statusTimer = window.setInterval(async () => {
        if (started() && !buffering()) return;
        setTorrent(await player.torrentStatus(hash).catch(() => null));
      }, 1000);
    }

    const onKey = (e: KeyboardEvent) => {
      if (!props.active || menu()) return;
      const k = e.key;
      if (k === 'MediaPlayPause' || (k === ' ' && !visible())) {
        e.preventDefault();
        togglePause();
        return;
      }
      if (k === 'MediaPlay') return void player.set('pause', false);
      if (k === 'MediaPause') return void player.set('pause', true);
      if (k === 'MediaTrackNext' && nextVideo()) return void playNext();
      if (k === 'MediaFastForward') return seek(30);
      if (k === 'MediaRewind') return seek(-30);
      const onSeekBar = document.activeElement?.classList.contains('seek');
      if ((!visible() || onSeekBar) && (k === 'ArrowLeft' || k === 'ArrowRight')) {
        e.preventDefault();
        e.stopImmediatePropagation();
        seek(k === 'ArrowLeft' ? -seekStep : seekStep);
        if (onSeekBar) showControls(false);
        return;
      }
      // Menu toggles the controls without touching playback.
      if (menuKeys.has(k)) {
        e.preventDefault();
        e.stopImmediatePropagation();
        toggleFromMenu();
        return;
      }
      // OK on a bare video pauses and shows the controls together.
      if (!visible() && k === 'Enter') {
        e.preventDefault();
        e.stopImmediatePropagation();
        if (started() && !paused()) void player.set('pause', true);
        showControls();
        return;
      }
      if (!visible() && (k === 'ArrowUp' || k === 'ArrowDown')) {
        e.preventDefault();
        e.stopImmediatePropagation();
        showControls();
        return;
      }
      if (visible()) showControls(false);
    };
    addEventListener('keydown', onKey, true);
    // Some remotes only produce the context-menu event for their Menu key.
    const onContextMenu = (e: MouseEvent) => {
      e.preventDefault();
      if (props.active && !menu()) toggleFromMenu();
    };
    addEventListener('contextmenu', onContextMenu, true);
    // WebKit and pointer devices can emit stationary or tiny mouse moves during playback.
    const onMouse = (e: MouseEvent) => {
      if (!props.active) return;
      const current = { x: e.clientX, y: e.clientY };
      if (mousePosition?.x === current.x && mousePosition.y === current.y) return;
      mousePosition = current;
      if (visible()) {
        showControls(false);
        return;
      }
      const now = performance.now();
      if (!mouseMoveStart || now - mouseMoveStart.at > mouseRevealWindow) {
        mouseMoveStart = { ...current, at: now };
        return;
      }
      if (Math.hypot(current.x - mouseMoveStart.x, current.y - mouseMoveStart.y) >= mouseRevealDistance) {
        mouseMoveStart = undefined;
        showControls(false);
      }
    };
    addEventListener('mousemove', onMouse);

    const offBack = onBack(() => {
      if (!props.active) return false;
      if (countdown() !== undefined) {
        setCountdown(undefined);
        return true;
      }
      if (visible() && started() && !paused() && !error()) {
        hideControls();
        return true;
      }
      exit();
      return true;
    });

    onCleanup(() => {
      offBack();
      removeEventListener('keydown', onKey, true);
      removeEventListener('contextmenu', onContextMenu, true);
      removeEventListener('mousemove', onMouse);
      clearInterval(saveTimer);
      clearInterval(statusTimer);
      clearTimeout(hideTimer);
      clearTimeout(osdTimer);
      clearInterval(countTimer);
      document.body.classList.remove('mouse-hidden');
      if (!closing) {
        save();
        void player.stop();
      }
    });
    showControls();
  });

  let countTimer: number | undefined;
  const startCountdown = () => {
    setCountdown(10);
    showControls(false);
    countTimer = window.setInterval(() => {
      const n = (countdown() ?? 0) - 1;
      if (countdown() === undefined) return clearInterval(countTimer);
      if (n <= 0) {
        clearInterval(countTimer);
        void playNext();
      } else setCountdown(n);
    }, 1000);
    queueMicrotask(() => {
      const btn = document.querySelector<HTMLElement>('.next-card .btn');
      if (btn) focusEl(btn);
    });
  };

  const audioTracks = () => tracks().filter((tr) => tr.type === 'audio');
  const subTracks = () => tracks().filter((tr) => tr.type === 'sub');
  const fraction = () => (duration() > 0 ? time() / duration() : 0);
  const cached = () => (duration() > 0 ? Math.min(1, cacheTime() / duration()) : 0);
  const subtitle = () => (ctx.video ? `${episodeLabel(ctx.video)} · ${ctx.video.name ?? ctx.video.title ?? ''}` : '');

  const resumeFrom = (newCtx: PlayContext) => {
    closing = true;
    save();
    replaceTop({ name: 'player', play: newCtx });
  };

  return (
    <div class="screen player" onClick={() => showControls(false)}>
      <Show when={!started() && ctx.meta.background}>
        <div class="loading-backdrop" style={{ 'background-image': `url("${ctx.meta.background}")` }} />
      </Show>
      <div class="scrim" classList={{ 'hidden-ui': !visible() }} />

      <Show when={(buffering() || !started()) && !error()}>
        <div class="center">
          <Spinner />
          <p>{t('player.loading')}</p>
          <Show when={torrent()}>
            {(st) => <p>{t('player.peers', { p: st().peers, s: formatBytes(st().downloadSpeed) })}</p>}
          </Show>
        </div>
      </Show>

      <Show when={error()}>
        <div class="center" style={{ 'pointer-events': 'auto' }}>
          <p class="error" style={{ 'font-size': '1.3rem' }}>
            {t('player.failed')}
          </p>
          <p>{error()}</p>
          <div class="btn-row">
            <button class="btn" data-nav data-nav-default ref={(el) => queueMicrotask(() => focusEl(el))} onClick={() => setMenu('sources')}>
              <Layers />
              {t('details.sources')}
            </button>
            <button class="btn" data-nav onClick={exit}>
              {t('common.back')}
            </button>
          </div>
        </div>
      </Show>

      <Show when={osd()}>
        <div class="osd">{osd()}</div>
      </Show>

      <div class="top" classList={{ 'hidden-ui': !visible() }}>
        <h2>{ctx.meta.name}</h2>
        <Show when={subtitle()}>
          <p>{subtitle()}</p>
        </Show>
      </div>

      <Show when={countdown() !== undefined && nextVideo()}>
        <div class="next-card">
          <button class="btn primary" data-nav onClick={() => void playNext()}>
            <SkipForward />
            {t('player.nextIn', { n: countdown()! })}
          </button>
        </div>
      </Show>

      <Show when={!error()}>
        <div class="controls" classList={{ 'hidden-ui': !visible() }} inert={!visible()}>
          <div class="times">
            <span>{formatTime(time())}</span>
            <span>-{formatTime(duration() - time())}</span>
          </div>
          <button class="seek" data-nav aria-label="Seek" onClick={togglePause}>
            <div class="bar">
              <div class="cache" style={{ width: `${cached() * 100}%` }} />
              <div class="done" style={{ width: `${fraction() * 100}%` }} />
              <div class="knob" style={{ left: `${fraction() * 100}%` }} />
            </div>
          </button>
          <div class="buttons" data-nav-group>
            <button class="btn icon" data-nav data-nav-default onClick={togglePause} aria-label={paused() ? 'Play' : 'Pause'}>
              {paused() ? <Play /> : <Pause />}
            </button>
            <button class="btn icon" data-nav onClick={() => seek(-seekStep)} aria-label="-10s">
              <RotateCcw />
            </button>
            <button class="btn icon" data-nav onClick={() => seek(seekStep)} aria-label="+10s">
              <RotateCw />
            </button>
            <Show when={nextVideo()}>
              <button class="btn" data-nav onClick={() => void playNext()}>
                <SkipForward />
                {t('player.next')}
              </button>
            </Show>
            <div class="spacer" />
            <Show when={audioTracks().length > 1}>
              <button class="btn" data-nav onClick={() => setMenu('audio')}>
                <AudioLines />
                {t('player.audio')}
              </button>
            </Show>
            <button class="btn" data-nav onClick={() => setMenu('sub')}>
              <Subtitles />
              {t('player.subtitles')}
            </button>
            <button class="btn" data-nav onClick={() => setMenu('speed')}>
              <Gauge />
              {speed() === 1 ? t('player.speed') : `${speed()}×`}
            </button>
            <button class="btn" data-nav onClick={() => setMenu('sources')}>
              <Layers />
              {t('details.sources')}
            </button>
          </div>
        </div>
      </Show>

      <Show when={menu() === 'audio'}>
        <Panel title={t('player.audio')} onClose={() => setMenu(undefined)}>
          <For each={audioTracks()}>
            {(tr) => (
              <button
                class="option"
                classList={{ selected: tr.selected }}
                data-nav
                data-nav-default={tr.selected ? '' : undefined}
                onClick={() => {
                  void player.set('aid', tr.id);
                  remember('audio', tr);
                  setMenu(undefined);
                }}
              >
                <div class="main">
                  <div class="title">{trackLabel(tr)}</div>
                </div>
              </button>
            )}
          </For>
        </Panel>
      </Show>

      <Show when={menu() === 'sub'}>
        <Panel title={t('player.subtitles')} onClose={() => setMenu(undefined)}>
          <button
            class="option"
            classList={{ selected: !subTracks().some((s) => s.selected) }}
            data-nav
            data-nav-default={!subTracks().some((s) => s.selected) ? '' : undefined}
            onClick={() => {
              void player.set('sid', 'no');
              remember('sub');
              setMenu(undefined);
            }}
          >
            <div class="main">
              <div class="title">{t('player.off')}</div>
            </div>
          </button>
          <For each={subTracks()}>
            {(tr) => (
              <button
                class="option"
                classList={{ selected: tr.selected }}
                data-nav
                data-nav-default={tr.selected ? '' : undefined}
                onClick={() => {
                  void player.set('sid', tr.id);
                  remember('sub', tr);
                  setMenu(undefined);
                }}
              >
                <div class="main">
                  <div class="title">{trackLabel(tr)}</div>
                  <Show when={tr.external}>
                    <div class="sub">Addon</div>
                  </Show>
                </div>
              </button>
            )}
          </For>
        </Panel>
      </Show>

      <Show when={menu() === 'speed'}>
        <Panel title={t('player.speed')} onClose={() => setMenu(undefined)}>
          <For each={[0.5, 0.75, 1, 1.25, 1.5, 2]}>
            {(v) => (
              <button
                class="option"
                classList={{ selected: speed() === v }}
                data-nav
                data-nav-default={speed() === v ? '' : undefined}
                onClick={() => {
                  void player.set('speed', v);
                  setMenu(undefined);
                }}
              >
                <div class="main">
                  <div class="title">{v}×</div>
                </div>
              </button>
            )}
          </For>
        </Panel>
      </Show>

      <Show when={menu() === 'sources'}>
        <StreamPicker
          type={ctx.meta.type}
          videoId={ctx.video?.id ?? ctx.meta.id}
          title={ctx.meta.name}
          subtitle={subtitle() || ctx.meta.name}
          selectedKey={ctx.streamKey}
          onClose={() => setMenu(undefined)}
          onPick={(s, g) => {
            setMenu(undefined);
            resumeFrom({
              ...ctx,
              request: { ...toRequest(s), start: ended ? 0 : time(), title: ctx.request.title },
              streamKey: streamKey(s),
              bingeGroup: s.behaviorHints?.bingeGroup,
              addonId: g.addonId,
            });
          }}
        />
      </Show>
    </div>
  );
}
