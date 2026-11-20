import { render } from 'solid-js/web';
import { createEffect, createSignal, For, onCleanup } from 'solid-js';
import { LinkButton, Theme } from '@plunz/storybook';
import { copy, detectLocale, readPreference, savePreference, type Copy, type Locale, type Preference } from './locale';
import '@plunz/storybook/styles.css';
import './style.css';

const repo = 'https://github.com/plu-nz/app.lounge';
const archive = 'lounge-linux-x86_64.tar.gz';
const download = `${repo}/releases/latest/download/${archive}`;

function Icon(props: { class?: string }) {
  return <svg class={props.class} viewBox="0 0 128 128" aria-hidden="true">
    <rect width="128" height="128" rx="28" fill="#1c1c1c" />
    <rect x="22" y="28" width="84" height="54" rx="8" fill="none" stroke="#89b4fa" stroke-width="7" />
    <path d="M56 44v22l18-11z" fill="#89b4fa" />
    <path d="M30 96c0-6 4-9 9-9h50c5 0 9 3 9 9v6H30z" fill="#cdd6f4" />
  </svg>;
}

function Arrow(props: { direction?: 'down' | 'up' }) {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path d={props.direction === 'up' ? 'M7 17 17 7M8 7h9v9' : 'M12 4v16m-6-6 6 6 6-6'} stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
  </svg>;
}

/** An illustration of Lounge's home screen; the posters are abstract, not real titles. */
function Screen(props: { t: Copy }) {
  const posters = ['p1', 'p2', 'p3', 'p4', 'p5', 'p6'];
  return <figure class="tv" aria-hidden="true">
    <div class="screen">
      <div class="mock-backdrop" />
      <div class="mock-nav">
        <span class="on">{props.t.mockHome}</span><span>{props.t.mockMovies}</span><span>{props.t.mockSeries}</span><span>{props.t.mockSearch}</span>
      </div>
      <div class="mock-hero">
        <p class="mock-kicker">{props.t.mockKicker}</p>
        <p class="mock-title">Northern Lights</p>
        <p class="mock-meta">{props.t.mockMeta}</p>
        <div class="mock-progress"><span /></div>
      </div>
      <div class="mock-row"><For each={posters}>{(poster, index) => <span class={`poster ${poster}`} classList={{ focused: index() === 1 }} />}</For></div>
    </div>
    <div class="stand" />
  </figure>;
}

function CodeLine(props: { code: string; t: Copy }) {
  const [copied, setCopied] = createSignal(false);
  let timer: number | undefined;
  onCleanup(() => clearTimeout(timer));
  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(props.code);
      setCopied(true);
      clearTimeout(timer);
      timer = window.setTimeout(() => setCopied(false), 1600);
    } catch { /* The visitor can still select the text. */ }
  };
  return <div class="code"><code>{props.code}</code><button type="button" onClick={copyCode} aria-live="polite">{copied() ? props.t.copied : props.t.copy}</button></div>;
}

