import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { spacing, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; message: string; kind: ToastKind } | null;

let current: ToastState = null;
let seq = 0;
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

/** Lightweight global feedback (no provider needed, callable from anywhere). */
export function showToast(message: string, kind: ToastKind = "info") {
  current = { id: ++seq, message, kind };
  emit();
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => {
    current = null;
    emit();
  }, 3500);
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

export function ToastHost() {
  const toast = useSyncExternalStore(
    subscribe,
    () => current,
    () => current
  );
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(opacity, { toValue: toast ? 1 : 0, duration: 180, useNativeDriver: true }).start();
  }, [toast, opacity]);

  if (!toast) {
    return null;
  }
  const tint = toast.kind === "error" ? colors.danger : toast.kind === "success" ? colors.success : colors.text;
  return (
    <View pointerEvents="none" style={[styles.wrap, { top: insets.top + spacing.xs }]}>
      <Animated.View
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        style={[styles.toast, { opacity, backgroundColor: colors.surfaceRaised, borderColor: colors.border }]}
      >
        <Icon name={toast.kind === "error" ? "alert-circle" : toast.kind === "success" ? "checkmark-circle" : "information-circle"} size={20} color={tint} />
        <Text variant="label" style={styles.text}>
          {toast.message}
        </Text>
      </Animated.View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.md, right: spacing.md, alignItems: "center" },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: 16,
    borderWidth: 1,
    maxWidth: 480,
    shadowColor: "#000",
    shadowOpacity: 0.12,
    shadowRadius: 16,
    shadowOffset: { width: 0, height: 6 },
    elevation: 6
  },
  text: { flexShrink: 1 }
});
