import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

type Tone = "info" | "success" | "error";
type ToastState = { id: number; message: string; tone: Tone } | null;

let state: ToastState = null;
let counter = 0;
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

/** Shows a short, non-blocking feedback message. */
export function toast(message: string, tone: Tone = "info") {
  counter += 1;
  state = { id: counter, message, tone };
  emit();
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => {
    state = null;
    emit();
  }, 3200);
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function ToastHost() {
  const current = useSyncExternalStore(
    subscribe,
    () => state,
    () => state
  );
  const insets = useSafeAreaInsets();
  const { colors } = useTheme();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(opacity, { toValue: current ? 1 : 0, duration: 180, useNativeDriver: true }).start();
  }, [current, opacity]);

  if (!current) return null;
  const icon = current.tone === "success" ? "checkmark-circle" : current.tone === "error" ? "alert-circle" : "information-circle";
  const color = current.tone === "success" ? "success" : current.tone === "error" ? "danger" : "primary";
  return (
    <Animated.View
      pointerEvents="box-none"
      style={[styles.wrap, { top: insets.top + spacing.sm, opacity }]}
      accessibilityLiveRegion="polite"
    >
      <Pressable
        onPress={() => {
          state = null;
          emit();
        }}
        accessibilityRole="alert"
        style={[styles.toast, { backgroundColor: colors.surfaceRaised, borderColor: colors.border }]}
      >
        <Icon name={icon} size={20} color={color} />
        <Text variant="label" style={styles.text}>
          {current.message}
        </Text>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.md, right: spacing.md, alignItems: "center", zIndex: 1000 },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.lg,
    borderWidth: 1,
    maxWidth: 480,
    shadowColor: "#000",
    shadowOpacity: 0.25,
    shadowRadius: 16,
    shadowOffset: { width: 0, height: 8 },
    elevation: 8
  },
  text: { flexShrink: 1 }
});
