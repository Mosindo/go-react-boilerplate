import React, { useMemo } from "react";
import { Text as RNText, type StyleProp, type TextProps as RNTextProps, type TextStyle } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { fontWeights, typography } from "./tokens";

export type TextVariant = "body" | "label" | "title" | "heading" | "caption" | "eyebrow" | "button";
export type TextTone = "default" | "muted" | "primary" | "secondary" | "danger" | "success" | "inverse";
export type TextWeight = "regular" | "medium" | "semibold" | "bold";

export type TextProps = RNTextProps & {
  variant?: TextVariant;
  tone?: TextTone;
  weight?: TextWeight;
  style?: StyleProp<TextStyle>;
};

const variantStyles: Record<TextVariant, TextStyle> = {
  body: typography.body,
  label: typography.label,
  title: typography.title,
  heading: typography.heading,
  caption: typography.caption,
  eyebrow: typography.eyebrow,
  button: typography.button
};

export function Text({
  children,
  style,
  tone = "default",
  variant = "body",
  weight = "regular",
  ...props
}: TextProps) {
  const { colors } = useTheme();
  const toneColor = useMemo<Record<TextTone, string>>(
    () => ({
      default: colors.text,
      muted: colors.textMuted,
      primary: colors.primary,
      secondary: colors.secondary,
      danger: colors.danger,
      success: colors.success,
      inverse: colors.inverse
    }),
    [colors]
  );

  return (
    <RNText
      style={[variantStyles[variant], { fontWeight: fontWeights[weight], color: toneColor[tone] }, style]}
      {...props}
    >
      {children}
    </RNText>
  );
}
