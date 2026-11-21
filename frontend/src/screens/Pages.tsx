import { createEffect, createResource, createSignal, For, on, onCleanup, onMount, Show } from 'solid-js';
import { ArrowDown, ArrowUp, Delete, LogIn, LogOut, Maximize, Plus, Power, RefreshCw, Trash2, UserRound } from 'lucide-solid';
import { Window, Application } from '@wailsio/runtime';
import { account, addons, addonsVersion, content, profiles, type Addon, type MetaPreview } from '../api';
import { openProfilePicker } from './Profiles';
import { Card, errorText, Spinner, toast } from '../components';
import { lang, languageNames, languages, t, typeName } from '../i18n';
import { focusEl, focusFirst } from '../nav';
import { push } from '../router';
import { followsGtk, gtkTheme, settings, updateSettings } from '../settings';

/* Search ---------------------------------------------------------------- */

const keys = 'abcdefghijklmnopqrstuvwxyz1234567890'.split('');

export function Search() {
  const [query, setQuery] = createSignal('');
  const [debounced, setDebounced] = createSignal('');
  let input!: HTMLInputElement;
  let timer: number | undefined;
  createEffect(
    on(query, (q) => {
      clearTimeout(timer);
      timer = window.setTimeout(() => setDebounced(q.trim()), 450);
    }),
  );
  const [results] = createResource(debounced, (q) => (q.length >= 2 ? content.search(q) : Promise.resolve([])));
  const type = (k: string) => setQuery(query() + k);
  onMount(() => queueMicrotask(() => focusFirst(document.querySelector('.keyboard') ?? document)));
  // A physical keyboard types into the query wherever focus is.
  const onKey = (e: KeyboardEvent) => {
    if (document.activeElement === input || e.ctrlKey || e.altKey || e.metaKey) return;
    if (e.key.length === 1 && e.key !== ' ') {
      e.preventDefault();
      type(e.key.toLowerCase());
    } else if (e.key === ' ' && query()) {
      e.preventDefault();
      type(' ');
    }
  };
  addEventListener('keydown', onKey);
  onCleanup(() => removeEventListener('keydown', onKey));
  return (
    <div class="page">
      <div class="search-layout">
        <div data-nav-group>
          <input
            ref={input}
            class="field"
            type="search"
            data-nav
            placeholder={t('search.placeholder')}
            value={query()}
            onInput={(e) => setQuery(e.currentTarget.value)}
            onKeyDown={(e) => e.key === 'Enter' && setDebounced(query().trim())}
          />
          <div class="keyboard">
            <For each={keys}>
              {(k, i) => (
                <button class="key" data-nav data-nav-default={i() === 0 ? '' : undefined} onClick={() => type(k)}>
                  {k}
                </button>
              )}
            </For>
            <button class="key wide" data-nav onClick={() => type(' ')}>
              {t('kb.space')}
            </button>
            <button class="key wide" data-nav onClick={() => setQuery(query().slice(0, -1))} aria-label="Backspace">
              <Delete />
            </button>
            <button class="key wide" data-nav onClick={() => setQuery('')}>
              {t('kb.clear')}
            </button>
          </div>
          <p class="notice">{t('search.hint')}</p>
        </div>
        <div class="search-results">
          <Show when={!results.loading} fallback={<Spinner />}>
            <Show when={debounced().length >= 2 && results()?.length === 0}>
              <p class="notice">{t('search.empty', { q: debounced() })}</p>
            </Show>
            <For each={results()}>
              {(row) => (
                <section class="row" data-nav-block>
                  <h2>
                    {row.name} · {typeName(row.type)}
                    <small>{row.addonName}</small>
                  </h2>
                  <div class="row-track" data-nav-row data-nav-group>
                    <For each={row.items}>
                      {(item) => <Card item={item} onSelect={() => push({ name: 'details', type: item.type, id: item.id, preview: item })} />}
                    </For>
                  </div>
                </section>
              )}
            </For>
          </Show>
        </div>
      </div>
    </div>
  );
}

/* Grid (view all) --------------------------------------------------------- */

