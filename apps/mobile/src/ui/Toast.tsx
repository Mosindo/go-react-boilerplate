import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, shadow, spacing } from "../theme";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error" | "match";
type ToastState = { id: number; message: string; kind: ToastKind; onPress?: () => void } | null;

let current: ToastState = null;
let counter = 0;
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

export function showToast(message: string, kind: ToastKind = "info", onPress?: () => void) {
  counter += 1;
  current = { id: counter, message, kind, onPress };
  emit();
  if (timer) clearTimeout(timer);
  timer = setTimeout(
    () => {
      current = null;
      emit();
    },
    kind === "error" ? 5000 : 3200
  );
}

export function dismissToast() {
  current = null;
  emit();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

const icons: Record<ToastKind, React.ComponentProps<typeof Ionicons>["name"]> = {
  info: "information-circle",
  success: "checkmark-circle",
  error: "alert-circle",
  match: "heart"
};

const iconColors: Record<ToastKind, string> = {
  info: colors.textMuted,
  success: colors.success,
  error: colors.danger,
  match: colors.accent
};

export function ToastHost() {
  const toast = useSyncExternalStore(
    subscribe,
    () => current,
    () => current
  );
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (toast) {
      opacity.setValue(0);
      Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    }
  }, [toast, opacity]);

  if (!toast) return null;
  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.sm }]}>
      <Animated.View style={{ opacity, transform: [{ translateY: opacity.interpolate({ inputRange: [0, 1], outputRange: [-12, 0] }) }] }}>
        <Pressable
          accessibilityRole="alert"
          accessibilityLiveRegion="polite"
          testID="toast"
          onPress={() => {
            toast.onPress?.();
            dismissToast();
          }}
          style={styles.toast}
        >
          <Ionicons name={icons[toast.kind]} size={20} color={iconColors[toast.kind]} />
          <Text variant="label" style={styles.message}>
            {toast.message}
          </Text>
        </Pressable>
      </Animated.View>
    </View>
  );
}

const styles = StyleSheet.create({
  host: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    alignItems: "center",
    zIndex: 1000
  },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    maxWidth: 520,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surfaceRaised,
    borderWidth: 1,
    borderColor: colors.borderStrong,
    ...shadow.card
  },
  message: {
    flexShrink: 1
  }
});
