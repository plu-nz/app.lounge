import { createEffect, createResource, createSignal, For, on, Show } from 'solid-js';
import { addonsVersion, content, library, type Entry, type MetaPreview, type Row } from '../api';
import { Backdrop, Card, MetaLine, MoreCard } from '../components';
import { t, typeName } from '../i18n';
import { focusFirst } from '../nav';
import { push } from '../router';

const rowLimit = 24;

export function Home(props: { type: '' | 'movie' | 'series'; active: boolean }) {
  const [focused, setFocused] = createSignal<MetaPreview>();
  const [rows] = createResource(() => [props.type, addonsVersion()] as const, ([type]) => content.rows(type));
  const [resume, { refetch }] = createResource(async () => {
    const list = await library.continueWatching();
    return props.type ? list.filter((e) => e.type === props.type) : list;
  });

  // Refresh progress whenever the screen comes back into view.
  createEffect(on(() => props.active, (active, was) => active && was === false && refetch()));

  // Focus the first card once one renders, unless the user already moved.
  let focusedOnce = false;
  const ready = () => {
    if (focusedOnce || !props.active) return;
    setTimeout(() => {
      if (focusedOnce) return;
      const active = document.activeElement;
      if (active && active !== document.body) {
        focusedOnce = true;
        return;
      }
      focusFirst(document.querySelector('.home-pane') ?? document);
      focusedOnce = document.activeElement !== document.body;
    });
  };

  // Before anything is focused, the hero shows the first catalog item.
  const [first, setFirst] = createSignal<MetaPreview>();
  const hero = () => focused() ?? first();
  return (
    <div class="screen">
      <Backdrop src={hero()?.background} />
      <div class="hero">
        <Show when={hero()}>
          {(item) => (
            <>
              <Show when={item().logo} fallback={<h1>{item().name}</h1>}>
                <img class="logo" src={item().logo} alt={item().name} />
              </Show>
              <MetaLine item={item()} />
              <Show when={item().description}>
                <p class="description">{item().description}</p>
              </Show>
            </>
          )}
        </Show>
      </div>
      <div class="pane home-pane" data-nav-pane data-nav-top="0.02">
        <Show when={resume()?.length}>
          <ResumeRow entries={resume()!} onFocusItem={setFocused} onReady={ready} />
        </Show>
        <Show when={rows() && rows()!.length === 0}>
          <p class="notice" style={{ 'padding-left': 'var(--gutter)' }}>
            {t('home.empty')}
          </p>
        </Show>
        <For each={rows()}>
          {(row) => (
            <CatalogRow
              row={row}
              onFocusItem={setFocused}
              onReady={(items) => {
                if (!first()) setFirst(items[0]);
                ready();
              }}
            />
          )}
        </For>
      </div>
    </div>
  );
}

function ResumeRow(props: { entries: Entry[]; onFocusItem: (m: MetaPreview) => void; onReady: () => void }) {
  props.onReady();
  return (
    <section class="row" data-nav-block>
      <h2>{t('home.continue')}</h2>
      <div class="row-track" data-nav-row data-nav-group>
        <For each={props.entries}>
          {(e) => {
            const item: MetaPreview = { id: e.id, type: e.type, name: e.name, poster: e.poster, background: e.background, logo: e.logo };
            const episode = e.season || e.episode ? `S${e.season ?? 0} E${e.episode ?? 0}` : undefined;
            const fraction = e.duration > 0 ? e.position / e.duration : 0;
            return (
              <Card
                item={item}
                shape="landscape"
                progress={fraction >= 0.92 ? undefined : fraction}
                caption={e.name}
                subcaption={episode && e.episodeTitle ? `${episode} · ${e.episodeTitle}` : episode}
                default
                onFocus={() => props.onFocusItem(item)}
                onSelect={() => push({ name: 'details', type: e.type, id: e.id, preview: item, autoplay: true })}
              />
            );
          }}
        </For>
      </div>
    </section>
  );
}

function CatalogRow(props: { row: Row; onFocusItem: (m: MetaPreview) => void; onReady: (items: MetaPreview[]) => void }) {
  const [items] = createResource(() => content.catalog(props.row.addonId, props.row.type, props.row.catalogId));
  createEffect(() => {
    const list = items();
    if (list?.length) props.onReady(list);
  });
  const title = () => `${props.row.name} · ${typeName(props.row.type)}`;
  return (
    <Show when={items.loading || items()?.length}>
      <section class="row" data-nav-block>
        <h2>
          {title()}
          <small>{props.row.addonName}</small>
        </h2>
        <Show when={items()} fallback={<div class="row-placeholder">{t('common.loading')}</div>}>
          <div class="row-track" data-nav-row data-nav-group>
            <For each={items()!.slice(0, rowLimit)}>
              {(item, i) => (
                <Card
                  item={item}
                  default={i() === 0}
                  onFocus={() => props.onFocusItem(item)}
                  onSelect={() => push({ name: 'details', type: item.type, id: item.id, preview: item })}
                />
              )}
            </For>
            <Show when={items()!.length >= rowLimit}>
              <MoreCard
                onSelect={() =>
                  push({
                    name: 'grid',
                    addonId: props.row.addonId,
                    type: props.row.type,
                    catalogId: props.row.catalogId,
                    title: title(),
                    genres: props.row.genres,
                  })
                }
              />
            </Show>
          </div>
        </Show>
      </section>
    </Show>
  );
}
