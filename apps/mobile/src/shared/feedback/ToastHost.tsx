import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "../ui/Text";
import { dismissToast, useToasts } from "./store";

/** Floating, non-blocking feedback after actions (match, errors, saved…). */
export function ToastHost() {
  const toasts = useToasts();
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) return null;
  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.sm }]}>
      {toasts.map((toast) => {
        const color = toast.tone === "error" ? theme.danger : toast.tone === "success" ? theme.success : theme.accent;
        return (
          <Pressable
            key={toast.id}
            testID="toast"
            accessibilityRole="alert"
            onPress={() => dismissToast(toast.id)}
            style={[styles.toast, { backgroundColor: theme.surface, borderColor: color }]}
          >
            <Text variant="label" style={{ color }}>
              {toast.message}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: spacing.lg, right: spacing.lg, gap: spacing.sm, zIndex: 100 },
  toast: {
    borderWidth: 1,
    borderRadius: radius.md,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    shadowColor: "#000",
    shadowOpacity: 0.12,
    shadowRadius: 8,
    elevation: 4
  }
});
