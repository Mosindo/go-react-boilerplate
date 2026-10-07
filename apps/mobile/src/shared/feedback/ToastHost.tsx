import React, { useEffect } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../../theme/useTheme";
import { Text, radii, shadows, spacing } from "../ui";
import { dismissToast, useGlobalFeedback } from "./store";

const TOAST_DURATION_MS = 4500;

/** Mount once near the root (inside SafeAreaProvider). Shows the latest toast / global error. */
export function ToastHost() {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const { toast } = useGlobalFeedback();

  useEffect(() => {
    if (!toast) {
      return;
    }
    const id = setTimeout(dismissToast, TOAST_DURATION_MS);
    return () => clearTimeout(id);
  }, [toast]);

  if (!toast) {
    return null;
  }

  const accent =
    toast.tone === "error" ? colors.danger : toast.tone === "success" ? colors.success : colors.primary;

  return (
    <View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + spacing.sm }]}>
      <Pressable
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        onPress={dismissToast}
        style={[
          styles.toast,
          { backgroundColor: colors.backgroundElevated, borderColor: accent },
          shadows.floating
        ]}
      >
        <View style={[styles.dot, { backgroundColor: accent }]} />
        <Text style={styles.message} variant="label" weight="medium">
          {toast.message}
        </Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.lg, right: spacing.lg, alignItems: "center", zIndex: 1000 },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    maxWidth: 520,
    width: "100%",
    borderWidth: 1,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  dot: { width: 8, height: 8, borderRadius: 4 },
  message: { flex: 1 }
});
