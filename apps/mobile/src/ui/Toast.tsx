import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, shadows, spacing } from "../theme/tokens";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; message: string; kind: ToastKind } | null;

let current: ToastState = null;
let seq = 0;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

/** Shows a short, non-blocking message. Safe to call from anywhere. */
export function showToast(message: string, kind: ToastKind = "info") {
  seq += 1;
  current = { id: seq, message, kind };
  emit();
}

function dismiss(id: number) {
  if (current?.id === id) {
    current = null;
    emit();
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

const icons: Record<ToastKind, keyof typeof Ionicons.glyphMap> = {
  info: "information-circle",
  success: "checkmark-circle",
  error: "alert-circle"
};

const tint: Record<ToastKind, string> = { info: colors.accent, success: colors.success, error: colors.danger };

export function ToastHost() {
  const toast = useSyncExternalStore(
    subscribe,
    () => current,
    () => current
  );
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!toast) {
      return;
    }
    opacity.setValue(0);
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timer = setTimeout(() => dismiss(toast.id), toast.kind === "error" ? 4500 : 3000);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  if (!toast) {
    return null;
  }
  return (
    <Animated.View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + spacing.sm, opacity }]}>
      <Pressable
        onPress={() => dismiss(toast.id)}
        style={[styles.toast, shadows.floating]}
        accessibilityRole="alert"
        accessibilityLiveRegion="polite"
        testID="toast"
      >
        <Ionicons name={icons[toast.kind]} size={20} color={tint[toast.kind]} />
        <Text variant="caption" style={styles.text}>
          {toast.message}
        </Text>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.lg, right: spacing.lg, alignItems: "center", zIndex: 1000 },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    backgroundColor: colors.surfaceRaised,
    borderColor: colors.border,
    borderWidth: 1,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    maxWidth: 520
  },
  text: { flexShrink: 1, color: colors.text, fontSize: 14 }
});