function App() {
  const [preference, setPreference] = createSignal<Preference>(readPreference());
  const [systemLocale, setSystemLocale] = createSignal<Locale>(detectLocale());
  const locale = (): Locale => preference() === 'system' ? systemLocale() : preference() as Locale;
  const t = () => copy[locale()];
  const changeLanguage = (value: Preference) => { setPreference(value); savePreference(value); };
  const refreshSystem = () => setSystemLocale(detectLocale());
  const refreshPreference = () => setPreference(readPreference());
  window.addEventListener('languagechange', refreshSystem);
  window.addEventListener('pageshow', refreshPreference);
  onCleanup(() => {
    window.removeEventListener('languagechange', refreshSystem);
    window.removeEventListener('pageshow', refreshPreference);
  });
  createEffect(() => {
    document.documentElement.lang = locale();
    document.title = t().title;
    document.querySelector('meta[name="description"]')?.setAttribute('content', t().description);
  });

  return <Theme accent="blue" shape="square">
    <a class="skip-link" href="#main">{t().skip}</a>
    <header class="site-header shell">
      <a class="brand" href="/"><Icon class="brand-icon" /><span>Lounge</span></a>
      <nav aria-label="Lounge"><a href="#idea">{t().navIdea}</a><a href="#features">{t().navFeatures}</a><a href="#install">{t().navInstall}</a></nav>
      <label class="language-control"><span>{t().language}</span><select aria-label={t().language} value={preference()} onChange={event => changeLanguage(event.currentTarget.value as Preference)}><option value="system">{t().system}</option><option value="en">English</option><option value="pt">Português</option><option value="es">Español</option></select></label>
    </header>

    <main id="main">
      <section class="hero shell" aria-labelledby="hero-title">
        <div class="hero-copy">
          <p class="eyebrow">{t().eyebrow}</p>
          <h1 id="hero-title">{t().headlineStart}<br /><em>{t().headlineEnd}</em></h1>
          <p class="lead">{t().lead}</p>
          <div class="actions">
            <LinkButton href={download}>{t().download} <Arrow /></LinkButton>
            <LinkButton href={repo} variant="outline">{t().source}</LinkButton>
          </div>
          <p class="download-meta">{t().downloadMeta}</p>
        </div>
        <Screen t={t()} />
      </section>

      <section class="idea" id="idea" aria-labelledby="idea-title"><div class="shell">
        <p class="eyebrow">{t().ideaEyebrow}</p>
        <h2 id="idea-title">{t().ideaTitle}</h2>
        <p class="idea-body">{t().ideaBody}</p>
        <div class="pillars"><For each={t().pillars}>{(pillar, index) => <article><span class="pillar-no">0{index() + 1}</span><h3>{pillar.title}</h3><p>{pillar.body}</p></article>}</For></div>
      </div></section>

      <section class="features shell" id="features" aria-labelledby="features-title">
        <div>
          <p class="eyebrow">{t().featuresEyebrow}</p>
          <h2 id="features-title">{t().featuresTitle}</h2>
          <ul class="feature-list"><For each={t().features}>{feature => <li>{feature}</li>}</For></ul>
        </div>
        <div class="controls">
          <h3>{t().controlsTitle}</h3>
          <dl><For each={t().controls}>{([key, action]) => <div><dt><kbd>{key}</kbd></dt><dd>{action}</dd></div>}</For></dl>
        </div>
      </section>

      <section class="install" id="install" aria-labelledby="install-title"><div class="shell">
        <div class="install-head">
          <div><p class="eyebrow">{t().installEyebrow}</p><h2 id="install-title">{t().installTitle}</h2></div>
          <div><p>{t().installLead}</p><LinkButton href={download}>{t().download} <Arrow /></LinkButton></div>
        </div>
        <ol class="steps"><For each={t().steps}>{(step, index) => <li><span class="step-no">{index() + 1}</span><div><h3>{step.title}</h3><CodeLine code={step.code} t={t()} /></div></li>}</For></ol>
        <p class="install-note">{t().installNote}</p>
        <p class="install-links"><a href={`${download}.sha256`}>{t().checksum}</a><a href={`${repo}/releases`}>{t().releases}</a><a href={`${repo}#install`}>{t().fromSource}</a></p>
      </div></section>

      <section class="pair shell" aria-label={t().pairEyebrow}>
        <p class="eyebrow">{t().pairEyebrow}</p>
        <p>{t().pairBody}</p>
        <a href="https://nexus.plu.nz/">{t().pairLink} <Arrow direction="up" /></a>
      </section>
    </main>

    <footer class="site-footer shell">
      <p class="disclaimer">{t().disclaimer}</p>
      <div class="footer-links">
        <a href="https://www.plu.nz/">{t().partOf}</a>
        <a href="https://www.plu.nz/privacy">{t().privacy}</a>
        <a href="https://www.plu.nz/terms">{t().terms}</a>
        <a href={repo}>GitHub <Arrow direction="up" /></a>
      </div>
    </footer>
  </Theme>;
}

render(() => <App />, document.getElementById('root')!);
