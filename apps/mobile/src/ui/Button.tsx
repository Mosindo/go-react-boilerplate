import React from "react";
import { ActivityIndicator, View } from "react-native";
import { createStyles } from "../theme/ThemeProvider";
import { ScalePressable } from "./Pressable";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

type Props = {
  label: string;
  onPress: () => void;
  variant?: Variant;
  loading?: boolean;
  disabled?: boolean;
  testID?: string;
  accessibilityLabel?: string;
  compact?: boolean;
};

const useStyles = createStyles(({ colors, radius, spacing, minTouch }) => ({
  base: { alignSelf: "stretch" },
  inner: {
    minHeight: minTouch + 4,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center",
    flexDirection: "row",
    gap: spacing.sm
  },
  compact: { minHeight: minTouch, paddingHorizontal: spacing.lg },
  primary: { backgroundColor: colors.primary },
  secondary: { backgroundColor: colors.surfaceAlt, borderWidth: 1, borderColor: colors.border },
  ghost: { backgroundColor: "transparent" },
  danger: { backgroundColor: colors.danger },
  disabled: { opacity: 0.5 }
}));

export function Button({
  label,
  onPress,
  variant = "primary",
  loading,
  disabled,
  testID,
  accessibilityLabel,
  compact
}: Props) {
  const styles = useStyles();
  const inactive = disabled || loading;
  const tone = variant === "primary" ? "onPrimary" : variant === "danger" ? "onPrimary" : "primary";
  return (
    <ScalePressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      style={[styles.base, inactive ? styles.disabled : {}]}
      testID={testID}
    >
      <View style={[styles.inner, compact ? styles.compact : {}, styles[variant]]}>
        {loading ? (
          <ActivityIndicator
            color={variant === "ghost" || variant === "secondary" ? undefined : "#fff"}
          />
        ) : null}
        <Text variant="label" tone={tone}>
          {label}
        </Text>
      </View>
    </ScalePressable>
  );
}
