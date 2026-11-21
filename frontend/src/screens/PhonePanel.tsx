import { createSignal, onCleanup, onMount, Show } from 'solid-js';
import { phoneAdd, type PhoneSession } from '../api';
import { errorText, Panel, Spinner, toast } from '../components';
import { t } from '../i18n';

/** PhonePanel shows a QR code for the short-lived LAN page that adds addons. */
export function PhonePanel(props: { onClose: () => void }) {
  const [session, setSession] = createSignal<PhoneSession>();
  const [error, setError] = createSignal('');
  const [ended, setEnded] = createSignal(false);
  let closed = false;

  const close = () => {
    closed = true;
    void phoneAdd.stop();
    props.onClose();
  };

  onMount(() => {
    const off = phoneAdd.on((e) => {
      if (e.stopped) {
        if (!closed) setEnded(true);
        return;
      }
      if (e.ok) {
        toast(t('addons.installed', { name: e.name ?? '' }));
        close();
      } else if (e.error) {
        // The phone shows the error too; keep waiting for another try.
        toast(e.error);
      }
    });
    onCleanup(off);
    phoneAdd
      .start()
      .then(setSession)
      .catch((err) => setError(t('phone.noNetwork', { error: errorText(err) })));
  });
  // Leaving the screen any other way still stops the server.
  onCleanup(() => {
    if (!closed) void phoneAdd.stop();
  });

  return (
    <Panel title={t('phone.title')} onClose={close}>
      <div class="phone-add">
        <Show when={!error()} fallback={<p class="error">{error()}</p>}>
          <Show when={session()} fallback={<Spinner />}>
            {(s) => (
              <Show when={!ended()} fallback={<p class="notice">{t('phone.ended')}</p>}>
                <img class="qr" src={s().qrcode} alt="" />
                <p class="notice">{t('phone.help', { link: s().url.replace(/^https?:\/\//, '') })}</p>
                <p class="notice">{t('phone.waiting')}</p>
              </Show>
            )}
          </Show>
        </Show>
        <button class="btn" data-nav data-nav-default onClick={close}>
          {t('phone.close')}
        </button>
      </div>
    </Panel>
  );
}
