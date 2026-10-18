import React, { useEffect, useSyncExternalStore } from "react";
import { StyleSheet, View } from "react-native";
import { Text, colors, radii, shadows, spacing } from "../ui";

export type ToastTone = "info" | "success" | "error";
type Toast = { id: number; message: string; tone: ToastTone };

let current: Toast | null = null;
let counter = 0;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

export function showToast(message: string, tone: ToastTone = "info") {
  counter += 1;
  current = { id: counter, message, tone };
  emit();
}

export function dismissToast(id: number) {
  if (current?.id === id) {
    current = null;
    emit();
  }
}

function useToast(): Toast | null {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    () => current,
    () => current
  );
}

/** Mount once near the app root; shows one short-lived message at a time. */
export function ToastHost() {
  const toast = useToast();

  useEffect(() => {
    if (!toast) {
      return;
    }
    const timer = setTimeout(() => dismissToast(toast.id), 3500);
    return () => clearTimeout(timer);
  }, [toast]);

  if (!toast) {
    return null;
  }
  return (
    <View pointerEvents="none" style={styles.wrap}>
      <View
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        style={[styles.toast, toast.tone === "error" ? styles.error : toast.tone === "success" ? styles.success : null]}
        testID="toast"
      >
        <Text style={styles.text} tone="inverse" weight="semibold">
          {toast.message}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", top: spacing.xxxl, left: spacing.lg, right: spacing.lg, alignItems: "center", zIndex: 50 },
  toast: {
    backgroundColor: colors.text,
    borderRadius: radii.pill,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xl,
    maxWidth: 480,
    ...shadows.floating
  },
  success: { backgroundColor: colors.success },
  error: { backgroundColor: colors.danger },
  text: { textAlign: "center" }
});
