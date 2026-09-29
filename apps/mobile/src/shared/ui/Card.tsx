import React from "react";
import { View, type StyleProp, type ViewProps, type ViewStyle } from "react-native";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type CardVariant = "default" | "muted" | "accent";
export type CardPadding = "sm" | "md" | "lg" | "xl";

export type CardProps = Omit<ViewProps, "style"> & {
  interactive?: boolean;
  selected?: boolean;
  variant?: CardVariant;
  padding?: CardPadding;
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  base: {
    borderRadius: t.radii.lg,
    borderWidth: 1,
    borderColor: t.colors.border,
    backgroundColor: t.colors.surface
  },
  default: {},
  muted: { backgroundColor: t.colors.surfaceMuted },
  accent: { backgroundColor: t.colors.primarySoft, borderColor: t.colors.primary },
  interactive: { borderColor: t.colors.borderStrong },
  selected: {
    backgroundColor: t.colors.primarySoft,
    borderColor: t.colors.primary,
    borderWidth: 2
  },
  sm: { padding: t.spacing.md },
  md: { padding: t.spacing.lg },
  lg: { padding: t.spacing.xl },
  xl: { padding: t.spacing.xxl }
});

export function Card({
  children,
  interactive = false,
  padding = "md",
  selected = false,
  style,
  variant = "default",
  ...viewProps
}: CardProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View
      style={[
        styles.base,
        styles[variant],
        interactive ? styles.interactive : null,
        selected ? styles.selected : null,
        styles[padding],
        style
      ]}
      {...viewProps}
    >
      {children}
    </View>
  );
}
