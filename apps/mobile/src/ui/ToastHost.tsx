import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useColors } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";
import { dismissToast, useToasts } from "./toast";

export function ToastHost() {
  const toasts = useToasts();
  const colors = useColors();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) return null;

  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.xs }]}>
      {toasts.map((item) => {
        const accent =
          item.kind === "error" ? colors.danger : item.kind === "success" ? colors.success : colors.primary;
        return (
          <Pressable
            key={item.id}
            accessibilityLabel={`${item.message}. Tap to dismiss`}
            accessibilityLiveRegion="polite"
            accessibilityRole="alert"
            onPress={() => dismissToast(item.id)}
            style={[styles.toast, { backgroundColor: colors.surface, borderColor: accent }]}
          >
            <Icon
              color={accent}
              name={
                item.kind === "error"
                  ? "alert-circle"
                  : item.kind === "success"
                    ? "checkmark-circle"
                    : "information-circle"
              }
              size={22}
            />
            <Text style={styles.message} variant="small" weight="medium">
              {item.message}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  host: {
    position: "absolute",
    left: spacing.md,
    right: spacing.md,
    gap: spacing.xs,
    alignItems: "center",
    zIndex: 1000
  },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    maxWidth: 520,
    width: "100%",
    minHeight: 48,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderRadius: radii.md,
    borderWidth: 1,
    shadowColor: "#000",
    shadowOpacity: 0.15,
    shadowRadius: 10,
    shadowOffset: { width: 0, height: 4 },
    elevation: 6
  },
  message: { flex: 1 }
});
