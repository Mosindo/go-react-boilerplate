import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, shadow, spacing } from "../theme";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; message: string; kind: ToastKind } | null;

let state: ToastState = null;
let nextId = 1;
const listeners = new Set<() => void>();
let hideTimer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

/** Shows a short, non-blocking feedback message. */
export function showToast(message: string, kind: ToastKind = "info") {
  state = { id: nextId++, message, kind };
  emit();
  if (hideTimer) {
    clearTimeout(hideTimer);
  }
  hideTimer = setTimeout(() => {
    state = null;
    emit();
  }, 3200);
}

function useToast() {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => {
        listeners.delete(l);
      };
    },
    () => state,
    () => state
  );
}

const icons: Record<ToastKind, keyof typeof Ionicons.glyphMap> = {
  info: "information-circle",
  success: "checkmark-circle",
  error: "alert-circle"
};

const tint: Record<ToastKind, string> = {
  info: colors.gold,
  success: colors.success,
  error: colors.danger
};

export function ToastHost() {
  const toast = useToast();
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(opacity, { toValue: toast ? 1 : 0, duration: 180, useNativeDriver: true }).start();
  }, [toast, opacity]);

  if (!toast) {
    return null;
  }
  return (
    <View pointerEvents="none" style={[styles.host, { top: insets.top + spacing.sm }]}>
      <Animated.View style={[styles.toast, { opacity }]} accessibilityLiveRegion="polite" accessibilityRole="alert" testID="toast">
        <Ionicons name={icons[toast.kind]} size={20} color={tint[toast.kind]} />
        <Text variant="small" style={styles.message}>
          {toast.message}
        </Text>
      </Animated.View>
    </View>
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
    maxWidth: 480,
    width: "100%",
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.lg,
    backgroundColor: colors.surfaceHigh,
    borderWidth: 1,
    borderColor: colors.borderStrong,
    ...shadow.card
  },
  message: {
    flex: 1
  }
});
