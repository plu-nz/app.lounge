import { render } from 'solid-js/web';
import { createEffect, For, on, onMount, Show } from 'solid-js';
import { Window } from '@wailsio/runtime';
import './styles.css';
import { Rail, Toast } from './components';
import { focusEl, focusFirst, installNavigation, onBack } from './nav';
import { depth, pop, restoredFocus, route, stackRoutes, type Route } from './router';
import { loadSettings, settings } from './settings';
import { Details } from './screens/Details';
import { Home } from './screens/Home';
import { Addons, Grid, Search, SettingsPage } from './screens/Pages';
import { Player } from './screens/Player';
import { Login } from './screens/Login';

function Screen(props: { route: Route; active: boolean }) {
  const r = props.route;
  switch (r.name) {
    case 'home':
      return <Home type={r.type} active={props.active} />;
    case 'search':
      return <Search />;
    case 'addons':
      return <Addons />;
    case 'settings':
      return <SettingsPage active={props.active} />;
    case 'login':
      return <Login />;
    case 'grid':
      return <Grid {...r} />;
    case 'details':
      return <Details type={r.type} id={r.id} preview={r.preview} autoplay={r.autoplay} active={props.active} />;
    case 'player':
      return <Player play={r.play} active={props.active} />;
  }
}

function App() {
  installNavigation();
  onBack(() => pop());
  addEventListener('keydown', (e) => {
    if (e.key === 'F11') {
      e.preventDefault();
      void Window.ToggleFullscreen();
    }
  });
  onMount(() => void loadSettings().catch(() => undefined));

  // Return focus to where it was when a pushed screen closes.
  createEffect(
    on(depth, (d, prev) => {
      if (prev !== undefined && d < prev) {
        queueMicrotask(() => {
          const el = restoredFocus();
          if (el) focusEl(el);
          else focusFirst();
        });
      }
    }),
  );

  const isRoot = () => depth() === 1;
  const playing = () => route().name === 'player';
  return (
    <div class="app plz-theme" data-plz-shape="rounded" data-plz-accent={settings().accent || 'blue'} classList={{ playing: playing() }}>
      <For each={stackRoutes()}>
        {(r, i) => (
          <div class="screen" classList={{ hidden: i() !== stackRoutes().length - 1 }}>
            <Screen route={r} active={i() === stackRoutes().length - 1} />
          </div>
        )}
      </For>
      <Show when={isRoot()}>
        <Rail />
      </Show>
      <Toast />
    </div>
  );
}

render(() => <App />, document.getElementById('root')!);

