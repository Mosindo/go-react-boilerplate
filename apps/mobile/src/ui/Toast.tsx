import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

type Tone = "info" | "success" | "error";
type ToastState = { id: number; message: string; tone: Tone } | null;

let state: ToastState = null;
let seq = 0;
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

/** Shows a short, non-blocking feedback message. */
export function showToast(message: string, tone: Tone = "info") {
  state = { id: ++seq, message, tone };
  emit();
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => {
    state = null;
    emit();
  }, 3500);
}

function useToast(): ToastState {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => state,
    () => state
  );
}

export function ToastHost() {
  const toast = useToast();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(opacity, { toValue: toast ? 1 : 0, duration: 180, useNativeDriver: true }).start();
  }, [toast, opacity]);

  if (!toast) return null;
  const icon = toast.tone === "success" ? "checkmark-circle" : toast.tone === "error" ? "alert-circle" : "information-circle";
  const tint = toast.tone === "success" ? colors.success : toast.tone === "error" ? colors.danger : colors.accent;
  return (
    <View pointerEvents="none" style={[styles.wrap, { top: insets.top + spacing.sm }]}>
      <Animated.View
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        style={[styles.toast, { opacity, backgroundColor: colors.surfaceRaised, borderColor: colors.border, shadowColor: colors.shadow }]}
        testID="toast"
      >
        <Icon color={tint} name={icon} size={20} />
        <Text style={styles.text} variant="label">
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
    maxWidth: 520,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.lg,
    borderWidth: 1,
    shadowOpacity: 0.15,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 6 },
    elevation: 6
  },
  text: { flexShrink: 1 }
});
