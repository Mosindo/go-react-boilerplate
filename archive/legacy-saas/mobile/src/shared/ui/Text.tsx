import React from "react";
import { StyleSheet, Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { colors, fontWeights, typography } from "./tokens";

export type TextVariant = "body" | "label" | "title" | "heading" | "caption" | "eyebrow" | "button";
export type TextTone = "default" | "muted" | "primary" | "secondary" | "danger" | "success" | "inverse";
export type TextWeight = "regular" | "medium" | "semibold" | "bold";

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: TextWeight;
  style?: StyleProp<TextStyle>;
};

const styles = StyleSheet.create({
  base: {
    color: colors.text,
    ...typography.body
  },
  regular: {
    fontWeight: fontWeights.regular
  },
  medium: {
    fontWeight: fontWeights.medium
  },
  semibold: {
    fontWeight: fontWeights.semibold
  },
  bold: {
    fontWeight: fontWeights.bold
  },
  defaultTone: {
    color: colors.text
  },
  mutedTone: {
    color: colors.textMuted
  },
  primaryTone: {
    color: colors.primary
  },
  secondaryTone: {
    color: colors.secondary
  },
  dangerTone: {
    color: colors.danger
  },
  successTone: {
    color: colors.success
  },
  inverseTone: {
    color: colors.inverse
  }
});

const variantStyles: Record<TextVariant, TextStyle> = {
  body: typography.body,
  label: typography.label,
  title: typography.title,
  heading: typography.heading,
  caption: typography.caption,
  eyebrow: typography.eyebrow,
  button: typography.button
};

const weightStyles: Record<TextWeight, TextStyle> = {
  regular: styles.regular,
  medium: styles.medium,
  semibold: styles.semibold,
  bold: styles.bold
};

const toneStyles: Record<TextTone, TextStyle> = {
  default: styles.defaultTone,
  muted: styles.mutedTone,
  primary: styles.primaryTone,
  secondary: styles.secondaryTone,
  danger: styles.dangerTone,
  success: styles.successTone,
  inverse: styles.inverseTone
};

export function Text({
  children,
  style,
  tone = "default",
  variant = "body",
  weight = "regular",
  ...props
}: TextProps) {
  return (
    <RNText style={[styles.base, variantStyles[variant], weightStyles[weight], toneStyles[tone], style]} {...props}>
      {children}
    </RNText>
  );
}
