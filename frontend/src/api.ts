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
  /** subtitleMode is 'on', 'off', or 'auto' (only when the audio is not in the preferred language). */
  subtitleMode?: string;
  subtitleScale: number;
  /** Presets understood by internal/subtitle; empty means the default. */
  subtitleColor?: string;
  subtitleBackground?: string;
  subtitlePosition?: string;
  autoplayNext: boolean;
  /** skipSegments is 'off', 'ask' (a button), or 'auto'. */
  skipSegments: string;
  /** theme is 'gtk' (follow the desktop) or 'lounge'; empty means 'gtk'. */
  theme: string;
  accent: string;
  /** cacheLimitGB caps the torrent cache; 0 means unlimited. */
  cacheLimitGB: number;
  clearCacheOnExit: boolean;
  /** audioOutput is 'stereo' (default), 'auto' or 'passthrough'. */
  audioOutput: string;
  nightMode: boolean;
  /** audioDevice is an mpv device name; empty is the system default. */
  audioDevice: string;
  /** hdr is 'tonemap' (default) or 'off'. */
  hdr: string;
  /** sourceMode is "ask" (also empty) or "auto"; see internal/sources. */
  sourceMode: string;
  /** sourceMaxHeight is 0 (any), 2160, 1080 or 720. */
  sourceMaxHeight: number;
  /** sourceMaxSizeGb is 0 (any) or a size cap in GB. */
  sourceMaxSizeGb: number;
  /** sourcePrefer is "direct", "torrent" or empty. */
  sourcePrefer: string;
  sourceMinSeeders: number;
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

/** TorrentStats is the live snapshot the backend emits every second. */
export type TorrentStats = {
  active: boolean;
  infoHash: string;
  name: string;
  phase: 'connecting' | 'metadata' | 'downloading';
  peers: number;
  totalPeers: number;
  seeders: number;
  downloadRate: number;
  uploadRate: number;
  downloaded: number;
  size: number;
  fileSize: number;
  progress: number;
  bufferedBytes: number;
  bufferedPercent: number;
  noSeeders: boolean;
};

export type PlayerEvent ={ kind: 'prop' | 'loaded' | 'end'; name?: string; value?: any };

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

export type PhoneSession = { url: string; qrcode: string };
export type PhoneAddEvent = { ok?: boolean; name?: string; error?: string; stopped?: boolean };

export const phoneAdd = {
  start: () => call<PhoneSession>('PhoneAddService.Start'),
  stop: () => call<void>('PhoneAddService.Stop'),
  on: (fn: (e: PhoneAddEvent) => void) => Events.On('phoneadd', (ev: { data: PhoneAddEvent }) => fn(ev.data)),
};

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

/** sourcesApi ranks sources under the user's rules (internal/sources). */
export const sourcesApi = {
  /** pick returns the best source, or null when automatic selection is off or nothing passes. */
  pick: (groups: StreamGroup[]) =>
    call<{ group: number; stream: number } | null>('SourcesService.Pick', groups),
};

export const settingsApi = {
  get: () => call<Settings>('SettingsService.Get'),
  set: (s: Settings) => call<void>('SettingsService.Set', s),
};

export type TrackInfo = { id: number; type: string; lang: string; title: string; default: boolean; forced: boolean; external: boolean };
export type TrackChoice = { lang?: string; off?: boolean };
export type TrackResult = { audioId: number; subId: number; subOff: boolean };

export const tracksApi = {
  select: (titleId: string, list: TrackInfo[]) => call<TrackResult>('TracksService.Select', titleId, list),
  remember: (titleId: string, kind: 'audio' | 'sub', choice: TrackChoice) => call<void>('TracksService.Remember', titleId, kind, choice),
};

export const cacheApi = {
  /** size is the bytes of torrent data on disk. */
  size: () => call<number>('CacheService.Size'),
  /** clear deletes cached torrent data except the torrent playing; it returns the bytes freed. */
  clear: () => call<number>('CacheService.Clear'),
};

export const player = {
  load: (req: PlayRequest) => call<void>('PlayerService.Load', req),
  command: (...args: (string | number)[]) => call<void>('PlayerService.Command', args.map(String)),
  set: (name: string, value: string | number | boolean) =>
    call<void>('PlayerService.SetProperty', name, typeof value === 'boolean' ? (value ? 'yes' : 'no') : String(value)),
  get: async <T>(name: string) => JSON.parse(await call<string>('PlayerService.Property', name)) as T,
  subtitleStyle: (style: { scale: number; color: string; background: string; position: string }) =>
    call<void>('PlayerService.SetSubtitleStyle', style),
  stop: () => call<void>('PlayerService.Stop'),
  applyAV: (s: Settings) => call<void>('PlayerService.ApplyAV', s),
  audioDevices: () => call<{ name: string; description: string }[] | null>('PlayerService.AudioDevices').then((r) => r ?? []),
  torrentStatus: (hash: string) => call<TorrentStatus | null>('PlayerService.TorrentStatus', hash),
  keepWaiting: () => call<void>('PlayerService.KeepWaiting'),
  onTorrent: (fn: (s: TorrentStats) => void) => Events.On('torrent', (ev: { data: TorrentStats }) => fn(ev.data)),
  on: (fn: (e: PlayerEvent) => void) => Events.On('player', (ev: { data: PlayerEvent }) => fn(ev.data)),
};

export type Segment = { kind: 'intro' | 'recap' | 'credits'; start: number; end: number; source: string };
export type Chapter = { title: string; time: number };

export const segmentsApi = {
  /** resolve merges chapter-based segments with the online database; it never fails. */
  resolve: (chapters: Chapter[], duration: number, imdbId: string, season: number, episode: number) =>
    call<Segment[] | null>('SegmentService.Resolve', chapters, duration, imdbId, season, episode)
      .then((r) => r ?? [])
      .catch(() => [] as Segment[]),
};

export type AccountStatus = {
  signedIn: boolean;
  email: string;
  syncedAt: number;
  error?: string;
  librarySyncedAt: number;
  libraryPulled: number;
  libraryPushed: number;
  libraryError?: string;
};
export type LinkCode = { code: string; link: string; qrcode: string };

export const account = {
  status: () => call<AccountStatus>('AccountService.Status'),
  login: (email: string, password: string) => call<AccountStatus>('AccountService.Login', email, password),
  createLink: () => call<LinkCode>('AccountService.CreateLink'),
  checkLink: (code: string) => call<AccountStatus>('AccountService.CheckLink', code),
  logout: () => call<void>('AccountService.Logout'),
  sync: () => call<AccountStatus>('AccountService.Sync'),
};

export type Profile = { id: string; name: string; hue: number };

export const profiles = {
  list: () => call<Profile[]>('ProfileService.List'),
  active: () => call<Profile>('ProfileService.Active'),
  needsPicker: () => call<boolean>('ProfileService.NeedsPicker'),
  switch: (id: string) => call<void>('ProfileService.Switch', id),
  create: (name: string) => call<Profile>('ProfileService.Create', name),
  rename: (id: string, name: string) => call<void>('ProfileService.Rename', id, name),
  remove: (id: string) => call<void>('ProfileService.Delete', id),
};
