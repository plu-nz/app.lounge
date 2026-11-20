import { createSignal, onCleanup, onMount, Show } from 'solid-js';
import { LogIn, RefreshCw } from 'lucide-solid';
import { account, type LinkCode } from '../api';
import { errorText, Spinner, toast } from '../components';
import { t } from '../i18n';
import { focusFirst } from '../nav';
import { pop } from '../router';

const pollEvery = 3000;
const codeLifetime = 5 * 60 * 1000;

/** Login offers the phone code flow first, since typing on a remote is slow. */
export function Login() {
  const [link, setLink] = createSignal<LinkCode>();
  const [expired, setExpired] = createSignal(false);
  const [linkError, setLinkError] = createSignal('');
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [busy, setBusy] = createSignal(false);
  let poll: number | undefined;
  let done = false;

  const finish = () => {
    done = true;
    clearInterval(poll);
    toast(t('account.welcome'));
    pop();
  };

  const startLink = async () => {
    clearInterval(poll);
    setExpired(false);
    setLinkError('');
    setLink(undefined);
    try {
      const code = await account.createLink();
      if (done) return;
      setLink(code);
      const started = Date.now();
      let checking = false;
      poll = window.setInterval(async () => {
        if (checking || done) return;
        if (Date.now() - started > codeLifetime) {
          clearInterval(poll);
          setExpired(true);
          return;
        }
        checking = true;
        try {
          const status = await account.checkLink(code.code);
          if (status.signedIn) finish();
        } catch (err) {
          setLinkError(errorText(err));
        } finally {
          checking = false;
        }
      }, pollEvery);
    } catch (err) {
      setLinkError(errorText(err));
    }
  };

  const submit = async () => {
    if (busy() || !email().trim() || !password()) return;
    setBusy(true);
    try {
      await account.login(email().trim(), password());
      finish();
    } catch (err) {
      toast(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  onMount(() => {
    void startLink();
    queueMicrotask(() => focusFirst(document.querySelector('.login') ?? document));
  });
  onCleanup(() => {
    done = true;
    clearInterval(poll);
  });

  return (
    <div class="page login">
      <h1>{t('account.signIn')}</h1>
      <div class="login-layout">
        <section class="login-card">
          <h2>{t('account.linkTitle')}</h2>
          <Show when={link()} fallback={<Show when={!linkError()} fallback={<p class="error">{linkError()}</p>}><Spinner /></Show>}>
            {(code) => (
              <>
                <Show when={!expired()} fallback={<p class="notice">{t('account.linkExpired')}</p>}>
                  <img class="qr" src={code().qrcode} alt="" />
                  <div class="link-code">{code().code}</div>
                  <p class="notice">{t('account.linkHelp', { link: code().link.replace(/^https?:\/\//, '') })}</p>
                  <p class="notice">{t('account.linkWaiting')}</p>
                </Show>
              </>
            )}
          </Show>
          <button class="btn small" data-nav onClick={() => void startLink()}>
            <RefreshCw />
            {t('account.newCode')}
          </button>
        </section>
        <section class="login-card">
          <h2>{t('account.passwordTitle')}</h2>
          <input
            class="field"
            type="email"
            data-nav
            data-nav-default
            placeholder={t('account.email')}
            autocomplete="username"
            value={email()}
            onInput={(e) => setEmail(e.currentTarget.value)}
          />
          <input
            class="field"
            type="password"
            data-nav
            placeholder={t('account.password')}
            autocomplete="current-password"
            value={password()}
            onInput={(e) => setPassword(e.currentTarget.value)}
            onKeyDown={(e) => e.key === 'Enter' && void submit()}
          />
          <button class="btn primary" data-nav disabled={busy()} onClick={() => void submit()}>
            {busy() ? <Spinner /> : <LogIn />}
            {t('account.signIn')}
          </button>
        </section>
      </div>
    </div>
  );
}
