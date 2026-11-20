import { createSignal } from 'solid-js';
import { Call, Events } from '@wailsio/runtime';

const call = <T>(method: string, ...args: unknown[]) => Call.ByName(`main.${method}`, ...args) as Promise<T>;

export type MetaPreview = {
  id: string;
  type: string;
  name: string;
  poster?: string;
  posterShape?: 'poster' | 'landscape' | 'square';
  background?: string;
  logo?: string;
  description?: string;
  releaseInfo?: string;
  imdbRating?: string;
  genres?: string[];
  runtime?: string;
};

export type Video = {
  id: string;
  title?: string;
  name?: string;
  season?: number;
  episode?: number;
  number?: number;
  released?: string;
  thumbnail?: string;
  overview?: string;
};

export type Meta = MetaPreview & {
  videos?: Video[];
  cast?: string[];
  director?: string[];
  released?: string;
  links?: { name: string; category: string; url: string }[];
};

export type Subtitle = { id: string; url: string; lang: string };

export type Stream = {
  url?: string;
  ytId?: string;
  infoHash?: string;
  fileIdx?: number;
  externalUrl?: string;
  name?: string;
  title?: string;
  description?: string;
  sources?: string[];
  subtitles?: Subtitle[];
  behaviorHints?: {
    notWebReady?: boolean;
    bingeGroup?: string;
    filename?: string;
    videoSize?: number;
    proxyHeaders?: { request?: Record<string, string> };
  };
};

export type StreamGroup = { addonId: string; addonName: string; streams: Stream[] | null; error?: string };

export type Row = { addonId: string; addonName: string; type: string; catalogId: string; name: string; genres?: string[] };
export type SearchRow = Row & { items: MetaPreview[]; error?: string };

export type Manifest = {
  id: string;
  version: string;
  name: string;
  description?: string;
  logo?: string;
  types: string[];
  catalogs: { type: string; id: string; name: string }[];
  resources: { name: string }[];
};
export type Addon = { transportUrl: string; manifest: Manifest; enabled: boolean };

export type Entry = {
  id: string;
  type: string;
  name: string;
  poster?: string;
  background?: string;
  logo?: string;
  videoId: string;
  season?: number;
  episode?: number;
  episodeTitle?: string;
  position: number;
  duration: number;
  updatedAt?: number;
  streamKey?: string;
};

export type Settings = {
  language: string;
  audioLanguages: string[];
  subtitleLanguages: string[];
  subtitleScale: number;
  autoplayNext: boolean;
  /** theme is 'gtk' (follow the desktop) or 'lounge'; empty means 'gtk'. */
  theme: string;
  accent: string;
};

/** GtkTheme is the desktop's GTK theme; colors are CSS values, empty when undefined. */
export type GtkTheme = {
  name: string;
  dark: boolean;
  background: string;
  surface: string;
  text: string;
  accent: string;
  onAccent: string;
  border: string;
  subtle: string;
  error: string;
  success: string;
  warning: string;
};

export const themeApi = {
  gtk: () => Call.ByName('main.ThemeService.GTK') as Promise<GtkTheme>,
  /** onChange runs when the desktop's GTK theme changes. */
  onChange: (fn: () => void) => Events.On('theme', fn),
};

export type PlayRequest = {
  url?: string;
  infoHash?: string;
  fileIdx?: number;
  filename?: string;
  sources?: string[];
  headers?: Record<string, string>;
  start?: number;
  title?: string;
  subtitles?: Subtitle[];
};

export type TorrentStatus = {
  infoHash: string;
  name: string;
  peers: number;
  seeders: number;
  downloadSpeed: number;
  downloaded: number;
  size: number;
};

export type PlayerEvent = { kind: 'prop' | 'loaded' | 'end'; name?: string; value?: any };

export const content = {
  rows: (type = '') => call<Row[]>('ContentService.Rows', type),
  catalog: (addonId: string, type: string, catalogId: string, extra: Record<string, string> = {}) =>
    call<MetaPreview[] | null>('ContentService.Catalog', addonId, type, catalogId, extra).then((r) => r ?? []),
  search: (query: string) => call<SearchRow[] | null>('ContentService.Search', query).then((r) => r ?? []),
  meta: (type: string, id: string) => call<Meta>('ContentService.Meta', type, id),
  streams: (type: string, id: string) => call<StreamGroup[] | null>('ContentService.Streams', type, id).then((r) => r ?? []),
  subtitles: (type: string, id: string) => call<Subtitle[] | null>('ContentService.Subtitles', type, id).then((r) => r ?? []),
};

// addonsVersion changes whenever the addon list is replaced by an account sync.
const [addonsVersion, setAddonsVersion] = createSignal(0);
export { addonsVersion };
Events.On('addons', () => setAddonsVersion((v) => v + 1));

export const addons = {
  list: () => call<Addon[] | null>('AddonService.List').then((r) => r ?? []),
  managed: () => call<boolean>('AddonService.Managed'),
  install: (url: string) => call<Addon>('AddonService.Install', url),
  remove: (id: string) => call<void>('AddonService.Remove', id),
  setEnabled: (id: string, enabled: boolean) => call<void>('AddonService.SetEnabled', id, enabled),
  move: (id: string, delta: number) => call<void>('AddonService.Move', id, delta),
  refresh: () => call<void>('AddonService.Refresh'),
};

export const library = {
  continueWatching: () => call<Entry[] | null>('LibraryService.ContinueWatching').then((r) => r ?? []),
  get: (id: string) => call<Entry | null>('LibraryService.Get', id),
  save: (e: Entry) => call<void>('LibraryService.Save', e),
  remove: (id: string) => call<void>('LibraryService.Remove', id),
  setWatched: (videoId: string, watched: boolean) => call<void>('LibraryService.SetWatched', videoId, watched),
  watched: (ids: string[]) => call<string[] | null>('LibraryService.Watched', ids).then((r) => r ?? []),
};

export const settingsApi = {
  get: () => call<Settings>('SettingsService.Get'),
  set: (s: Settings) => call<void>('SettingsService.Set', s),
};

export const player = {
  load: (req: PlayRequest) => call<void>('PlayerService.Load', req),
  command: (...args: (string | number)[]) => call<void>('PlayerService.Command', args.map(String)),
  set: (name: string, value: string | number | boolean) =>
    call<void>('PlayerService.SetProperty', name, typeof value === 'boolean' ? (value ? 'yes' : 'no') : String(value)),
  get: async <T>(name: string) => JSON.parse(await call<string>('PlayerService.Property', name)) as T,
  stop: () => call<void>('PlayerService.Stop'),
  torrentStatus: (hash: string) => call<TorrentStatus | null>('PlayerService.TorrentStatus', hash),
  on: (fn: (e: PlayerEvent) => void) => Events.On('player', (ev: { data: PlayerEvent }) => fn(ev.data)),
};

export type AccountStatus = { signedIn: boolean; email: string; syncedAt: number; error?: string };
export type LinkCode = { code: string; link: string; qrcode: string };

export const account = {
  status: () => call<AccountStatus>('AccountService.Status'),
  login: (email: string, password: string) => call<AccountStatus>('AccountService.Login', email, password),
  createLink: () => call<LinkCode>('AccountService.CreateLink'),
  checkLink: (code: string) => call<AccountStatus>('AccountService.CheckLink', code),
  logout: () => call<void>('AccountService.Logout'),
  sync: () => call<AccountStatus>('AccountService.Sync'),
};
