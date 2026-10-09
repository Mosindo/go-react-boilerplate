import React from 'react';
import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from 'react-native';
import { fontSize, useTheme } from '../../theme';

export type TextVariant =
  'display' | 'title' | 'heading' | 'body' | 'bodyStrong' | 'caption' | 'label';
export type TextTone = 'default' | 'muted' | 'primary' | 'danger' | 'success' | 'onPrimary';

export type TextProps = RNTextProps & { variant?: TextVariant; tone?: TextTone; center?: boolean };

const variants: Record<TextVariant, TextStyle> = {
  display: { fontSize: fontSize.xxl, fontWeight: '800', lineHeight: 36, letterSpacing: -0.5 },
  title: { fontSize: fontSize.xl, fontWeight: '700', lineHeight: 28, letterSpacing: -0.2 },
  heading: { fontSize: fontSize.lg, fontWeight: '700', lineHeight: 24 },
  body: { fontSize: fontSize.md, fontWeight: '400', lineHeight: 22 },
  bodyStrong: { fontSize: fontSize.md, fontWeight: '600', lineHeight: 22 },
  caption: { fontSize: fontSize.sm, fontWeight: '400', lineHeight: 18 },
  label: { fontSize: fontSize.sm, fontWeight: '600', lineHeight: 18 },
};

function TextImpl({ variant = 'body', tone = 'default', center, style, ...rest }: TextProps) {
  const { colors } = useTheme();
  const color = {
    default: colors.text,
    muted: colors.textMuted,
    primary: colors.primary,
    danger: colors.danger,
    success: colors.success,
    onPrimary: colors.onPrimary,
  }[tone];
  return (
    <RNText
      {...rest}
      style={[variants[variant], { color }, center ? { textAlign: 'center' } : null, style]}
    />
  );
}

export const Text = React.memo(TextImpl);