export function Grid(props: { addonId: string; type: string; catalogId: string; title: string; genres?: string[] }) {
  const [genre, setGenre] = createSignal('');
  const [items, setItems] = createSignal<MetaPreview[]>([]);
  const [loading, setLoading] = createSignal(false);
  const [done, setDone] = createSignal(false);
  let generation = 0;

  const load = async (reset: boolean) => {
    if (loading() && !reset) return;
    const gen = reset ? ++generation : generation;
    setLoading(true);
    const extra: Record<string, string> = {};
    if (genre()) extra.genre = genre();
    const skip = reset ? 0 : items().length;
    if (skip) extra.skip = String(skip);
    try {
      const page = await content.catalog(props.addonId, props.type, props.catalogId, extra);
      if (gen !== generation) return;
      const seen = new Set(reset ? [] : items().map((i) => i.id));
      const fresh = page.filter((i) => !seen.has(i.id));
      setItems(reset ? fresh : [...items(), ...fresh]);
      setDone(fresh.length === 0);
    } catch (err) {
      toast(errorText(err));
      setDone(true);
    } finally {
      if (gen === generation) setLoading(false);
    }
  };

  createEffect(on(genre, () => void load(true)));
  let focusedOnce = false;
  createEffect(() => {
    if (!focusedOnce && items().length) {
      focusedOnce = true;
      queueMicrotask(() => focusFirst(document.querySelector('.grid') ?? document));
    }
  });

  const onCardFocus = (i: number) => {
    if (!done() && i >= items().length - 12) void load(false);
  };

  return (
    <div class="page" data-nav-pane data-nav-top="0.25">
      <h1>{props.title}</h1>
      <Show when={props.genres?.length}>
        <div class="seasons" style={{ padding: '0 0 1rem' }} data-nav-row data-nav-group>
          <button class="chip" classList={{ selected: genre() === '' }} data-nav onClick={() => setGenre('')}>
            {t('common.all')}
          </button>
          <For each={props.genres}>
            {(g) => (
              <button class="chip" classList={{ selected: genre() === g }} data-nav onClick={() => setGenre(g)}>
                {g}
              </button>
            )}
          </For>
        </div>
      </Show>
      <div class="grid">
        <For each={items()}>
          {(item, i) => (
            <Card
              item={item}
              default={i() === 0}
              onFocus={() => onCardFocus(i())}
              onSelect={() => push({ name: 'details', type: item.type, id: item.id, preview: item })}
            />
          )}
        </For>
      </div>
      <Show when={loading()}>
        <Spinner />
      </Show>
    </div>
  );
}

/* Addons ------------------------------------------------------------------ */

