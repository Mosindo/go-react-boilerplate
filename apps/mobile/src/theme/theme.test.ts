import { describe, expect, it } from "vitest";
import { darkColors, lightColors, type Colors } from "./tokens";

function luminance(hex: string): number {
  const channels = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255);
  const [r, g, b] = channels.map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

// WCAG AA: 4.5:1 for normal text.
const pairs: [keyof Colors, keyof Colors][] = [
  ["text", "background"],
  ["text", "surface"],
  ["textMuted", "background"],
  ["textMuted", "surface"],
  ["onPrimary", "primary"],
  ["primary", "background"],
  ["onBubbleMine", "bubbleMine"],
  ["text", "bubbleTheirs"],
  ["danger", "dangerSoft"],
  ["primary", "primarySoft"]
];

describe.each([
  ["light", lightColors],
  ["dark", darkColors]
])("%s palette", (_name, colors) => {
  it.each(pairs)("%s on %s meets WCAG AA contrast", (fg, bg) => {
    expect(contrast(colors[fg], colors[bg])).toBeGreaterThanOrEqual(4.5);
  });
});
