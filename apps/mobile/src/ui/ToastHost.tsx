import React from "react";
import { Pressable, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { makeStyles } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";
import { dismissToast, useToasts } from "./toast";

export function ToastHost() {
  const toasts = useToasts();
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) {
    return null;
  }
  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + 8 }]}>
      {toasts.map((toast) => (
        <Pressable
          key={toast.id}
          accessibilityRole="alert"
          accessibilityLiveRegion="polite"
          onPress={() => {
            toast.onPress?.();
            dismissToast(toast.id);
          }}
          style={[styles.toast, toast.tone === "error" ? styles.error : toast.tone === "success" ? styles.success : null]}
        >
          <Icon
            name={toast.tone === "error" ? "alert-circle" : toast.tone === "success" ? "checkmark-circle" : "sparkles"}
            size={18}
            color={toast.tone === "error" ? "danger" : toast.tone === "success" ? "success" : "primary"}
          />
          <Text variant="label" style={styles.text}>
            {toast.message}
          </Text>
        </Pressable>
      ))}
    </View>
  );
}

const useStyles = makeStyles((t) => ({
  host: { position: "absolute", left: t.spacing.lg, right: t.spacing.lg, gap: t.spacing.sm, alignItems: "center", zIndex: 1000 },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: t.spacing.sm,
    maxWidth: 520,
    width: "100%",
    paddingHorizontal: t.spacing.lg,
    paddingVertical: t.spacing.md,
    borderRadius: t.radii.lg,
    backgroundColor: t.colors.surfaceRaised,
    borderWidth: 1,
    borderColor: t.colors.border,
    shadowColor: t.colors.shadow,
    shadowOpacity: 0.15,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 8 },
    elevation: 6
  },
  error: { borderColor: t.colors.danger },
  success: { borderColor: t.colors.success },
  text: { flex: 1 }
}));