export function Addons() {
  const [list, { refetch, mutate }] = createResource(() => addonsVersion() + 1, () => addons.list());
  const [managed, { refetch: refetchManaged }] = createResource(() => addonsVersion() + 1, () => addons.managed());
  const sync = async (el: HTMLElement) => {
    try {
      await account.sync();
      toast(t('account.synced'));
      refetchManaged();
      if (el.isConnected) focusEl(el);
    } catch (err) {
      toast(errorText(err));
    }
  };
  const [url, setUrl] = createSignal('');
  const [busy, setBusy] = createSignal(false);

  const run = async (fn: () => Promise<unknown>, keepFocus?: HTMLElement) => {
    try {
      await fn();
      const next = await addons.list();
      mutate(next);
      if (keepFocus?.isConnected) focusEl(keepFocus);
    } catch (err) {
      toast(errorText(err));
    }
  };

  const install = async () => {
    if (!url().trim() || busy()) return;
    setBusy(true);
    try {
      const ad = await addons.install(url());
      toast(t('addons.installed', { name: ad.manifest.name }));
      setUrl('');
      refetch();
    } catch (err) {
      toast(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div class="page" data-nav-pane>
      <h1>{t('addons.title')}</h1>
      <Show when={managed()}>
        <p class="notice" style={{ 'margin-bottom': '1rem' }}>{t('account.managed')}</p>
        <div class="btn-row" style={{ 'margin-bottom': '1.6rem' }} data-nav-block>
          <button class="btn primary" data-nav data-nav-default onClick={(e) => void sync(e.currentTarget)}>
            <RefreshCw />
            {t('account.syncNow')}
          </button>
        </div>
      </Show>
      <Show when={managed() === false}>
      <div class="btn-row" style={{ 'margin-bottom': '1.6rem', 'flex-wrap': 'nowrap' }} data-nav-block>
        <input
          class="field"
          type="url"
          data-nav
          data-nav-default
          placeholder={t('addons.urlPlaceholder')}
          value={url()}
          onInput={(e) => setUrl(e.currentTarget.value)}
          onKeyDown={(e) => e.key === 'Enter' && install()}
        />
        <button class="btn primary" data-nav disabled={busy()} onClick={install}>
          {busy() ? <Spinner /> : <Plus />}
          {t('addons.install')}
        </button>
        <button class="btn" data-nav onClick={(e) => run(() => addons.refresh(), e.currentTarget)}>
          <RefreshCw />
          {t('addons.refresh')}
        </button>
      </div>
      </Show>
      <For each={list()}>
        {(ad: Addon, i) => (
          <div class="addon" classList={{ disabled: !ad.enabled }} data-nav-block>
            <Show when={ad.manifest.logo} fallback={<div class="logo-fallback" />}>
              <img src={ad.manifest.logo} alt="" />
            </Show>
            <div>
              <h3>
                {ad.manifest.name}
                <small>
                  v{ad.manifest.version} · {ad.manifest.types?.map(typeName).join(', ')}
                  {ad.enabled ? '' : ` · ${t('addons.disabled')}`}
                </small>
              </h3>
              <p>{ad.manifest.description}</p>
            </div>
            <div class="btn-row" data-nav-group>
              <button class="btn small" data-nav onClick={(e) => run(() => addons.setEnabled(ad.manifest.id, !ad.enabled), e.currentTarget)}>
                {ad.enabled ? t('addons.disable') : t('addons.enable')}
              </button>
              <Show when={!managed()}>
              <button class="btn small icon" data-nav disabled={i() === 0} aria-label={t('addons.up')} onClick={(e) => run(() => addons.move(ad.manifest.id, -1), e.currentTarget)}>
                <ArrowUp />
              </button>
              <button
                class="btn small icon"
                data-nav
                disabled={i() === (list()?.length ?? 0) - 1}
                aria-label={t('addons.down')}
                onClick={(e) => run(() => addons.move(ad.manifest.id, 1), e.currentTarget)}
              >
                <ArrowDown />
              </button>
              <button class="btn small icon" data-nav aria-label={t('addons.remove')} onClick={() => run(() => addons.remove(ad.manifest.id))}>
                <Trash2 />
              </button>
              </Show>
            </div>
          </div>
        )}
      </For>
    </div>
  );
}

/* Settings ---------------------------------------------------------------- */

const namedAccents = ['blue', 'mauve', 'lavender', 'green', 'orange', 'red'];
const themes = ['gtk', 'lounge'];
const mediaLangs = ['', 'por', 'eng', 'spa'];
const subSizes = [0.8, 1, 1.2, 1.4, 1.6];

function cycle<T>(list: T[], current: T): T {
  return list[(list.indexOf(current) + 1) % list.length];
}

function langLabel(code: string) {
  if (!code) return t('settings.none');
  return { por: 'Português', eng: 'English', spa: 'Español' }[code] ?? code;
}

export function SettingsPage(props: { active: boolean }) {
  onMount(() => queueMicrotask(() => focusFirst(document.querySelector('.settings-list') ?? document)));
  const s = settings;
  // The theme's own accent is offered only while following the GTK theme.
  const accents = () => (followsGtk() ? ['theme', ...namedAccents] : namedAccents);
  const accent = () => (accents().includes(s().accent) ? s().accent : 'blue');
  const [status, { refetch, mutate }] = createResource(() => props.active, () => account.status());
  const [activeProfile] = createResource(() => props.active, () => profiles.active());
  const syncedLabel = () => {
    const at = status()?.syncedAt;
    return at ? new Date(at).toLocaleString(lang()) : t('account.never');
  };
  const syncNow = async () => {
    try {
      mutate(await account.sync());
      toast(t('account.synced'));
    } catch (err) {
      toast(errorText(err));
      refetch();
    }
  };
  const signOut = async () => {
    try {
      await account.logout();
    } catch (err) {
      toast(errorText(err));
    }
    refetch();
  };
  const langPref = ['', ...languages];
  const item = (label: () => string, value: () => string, onClick: () => void, first = false) => (
    <button class="setting" data-nav data-nav-default={first ? '' : undefined} onClick={onClick}>
      <span>{label()}</span>
      <span>{value()}</span>
    </button>
  );
  return (
    <div class="page" data-nav-pane>
      <h1>{t('settings.title')}</h1>
      <div class="settings-list">
        <Show
          when={status()?.signedIn}
          fallback={
            <button class="setting" data-nav data-nav-default onClick={() => push({ name: 'login' })}>
              <span>{t('account.title')}</span>
              <span style={{ display: 'flex', gap: '0.6rem', 'align-items': 'center' }}>
                <LogIn />
                {t('account.signIn')}
              </span>
            </button>
          }
        >
          <div class="setting" style={{ cursor: 'default' }}>
            <span style={{ display: 'flex', gap: '0.8rem', 'align-items': 'center' }}>
              <UserRound />
              {t('account.signedInAs', { email: status()!.email || 'Stremio' })}
            </span>
            <span style={{ color: status()?.error ? 'var(--plz-red)' : undefined, 'font-weight': 500, 'font-size': '0.9rem' }}>
              {status()?.error || t('account.syncedAt', { time: syncedLabel() })}
            </span>
          </div>
          <button class="setting" data-nav data-nav-default onClick={() => void syncNow()}>
            <span>{t('account.syncNow')}</span>
            <RefreshCw />
          </button>
          <button class="setting" data-nav onClick={() => void signOut()}>
            <span>{t('account.signOut')}</span>
            <LogOut />
          </button>
        </Show>
        {item(
          () => t('profiles.title'),
          () => activeProfile()?.name ?? '',
          () => push({ name: 'profiles' }),
        )}
        {item(
          () => t('profiles.switch'),
          () => '',
          () => openProfilePicker(),
        )}
        {item(
          () => t('settings.language'),
          () => (s().language ? languageNames[s().language] : t('settings.system')),
          () => void updateSettings({ language: cycle(langPref, s().language) }),
        )}
        {item(
          () => t('settings.theme'),
          () => ((s().theme || 'gtk') === 'gtk' ? t('settings.themeGtk', { name: gtkTheme()?.name || 'GTK' }) : 'Lounge'),
          () => {
            const theme = cycle(themes, s().theme || 'gtk');
            // Following the desktop also takes its accent.
            void updateSettings({ theme, accent: theme === 'gtk' ? 'theme' : s().accent === 'theme' ? 'blue' : s().accent });
          },
        )}
        {item(
          () => t('settings.accent'),
          () => t(`accent.${accent()}` as Parameters<typeof t>[0]),
          () => void updateSettings({ accent: cycle(accents(), accent()) }),
        )}
        {item(
          () => t('settings.autoplay'),
          () => (s().autoplayNext ? t('settings.on') : t('settings.off')),
          () => void updateSettings({ autoplayNext: !s().autoplayNext }),
        )}
        {item(
          () => t('settings.audioLang'),
          () => langLabel(s().audioLanguages[0] ?? ''),
          () => {
            const next = cycle(mediaLangs, s().audioLanguages[0] ?? '');
            void updateSettings({ audioLanguages: next ? [next] : [] });
          },
        )}
        {item(
          () => t('settings.subLang'),
          () => langLabel(s().subtitleLanguages[0] ?? ''),
          () => {
            const next = cycle(mediaLangs, s().subtitleLanguages[0] ?? '');
            void updateSettings({ subtitleLanguages: next ? [next] : [] });
          },
        )}
        {item(
          () => t('settings.subSize'),
          () => `${Math.round((s().subtitleScale || 1) * 100)}%`,
          () => void updateSettings({ subtitleScale: cycle(subSizes, s().subtitleScale || 1) }),
        )}
        <button class="setting" data-nav onClick={() => void Window.ToggleFullscreen()}>
          <span>{t('settings.fullscreen')}</span>
          <Maximize />
        </button>
        <button class="setting" data-nav onClick={() => void Application.Quit()}>
          <span>{t('settings.quit')}</span>
          <Power />
        </button>
      </div>
    </div>
  );
}
