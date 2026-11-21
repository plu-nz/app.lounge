import type { Settings } from './api';
import type { t } from './i18n';

type MessageKey = Parameters<typeof t>[0];

/** Subtitle size presets, S to XL; the value is mpv's sub-scale. */
export const subSizes = [
  { value: 0.8, label: 'sub.size.s' },
  { value: 1, label: 'sub.size.m' },
  { value: 1.3, label: 'sub.size.l' },
  { value: 1.6, label: 'sub.size.xl' },
] as const;

// The names match internal/subtitle; the colors are for the Settings preview.
export const subColors = { white: '#ffffff', yellow: '#ffe94a', green: '#7cff8a', cyan: '#6fe7ff' } as const;
export const subBackgrounds = ['none', 'outline', 'box'] as const;
export const subPositions = ['bottom', 'raised', 'high'] as const;

/** nearestSize picks the preset closest to a saved scale. */
export function nearestSize(scale: number): number {
  const s = scale || 1;
  let best: number = subSizes[1].value;
  for (const o of subSizes) if (Math.abs(o.value - s) < Math.abs(best - s)) best = o.value;
  return best;
}

export const sizeKey = (scale: number) => subSizes.find((o) => o.value === nearestSize(scale))!.label as MessageKey;

/** previewCss draws a subtitle the way mpv will, for the Settings preview. */
export function previewCss(s: Settings): Record<string, string> {
  const color = subColors[(s.subtitleColor || 'white') as keyof typeof subColors] ?? subColors.white;
  const bg = s.subtitleBackground || 'outline';
  const css: Record<string, string> = {
    color,
    'font-size': `${1.1 * nearestSize(s.subtitleScale)}rem`,
    background: 'transparent',
    'text-shadow': 'none',
  };
  if (bg === 'outline') css['text-shadow'] = '0 0 0.2em #000, 0 0 0.12em #000, 0.05em 0.05em 0.1em #000';
  if (bg === 'box') css.background = 'rgb(0 0 0 / 0.7)';
  return css;
}
