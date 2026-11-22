import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Platform, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { Text } from "./Text";
import { colors, radii, spacing } from "./theme";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; kind: ToastKind; message: string } | null;

let current: ToastState = null;
let seq = 0;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

/** Shows a short, non-blocking message. Safe to call from non-React code. */
export function showToast(message: string, kind: ToastKind = "info") {
  current = { id: ++seq, kind, message };
  emit();
}

export function dismissToast() {
  current = null;
  emit();
}

const icons: Record<ToastKind, keyof typeof Ionicons.glyphMap> = {
  info: "information-circle",
  success: "checkmark-circle",
  error: "alert-circle"
};

const tint: Record<ToastKind, string> = {
  info: colors.accent,
  success: colors.like,
  error: colors.danger
};

export function ToastHost() {
  const toast = useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
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
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: Platform.OS !== "web" }).start();
    const timer = setTimeout(dismissToast, toast.kind === "error" ? 5000 : 3000);
    return () => clearTimeout(timer);
  }, [opacity, toast]);

  if (!toast) {
    return null;
  }
  return (
    <Animated.View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.sm, opacity }]}>
      <Pressable accessibilityLiveRegion="polite" accessibilityRole="alert" onPress={dismissToast} style={styles.toast} testID="toast">
        <Ionicons color={tint[toast.kind]} name={icons[toast.kind]} size={20} />
        <Text style={styles.text} variant="caption">
          {toast.message}
        </Text>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  host: {
    position: "absolute",
    left: spacing.md,
    right: spacing.md,
    alignItems: "center",
    zIndex: 1000
  },
  toast: {
    maxWidth: 520,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.md,
    backgroundColor: colors.surfaceHigh,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.borderStrong
  },
  text: {
    flexShrink: 1
  }
});
