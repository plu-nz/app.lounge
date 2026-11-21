import { createResource, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { Check, Pencil, Plus, Trash2, Users, X } from 'lucide-solid';
import { player, profiles, type Profile } from '../api';
import { errorText, toast } from '../components';
import { t } from '../i18n';
import { focusEl, focusFirst, onBack } from '../nav';

/** pickerMode is 'launch' while the user must choose, 'switch' when opened by choice. */
const [pickerMode, setPickerMode] = createSignal<'launch' | 'switch' | null>(null);
export { pickerMode };

/** openProfilePicker is the quick "switch profile" action. */
export const openProfilePicker = () => setPickerMode('switch');

/** showPickerAtLaunch asks who is watching when several profiles exist. */
export async function showPickerAtLaunch() {
  try {
    if (await profiles.needsPicker()) setPickerMode('launch');
  } catch {
    // Without the answer the app starts on the last profile.
  }
}

/** switchProfile changes the active profile. Services reload in Go; the UI
 * reloads too, so no screen keeps data from the previous profile. */
export async function switchProfile(id: string) {
  try {
    await player.stop().catch(() => undefined);
    await profiles.switch(id);
    location.reload();
  } catch (err) {
    toast(errorText(err));
  }
}

export function Avatar(props: { profile: Profile; big?: boolean }) {
  return (
    <span
      class="avatar"
      classList={{ big: props.big }}
      style={{ '--avatar-hue': String(props.profile.hue) }}
      aria-hidden="true"
    >
      {[...props.profile.name][0]?.toUpperCase()}
    </span>
  );
}

/** ProfilePicker is the full-screen "who is watching" choice. */
export function ProfilePicker(props: { mode: 'launch' | 'switch' }) {
  const [list] = createResource(() => Promise.all([profiles.list(), profiles.active()]));
  let ref!: HTMLDivElement;
  const close = () => setPickerMode(null);
  onCleanup(
    onBack(() => {
      if (props.mode === 'switch') close();
      return true;
    }),
  );
  const previous = document.activeElement as HTMLElement | null;
  onCleanup(() => {
    if (previous?.isConnected) focusEl(previous);
  });
  const choose = async (p: Profile, active: Profile) => {
    if (p.id === active.id) {
      // Keeping the current profile needs no reload; only record the choice.
      await profiles.switch(p.id).catch(() => undefined);
      close();
      return;
    }
    await switchProfile(p.id);
  };
  const ready = () => list.state === 'ready';
  onMount(() => {
    const wait = setInterval(() => {
      if (ref.querySelector('[data-nav]')) {
        clearInterval(wait);
        focusFirst(ref);
      }
    }, 30);
    onCleanup(() => clearInterval(wait));
  });
  return (
    <div class="profile-picker" data-nav-scope ref={ref}>
      <h1>{t('profiles.who')}</h1>
      <Show when={ready() && list()}>
        {(data) => (
          <div class="profile-tiles" data-nav-group>
            <For each={data()[0]}>
              {(p) => (
                <button
                  class="profile-tile"
                  data-nav
                  data-nav-default={p.id === data()[1].id ? '' : undefined}
                  onClick={() => void choose(p, data()[1])}
                >
                  <Avatar profile={p} big />
                  <span class="profile-name">{p.name}</span>
                </button>
              )}
            </For>
          </div>
        )}
      </Show>
    </div>
  );
}

type Mode = { kind: 'rename' | 'delete'; id: string } | null;

/** ProfilesPage manages profiles from Settings. */
export function ProfilesPage() {
  const [list, { refetch }] = createResource(() => Promise.all([profiles.list(), profiles.active()]));
  const [mode, setMode] = createSignal<Mode>(null);
  const [name, setName] = createSignal('');
  const [newName, setNewName] = createSignal('');
  let page!: HTMLDivElement;

  onMount(() => queueMicrotask(() => focusFirst(page)));
  onCleanup(
    onBack(() => {
      if (!mode()) return false;
      setMode(null);
      return true;
    }),
  );

  const focusSoon = (selector: string) =>
    setTimeout(() => {
      const el = page.querySelector<HTMLElement>(selector);
      if (el) focusEl(el);
    }, 0);

  const run = async (fn: () => Promise<unknown>) => {
    try {
      await fn();
      return true;
    } catch (err) {
      toast(errorText(err));
      return false;
    } finally {
      refetch();
    }
  };
  const startRename = (p: Profile) => {
    setName(p.name);
    setMode({ kind: 'rename', id: p.id });
    focusSoon('.profile-edit input');
  };
  const saveRename = async () => {
    const m = mode();
    if (!m || !name().trim()) return;
    if (await run(() => profiles.rename(m.id, name()))) {
      setMode(null);
      focusSoon('.profile-row');
    }
  };
  const confirmDelete = async () => {
    const m = mode();
    if (!m) return;
    if (await run(() => profiles.remove(m.id))) {
      setMode(null);
      focusSoon('.profile-row .btn');
    }
  };
  const add = async () => {
    if (!newName().trim()) return;
    if (await run(() => profiles.create(newName()))) setNewName('');
  };

  return (
    <div class="page profiles-page" data-nav-pane ref={page}>
      <h1>{t('profiles.title')}</h1>
      <p class="notice">{t('profiles.help')}</p>
      <Show when={list()}>
        {(data) => (
          <>
            <div class="profile-list">
              <For each={data()[0]}>
                {(p) => {
                  const isActive = () => p.id === data()[1].id;
                  const last = () => data()[0].length === 1;
                  const m = () => (mode()?.id === p.id ? mode()!.kind : null);
                  return (
                    <div class="profile-row">
                      <Avatar profile={p} />
                      <Show
                        when={m() !== 'rename'}
                        fallback={
                          <div class="profile-edit">
                            <input
                              class="field"
                              data-nav
                              data-nav-default
                              value={name()}
                              placeholder={t('profiles.namePlaceholder')}
                              maxLength={24}
                              onInput={(e) => setName(e.currentTarget.value)}
                              onKeyDown={(e) => e.key === 'Enter' && void saveRename()}
                            />
                            <button class="btn small primary" data-nav onClick={() => void saveRename()}>
                              <Check />
                              {t('profiles.save')}
                            </button>
                            <button class="btn small" data-nav onClick={() => setMode(null)}>
                              <X />
                              {t('profiles.cancel')}
                            </button>
                          </div>
                        }
                      >
                        <span class="profile-name">
                          {p.name}
                          <Show when={isActive()}>
                            <span class="badge">{t('profiles.active')}</span>
                          </Show>
                        </span>
                        <Show
                          when={m() !== 'delete'}
                          fallback={
                            <div class="profile-edit">
                              <span class="error">{t('profiles.deleteConfirm', { name: p.name })}</span>
                              <button class="btn small" data-nav data-nav-default onClick={() => setMode(null)}>
                                <X />
                                {t('profiles.cancel')}
                              </button>
                              <button class="btn small primary" data-nav onClick={() => void confirmDelete()}>
                                <Trash2 />
                                {t('profiles.delete')}
                              </button>
                            </div>
                          }
                        >
                          <div class="profile-actions">
                            <Show when={!isActive()}>
                              <button class="btn small" data-nav onClick={() => void switchProfile(p.id)}>
                                <Users />
                                {t('profiles.switch')}
                              </button>
                            </Show>
                            <button class="btn small" data-nav onClick={() => startRename(p)}>
                              <Pencil />
                              {t('profiles.rename')}
                            </button>
                            <Show when={!isActive() && !last()}>
                              <button
                                class="btn small"
                                data-nav
                                onClick={() => {
                                  setMode({ kind: 'delete', id: p.id });
                                  focusSoon('.profile-edit .btn');
                                }}
                              >
                                <Trash2 />
                                {t('profiles.delete')}
                              </button>
                            </Show>
                          </div>
                        </Show>
                      </Show>
                    </div>
                  );
                }}
              </For>
            </div>
            <div class="profile-add">
              <input
                class="field"
                data-nav
                value={newName()}
                placeholder={t('profiles.namePlaceholder')}
                maxLength={24}
                onInput={(e) => setNewName(e.currentTarget.value)}
                onKeyDown={(e) => e.key === 'Enter' && void add()}
              />
              <button class="btn primary" data-nav onClick={() => void add()}>
                <Plus />
                {t('profiles.add')}
              </button>
            </div>
          </>
        )}
      </Show>
    </div>
  );
}
