// Spatial navigation for remote controls: arrow keys move focus to the
// nearest focusable element in that direction.
//
// Mark focusable elements with `data-nav`. A container with `data-nav-scope`
// traps focus (the last one in the DOM wins, so dialogs stack). A container
// with `data-nav-group` remembers its last focused child, so moving back
// into a row returns to the same card. Horizontal rows with `data-nav-row`
// scroll to keep the focused card near their start.

type Dir = 'left' | 'right' | 'up' | 'down';

const keyDirs: Record<string, Dir> = {
  ArrowLeft: 'left',
  ArrowRight: 'right',
  ArrowUp: 'up',
  ArrowDown: 'down',
};

function scope(): ParentNode {
  const scopes = document.querySelectorAll<HTMLElement>('[data-nav-scope]');
  return scopes.length ? scopes[scopes.length - 1] : document;
}

function visible(el: HTMLElement): boolean {
  if (el.closest('[inert]') || (el as HTMLButtonElement).disabled) return false;
  const r = el.getBoundingClientRect();
  return r.width > 0 && r.height > 0;
}

export function focusables(root: ParentNode = scope()): HTMLElement[] {
  return [...root.querySelectorAll<HTMLElement>('[data-nav]')].filter(visible);
}

/** focusFirst focuses the preferred element in the current scope. */
export function focusFirst(root: ParentNode = scope()) {
  const preferred = root.querySelector<HTMLElement>('[data-nav-default]');
  const el = preferred && visible(preferred) ? preferred : focusables(root)[0];
  if (el) focusEl(el);
}

export function focusEl(el: HTMLElement) {
  el.focus({ preventScroll: true });
  reveal(el);
  const group = el.closest<HTMLElement>('[data-nav-group]');
  if (group) lastInGroup.set(group, el);
}

const lastInGroup = new WeakMap<HTMLElement, HTMLElement>();

// Scroll animations in flight, by element, with their end position.
const glides = new WeakMap<HTMLElement, { frame: number; to: number }>();

/**
 * glide scrolls an element quickly. WebKit's smooth scrolling takes too long
 * to follow a remote whose arrow key is held down.
 */
export function glide(el: HTMLElement, axis: 'x' | 'y', to: number) {
  const prop = axis === 'x' ? 'scrollLeft' : 'scrollTop';
  const max = axis === 'x' ? el.scrollWidth - el.clientWidth : el.scrollHeight - el.clientHeight;
  const end = Math.max(0, Math.min(to, max));
  cancelAnimationFrame(glides.get(el)?.frame ?? 0);
  glides.delete(el);
  const from = el[prop];
  if (Math.abs(end - from) < 1) return;
  const start = performance.now();
  const duration = 160;
  const step = (now: number) => {
    const t = Math.min(1, (now - start) / duration);
    el[prop] = from + (end - from) * (1 - (1 - t) ** 3);
    if (t < 1) glides.set(el, { frame: requestAnimationFrame(step), to: end });
    else glides.delete(el);
  };
  glides.set(el, { frame: requestAnimationFrame(step), to: end });
}

/** scrollTarget is where an element's scroll will rest once its glide ends. */
function scrollTarget(el: HTMLElement, axis: 'x' | 'y'): number {
  return glides.get(el)?.to ?? (axis === 'x' ? el.scrollLeft : el.scrollTop);
}

function reveal(el: HTMLElement) {
  const row = el.closest<HTMLElement>('[data-nav-row]');
  if (row) {
    const pad = parseFloat(getComputedStyle(row).paddingLeft) || 0;
    glide(row, 'x', el.offsetLeft - pad);
  }
  const pane = el.closest<HTMLElement>('[data-nav-pane]');
  if (pane) {
    const block = el.closest<HTMLElement>('[data-nav-block]') ?? el;
    // A block, or a container around it, sets where it rests with
    // data-nav-top: a fraction of the pane height, or "nearest" to scroll
    // only as far as needed.
    const top = block.closest<HTMLElement>('[data-nav-top]')?.dataset.navTop ?? '0.18';
    const pr = pane.getBoundingClientRect();
    const br = block.getBoundingClientRect();
    // Measure against the position a running glide will reach.
    const pending = scrollTarget(pane, 'y') - pane.scrollTop;
    const blockTop = br.top - pending;
    const blockBottom = br.bottom - pending;
    const margin = 24;
    if (blockTop < pr.top + margin || blockBottom > pr.bottom - margin) {
      const delta =
        top !== 'nearest'
          ? blockTop - pr.top - Number(top) * pr.height
          : blockTop < pr.top + margin
            ? blockTop - pr.top - margin
            : blockBottom - pr.bottom + margin;
      glide(pane, 'y', scrollTarget(pane, 'y') + delta);
    }
  } else {
    el.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'smooth' });
  }
}

function center(r: DOMRect) {
  return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
}

/** alignedWith reports a real overlap on the other axis, not a touching edge. */
function alignedWith(from: DOMRect, to: DOMRect, dir: Dir): boolean {
  const [a0, a1, b0, b1] =
    dir === 'left' || dir === 'right' ? [from.top, from.bottom, to.top, to.bottom] : [from.left, from.right, to.left, to.right];
  const overlap = Math.min(a1, b1) - Math.max(a0, b0);
  return overlap > Math.min(a1 - a0, b1 - b0) * 0.3;
}

