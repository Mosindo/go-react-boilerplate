import React, { type ReactNode } from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  type PressableProps,
  type StyleProp,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { controls, radii, spacing, type Palette } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive";
export type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  children?: ReactNode;
  label?: string;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
  textStyle?: StyleProp<TextStyle>;
};

function variantStyles(colors: Palette, variant: ButtonVariant): { box: ViewStyle; text: string } {
  switch (variant) {
    case "primary":
      return { box: { backgroundColor: colors.primary }, text: colors.primaryForeground };
    case "secondary":
      return { box: { backgroundColor: colors.primarySoft, borderWidth: 1, borderColor: colors.primaryBorder }, text: colors.primary };
    case "outline":
      return { box: { backgroundColor: "transparent", borderWidth: 1, borderColor: colors.borderStrong }, text: colors.text };
    case "destructive":
      return { box: { backgroundColor: colors.danger }, text: "#ffffff" };
    default:
      return { box: { backgroundColor: "transparent" }, text: colors.text };
  }
}

export function Button({
  children,
  label,
  loading = false,
  variant = "primary",
  size = "md",
  fullWidth = true,
  disabled,
  style,
  textStyle,
  accessibilityLabel,
  ...props
}: ButtonProps) {
  const { colors } = useTheme();
  const { box, text } = variantStyles(colors, variant);
  const inactive = disabled || loading;
  const content = label ?? children;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? (typeof content === "string" ? content : undefined)}
      accessibilityState={{ disabled: !!inactive, busy: loading }}
      disabled={inactive}
      style={({ pressed }) => [
        styles.base,
        { minHeight: controls.button[size] },
        fullWidth && styles.fullWidth,
        box,
        pressed && styles.pressed,
        inactive && styles.disabled,
        style
      ]}
      {...props}
    >
      {loading ? (
        <ActivityIndicator color={text} />
      ) : typeof content === "string" ? (
        <Text variant="button" style={[{ color: text }, textStyle]}>
          {content}
        </Text>
      ) : (
        content
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: spacing.xl,
    flexDirection: "row"
  },
  fullWidth: { alignSelf: "stretch" },
  pressed: { opacity: 0.85, transform: [{ scale: 0.985 }] },
  disabled: { opacity: 0.5 }
});
