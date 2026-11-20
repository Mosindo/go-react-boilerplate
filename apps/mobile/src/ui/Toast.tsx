import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme";
import { Text } from "./Text";

type ToastState = { id: number; message: string; tone: "info" | "error" | "success" } | null;

let current: ToastState = null;
let counter = 0;
const listeners = new Set<() => void>();

/** Show a short feedback message. Safe to call from anywhere (API client, hooks). */
export function showToast(message: string, tone: "info" | "error" | "success" = "info") {
  counter += 1;
  current = { id: counter, message, tone };
  listeners.forEach((l) => l());
}

function dismiss() {
  current = null;
  listeners.forEach((l) => l());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function ToastHost() {
  const toast = useSyncExternalStore(
    subscribe,
    () => current,
    () => current
  );
  const { colors, radii, spacing } = useTheme();
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!toast) {
      return;
    }
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 180, useNativeDriver: true }).start(() => dismiss());
    }, 3200);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  if (!toast) {
    return null;
  }
  const accent = toast.tone === "error" ? colors.danger : toast.tone === "success" ? colors.success : colors.primary;
  return (
    <Animated.View
      accessibilityLiveRegion="polite"
      pointerEvents="box-none"
      style={[styles.wrap, { top: insets.top + spacing.xs, opacity, paddingHorizontal: spacing.lg }]}
    >
      <Pressable
        accessibilityRole="alert"
        onPress={dismiss}
        style={[
          styles.toast,
          { backgroundColor: colors.surfaceRaised, borderColor: colors.border, borderLeftColor: accent, borderRadius: radii.md, padding: spacing.md }
        ]}
      >
        <Text variant="label">{toast.message}</Text>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: 0, right: 0, zIndex: 100 },
  toast: { borderWidth: 1, borderLeftWidth: 4, shadowOpacity: 0.2, shadowRadius: 12, shadowOffset: { width: 0, height: 6 }, elevation: 6 }
});
