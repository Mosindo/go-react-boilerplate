import React from "react";
import { Pressable, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Text } from "../ui/Text";
import { useTheme, type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";
import { dismissToast, useToasts } from "./toast";

const makeStyles = (t: Theme) => ({
  wrap: {
    position: "absolute" as const,
    left: t.spacing.lg,
    right: t.spacing.lg,
    gap: t.spacing.sm,
    alignItems: "center" as const
  },
  toast: {
    maxWidth: 480,
    width: "100%" as const,
    minHeight: 44,
    justifyContent: "center" as const,
    borderRadius: t.radii.md,
    paddingHorizontal: t.spacing.lg,
    paddingVertical: t.spacing.sm,
    backgroundColor: t.colors.text,
    ...t.shadows.floating
  }
});

/** Renders transient toasts above everything. Mount once near the app root. */
export function ToastHost() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const toasts = useToasts();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) {
    return null;
  }
  return (
    <View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + 8 }]}>
      {toasts.map((toast) => (
        <Pressable
          accessibilityLiveRegion="polite"
          accessibilityRole="alert"
          key={toast.id}
          onPress={() => dismissToast(toast.id)}
          style={styles.toast}
          testID="toast"
        >
          <Text style={{ color: theme.colors.background }} variant="label" weight="semibold">
            {toast.message}
          </Text>
        </Pressable>
      ))}
    </View>
  );
}
