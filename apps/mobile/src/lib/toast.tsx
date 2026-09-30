import React, { useEffect, useRef, useSyncExternalStore } from "react";
import { AccessibilityInfo, Animated, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../design/ThemeProvider";
import { Text } from "../design/components/Text";

type Toast = { id: number; message: string; tone: "info" | "error" | "success" };

let current: Toast | null = null;
let seq = 0;
const listeners = new Set<() => void>();
let hideTimer: ReturnType<typeof setTimeout> | null = null;

function emit() {
  listeners.forEach((l) => l());
}

export function showToast(message: string, tone: Toast["tone"] = "info") {
  current = { id: ++seq, message, tone };
  AccessibilityInfo.announceForAccessibility?.(message);
  emit();
  if (hideTimer) clearTimeout(hideTimer);
  hideTimer = setTimeout(() => {
    current = null;
    emit();
  }, 3500);
}

function useToast(): Toast | null {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => current
  );
}

export function ToastHost() {
  const toast = useToast();
  const { colors, radii } = useTheme();
  const insets = useSafeAreaInsets();
  const anim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.timing(anim, { toValue: toast ? 1 : 0, duration: 180, useNativeDriver: true }).start();
  }, [anim, toast]);

  if (!toast) {
    return null;
  }
  const icon = toast.tone === "error" ? "alert-circle" : toast.tone === "success" ? "checkmark-circle" : "information-circle";
  const tint = toast.tone === "error" ? colors.danger : toast.tone === "success" ? colors.success : colors.primary;
  return (
    <View pointerEvents="none" style={[styles.host, { top: insets.top + 8 }]}>
      <Animated.View
        accessibilityLiveRegion="polite"
        style={[
          styles.toast,
          {
            backgroundColor: colors.surfaceRaised,
            borderColor: colors.border,
            borderRadius: radii.lg,
            opacity: anim,
            transform: [{ translateY: anim.interpolate({ inputRange: [0, 1], outputRange: [-12, 0] }) }]
          }
        ]}
        testID="toast"
      >
        <Ionicons color={tint} name={icon} size={20} />
        <Text style={styles.text} variant="label">
          {toast.message}
        </Text>
      </Animated.View>
    </View>
  );
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: 16, right: 16, alignItems: "center", zIndex: 1000 },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderWidth: 1,
    maxWidth: 480,
    shadowColor: "#000",
    shadowOpacity: 0.15,
    shadowRadius: 16,
    shadowOffset: { width: 0, height: 8 },
    elevation: 6
  },
  text: { flexShrink: 1 }
});
