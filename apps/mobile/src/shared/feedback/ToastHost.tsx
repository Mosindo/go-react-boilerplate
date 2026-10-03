import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Text, colors, radii, shadows, spacing } from "../ui";
import { dismissToast, useToasts } from "./toast";

const toneColor = { info: colors.text, success: colors.success, error: colors.danger } as const;

export function ToastHost() {
  const toasts = useToasts();
  if (toasts.length === 0) {
    return null;
  }
  return (
    <SafeAreaView edges={["top"]} pointerEvents="box-none" style={styles.host}>
      {toasts.map((toast) => (
        <Pressable
          accessibilityLiveRegion="polite"
          accessibilityRole="alert"
          key={toast.id}
          onPress={() => {
            dismissToast(toast.id);
            toast.onPress?.();
          }}
          style={[styles.toast, { backgroundColor: toneColor[toast.tone] }]}
        >
          <View style={styles.row}>
            <Text tone="inverse" weight="semibold">
              {toast.message}
            </Text>
          </View>
        </Pressable>
      ))}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  host: {
    position: "absolute",
    top: 0,
    left: spacing.lg,
    right: spacing.lg,
    gap: spacing.sm
  },
  toast: {
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    ...shadows.floating
  },
  row: { flexDirection: "row", alignItems: "center" }
});
