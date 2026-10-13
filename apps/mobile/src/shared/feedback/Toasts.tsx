import React, { useEffect } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../../theme";
import { Text, radii, spacing } from "../ui";
import { dismissToast, useToasts, type Toast } from "./store";

function ToastItem({ toast }: { toast: Toast }) {
  const { colors } = useTheme();

  useEffect(() => {
    const timer = setTimeout(() => dismissToast(toast.id), toast.kind === "error" ? 6000 : 4000);
    return () => clearTimeout(timer);
  }, [toast.id, toast.kind]);

  return (
    <Pressable
      accessibilityLabel={toast.message}
      accessibilityLiveRegion="polite"
      accessibilityRole={toast.kind === "error" ? "alert" : "text"}
      onPress={() => dismissToast(toast.id)}
      style={[styles.toast, { backgroundColor: toast.kind === "error" ? colors.danger : colors.accent }]}
    >
      <Text style={styles.text} variant="label" weight="semibold">
        {toast.message}
      </Text>
    </Pressable>
  );
}

/** Transient messages (errors from the network layer, "new match" alerts). */
export function Toasts() {
  const toasts = useToasts();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) {
    return null;
  }
  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.sm }]}>
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: spacing.lg, right: spacing.lg, gap: spacing.sm, alignItems: "center", zIndex: 100 },
  toast: { maxWidth: 520, width: "100%", borderRadius: radii.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  text: { color: "#FFFFFF" }
});
