import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { radii, shadow, spacing } from "../theme/palette";
import { Icon } from "./Icon";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; message: string; kind: ToastKind } | null;

let state: ToastState = null;
let seq = 0;
const listeners = new Set<() => void>();

/** Shows a short, non-blocking feedback message. */
export function toast(message: string, kind: ToastKind = "info") {
  state = { id: ++seq, message, kind };
  listeners.forEach((l) => l());
}

function dismiss(id: number) {
  if (state?.id === id) {
    state = null;
    listeners.forEach((l) => l());
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function ToastHost() {
  const current = useSyncExternalStore(subscribe, () => state, () => state);
  const { colors, scheme } = useTheme();
  const insets = useSafeAreaInsets();
  const anim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!current) {
      return;
    }
    anim.setValue(0);
    Animated.spring(anim, { toValue: 1, useNativeDriver: true, friction: 8 }).start();
    const timer = setTimeout(() => dismiss(current.id), current.kind === "error" ? 5000 : 3000);
    return () => clearTimeout(timer);
  }, [current, anim]);

  if (!current) {
    return null;
  }
  const accent = current.kind === "error" ? colors.danger : current.kind === "success" ? colors.success : colors.primary;
  const icon = current.kind === "error" ? "alert-circle" : current.kind === "success" ? "checkmark-circle" : "information-circle";

  return (
    <Animated.View
      pointerEvents="box-none"
      style={[
        styles.wrap,
        { top: insets.top + spacing.xs },
        { opacity: anim, transform: [{ translateY: anim.interpolate({ inputRange: [0, 1], outputRange: [-20, 0] }) }] }
      ]}
    >
      <Pressable
        onPress={() => dismiss(current.id)}
        accessibilityRole="alert"
        accessibilityLiveRegion="assertive"
        style={[styles.toast, { backgroundColor: colors.surfaceRaised, borderColor: colors.border }, shadow(scheme, 2)]}
      >
        <Icon name={icon} size={20} color={accent} />
        <Text variant="label" style={styles.flex}>
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
    borderWidth: StyleSheet.hairlineWidth,
    maxWidth: 520,
    width: "100%"
  },
  flex: { flex: 1 }
});
