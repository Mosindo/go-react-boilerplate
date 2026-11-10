import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { Animated, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type ToastTone = "info" | "success" | "error";
type ToastState = { id: number; message: string; tone: ToastTone } | null;

let current: ToastState = null;
let seq = 0;
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

/** Shows a short, non-blocking feedback message. */
export function showToast(message: string, tone: ToastTone = "info") {
  current = { id: ++seq, message, tone };
  emit();
  if (timer) clearTimeout(timer);
  timer = setTimeout(hideToast, tone === "error" ? 5000 : 3000);
}

export function hideToast() {
  current = null;
  emit();
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
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  const anim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(anim, { toValue: toast ? 1 : 0, duration: 180, useNativeDriver: true }).start();
  }, [anim, toast]);

  if (!toast) return null;
  const icon = toast.tone === "success" ? "checkmark-circle" : toast.tone === "error" ? "alert-circle" : "information-circle";
  const color = toast.tone === "success" ? theme.colors.success : toast.tone === "error" ? theme.colors.danger : theme.colors.accent;
  return (
    <Animated.View
      pointerEvents="box-none"
      style={[
        styles.wrap,
        { top: insets.top + 8, opacity: anim, transform: [{ translateY: anim.interpolate({ inputRange: [0, 1], outputRange: [-12, 0] }) }] }
      ]}
    >
      <Pressable
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        onPress={hideToast}
        style={[styles.toast, theme.shadow(3), { backgroundColor: theme.colors.surfaceRaised, borderColor: theme.colors.border }]}
      >
        <Ionicons color={color} name={icon} size={20} />
        <Text style={styles.text} variant="small">
          {toast.message}
        </Text>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: 16, right: 16, alignItems: "center", zIndex: 1000 },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderRadius: 16,
    borderWidth: 1,
    maxWidth: 520,
    width: "100%"
  },
  text: { flex: 1 }
});
