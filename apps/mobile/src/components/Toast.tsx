import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Platform, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import Ionicons from "@expo/vector-icons/Ionicons";
import { colors, radii, shadow, spacing } from "../theme/tokens";
import { AppText } from "./AppText";

type ToastTone = "info" | "success" | "error";
type ToastState = { id: number; message: string; tone: ToastTone } | null;

let state: ToastState = null;
let counter = 0;
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

export function showToast(message: string, tone: ToastTone = "info") {
  counter += 1;
  state = { id: counter, message, tone };
  emit();
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => {
    state = null;
    emit();
  }, 2800);
}

function dismiss() {
  state = null;
  emit();
}

const icons: Record<ToastTone, keyof typeof Ionicons.glyphMap> = {
  info: "sparkles-outline",
  success: "checkmark-circle-outline",
  error: "alert-circle-outline"
};

export function ToastHost() {
  const toast = useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => state
  );
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(opacity, { toValue: toast ? 1 : 0, duration: 180, useNativeDriver: Platform.OS !== "web" }).start();
  }, [toast, opacity]);

  if (!toast) return null;
  const tint = toast.tone === "error" ? colors.danger : toast.tone === "success" ? colors.success : colors.primary;
  return (
    <Animated.View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + spacing.sm, opacity }]}>
      <Pressable onPress={dismiss} style={styles.toast} accessibilityRole="alert" accessibilityLiveRegion="polite" testID="toast">
        <Ionicons name={icons[toast.tone]} size={20} color={tint} />
        <AppText style={styles.text}>{toast.message}</AppText>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.md, right: spacing.md, alignItems: "center", zIndex: 1000 },
  toast: {
    maxWidth: 520,
    width: "100%",
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.md,
    backgroundColor: colors.surfaceRaised,
    borderWidth: 1,
    borderColor: colors.border,
    ...shadow.card
  },
  text: { flex: 1 }
});
