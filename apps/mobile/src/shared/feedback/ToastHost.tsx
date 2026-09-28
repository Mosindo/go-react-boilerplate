import React, { useEffect, useRef } from "react";
import { Animated, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "../ui/Text";
import { dismissToast, useToasts, type Toast } from "./store";

const VISIBLE_MS = 4000;

function ToastItem({ toast }: { toast: Toast }) {
  const theme = useTheme();
  const progress = useRef(new Animated.Value(0)).current;
  useEffect(() => {
    Animated.timing(progress, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timer = setTimeout(() => dismissToast(toast.id), VISIBLE_MS);
    return () => clearTimeout(timer);
  }, [progress, toast.id]);
  const accent = toast.kind === "error" ? theme.danger : toast.kind === "success" ? theme.success : theme.accent;
  return (
    <Animated.View
      style={{
        opacity: progress,
        transform: [{ translateY: progress.interpolate({ inputRange: [0, 1], outputRange: [-12, 0] }) }]
      }}
    >
      <Pressable
        accessibilityHint="Dismisses this message"
        accessibilityLabel={toast.message}
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        onPress={() => dismissToast(toast.id)}
        style={[styles.toast, { backgroundColor: theme.surface, borderColor: accent }]}
      >
        <Text variant="label">{toast.message}</Text>
      </Pressable>
    </Animated.View>
  );
}

/** Mount once near the app root. */
export function ToastHost() {
  const toasts = useToasts();
  const insets = useSafeAreaInsets();
  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.sm }]}>
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: spacing.lg, right: spacing.lg, gap: spacing.sm, zIndex: 100 },
  toast: {
    minHeight: 44,
    justifyContent: "center",
    borderRadius: radius.md,
    borderWidth: 1,
    borderLeftWidth: 5,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    shadowColor: "#000",
    shadowOpacity: 0.15,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 3 },
    elevation: 4
  }
});