function score(from: DOMRect, to: DOMRect, dir: Dir): number {
  const a = center(from);
  const b = center(to);
  const horizontal = dir === 'left' || dir === 'right';
  // Candidates must lie in the direction of travel.
  const ahead =
    dir === 'left' ? b.x < a.x - 1 : dir === 'right' ? b.x > a.x + 1 : dir === 'up' ? b.y < a.y - 1 : b.y > a.y + 1;
  if (!ahead) return Infinity;
  const gap =
    dir === 'left' ? from.left - to.right : dir === 'right' ? to.left - from.right : dir === 'up' ? from.top - to.bottom : to.top - from.bottom;
  // Elements that overlap on the other axis (same row or column) are
  // strongly preferred over diagonal ones.
  const overlaps = horizontal
    ? to.bottom > from.top && to.top < from.bottom
    : to.right > from.left && to.left < from.right;
  const offset = horizontal ? Math.abs(a.y - b.y) : Math.abs(a.x - b.x);
  return Math.max(gap, 0) + (overlaps ? offset * 0.3 : offset * 3 + 400);
}

export function move(dir: Dir): boolean {
  const current = document.activeElement as HTMLElement | null;
  const all = focusables();
  if (!current || !current.matches('[data-nav]') || !all.includes(current)) {
    focusFirst();
    return true;
  }
  const from = current.getBoundingClientRect();
  const fromGroup = current.closest<HTMLElement>('[data-nav-group]');
  // Candidates in the same row (or column) win over any diagonal one, so a
  // wide element above, like a seek bar, never steals a sideways move.
  let best: HTMLElement | null = null;
  let bestScore = Infinity;
  let bestAligned = false;
  for (const el of all) {
    if (el === current) continue;
    const to = el.getBoundingClientRect();
    const s = score(from, to, dir);
    if (s === Infinity) continue;
    const aligned = alignedWith(from, to, dir);
    if ((aligned && !bestAligned) || (aligned === bestAligned && s < bestScore)) {
      bestScore = s;
      best = el;
      bestAligned = aligned;
    }
  }
  if (!best) return false;
  // Entering another group lands on its entry point (data-nav-entry), or
  // vertically on the child it last focused.
  const group = best.closest<HTMLElement>('[data-nav-group]');
  if (group && group !== fromGroup) {
    const entry = group.querySelector<HTMLElement>('[data-nav-entry]');
    const last = lastInGroup.get(group);
    if (entry && visible(entry)) best = entry;
    else if ((dir === 'up' || dir === 'down') && last && last.isConnected && visible(last)) best = last;
  }
  focusEl(best);
  return true;
}

// Back handlers run newest first; one that returns true consumes the key.
const backHandlers: (() => boolean)[] = [];

export function onBack(fn: () => boolean): () => void {
  backHandlers.push(fn);
  return () => {
    const i = backHandlers.lastIndexOf(fn);
    if (i >= 0) backHandlers.splice(i, 1);
  };
}

export function back() {
  for (let i = backHandlers.length - 1; i >= 0; i--) {
    if (backHandlers[i]()) return;
  }
}

const backKeys = new Set(['Escape', 'GoBack', 'BrowserBack', 'XF86Back', 'Back']);

function isTextInput(el: Element | null): el is HTMLInputElement {
  return el instanceof HTMLInputElement && ['text', 'search', 'url', 'email', 'password'].includes(el.type);
}

let marked: HTMLElement | null = null;

/** mark moves the visual focus class, which does not depend on window focus. */
function mark(el: HTMLElement | null) {
  if (marked === el) return;
  marked?.classList.remove('focused');
  marked = el;
  el?.classList.add('focused');
}

export function installNavigation() {
  addEventListener('focusin', (e) => mark(e.target instanceof HTMLElement ? e.target : null));
  addEventListener(
    'keydown',
    (e) => {
      if (e.defaultPrevented) return;
      const dir = keyDirs[e.key];
      const active = document.activeElement;
      if (dir) {
        // Inside a text field, left/right move the caret.
        if (isTextInput(active) && (dir === 'left' || dir === 'right')) {
          const atStart = active.selectionStart === 0 && active.selectionEnd === 0;
          const atEnd = active.selectionStart === active.value.length;
          if (!(dir === 'left' && atStart) && !(dir === 'right' && atEnd)) return;
        }
        if (move(dir)) e.preventDefault();
        return;
      }
      if (backKeys.has(e.key) || (e.key === 'Backspace' && !isTextInput(active))) {
        e.preventDefault();
        back();
        return;
      }
      if (e.key === 'Enter' && active instanceof HTMLElement && active.matches('[data-nav]') && !(active instanceof HTMLButtonElement) && !isTextInput(active)) {
        e.preventDefault();
        active.click();
      }
    },
    false,
  );
  // Mouse back buttons act like the remote's back key.
  addEventListener('mouseup', (e) => {
    if (e.button === 3) back();
  });
}
