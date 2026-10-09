export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  onPrimary: string;
  primarySoft: string;
  accent: string;
  onAccent: string;
  success: string;
  danger: string;
  onDanger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
  skeleton: string;
  bubbleMine: string;
  onBubbleMine: string;
  bubbleTheirs: string;
  onBubbleTheirs: string;
  outside: string;
};

/** Warm cream + deep plum ink + coral accent. */
export const lightPalette: Palette = {
  background: '#FBF6F0',
  surface: '#FFFFFF',
  surfaceAlt: '#F3EAE1',
  border: '#E6D9CC',
  text: '#2A1633',
  textMuted: '#6B5A70',
  primary: '#D13F2B',
  onPrimary: '#FFFFFF',
  primarySoft: '#FBE3DD',
  accent: '#4B1D5C',
  onAccent: '#FFFFFF',
  success: '#2E7D4F',
  danger: '#B3261E',
  onDanger: '#FFFFFF',
  dangerSoft: '#FBE4E2',
  overlay: 'rgba(42,22,51,0.55)',
  scrim: 'rgba(20,8,26,0.55)',
  skeleton: '#EADFD3',
  bubbleMine: '#4B1D5C',
  onBubbleMine: '#FFFFFF',
  bubbleTheirs: '#F0E5DA',
  onBubbleTheirs: '#2A1633',
  outside: '#EFE6DC',
};

export const darkPalette: Palette = {
  background: '#171019',
  surface: '#241A2B',
  surfaceAlt: '#31243A',
  border: '#46364F',
  text: '#F6EEF3',
  textMuted: '#B9A8BF',
  primary: '#FF7A66',
  onPrimary: '#2A0F0A',
  primarySoft: '#4A2420',
  accent: '#C9A0DC',
  onAccent: '#1E0D27',
  success: '#6FCF97',
  danger: '#FF8A80',
  onDanger: '#2B0B09',
  dangerSoft: '#4A1F1D',
  overlay: 'rgba(0,0,0,0.65)',
  scrim: 'rgba(0,0,0,0.6)',
  skeleton: '#33263C',
  bubbleMine: '#7B3F95',
  onBubbleMine: '#FFFFFF',
  bubbleTheirs: '#33263C',
  onBubbleTheirs: '#F6EEF3',
  outside: '#0F0A11',
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 12, lg: 20, pill: 999 } as const;
export const fontSize = { xs: 12, sm: 14, md: 16, lg: 18, xl: 22, xxl: 30 } as const;
export const MAX_CONTENT_WIDTH = 560;
export const hitSlop = { top: 8, bottom: 8, left: 8, right: 8 } as const;
