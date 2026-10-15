import React from "react";
import { Pressable, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { elevation, useTheme } from "../../theme";
import { Text, radii, spacing } from "../ui";
import { dismissToast, useToasts } from "./store";

/** Renders the current toast above the navigation. Mount it once, at the app root. */
export function ToastHost() {
  const palette = useTheme();
  const insets = useSafeAreaInsets();
  const toast = useToasts()[0];

  if (!toast) {
    return null;
  }

  const background = toast.tone === "danger" ? palette.danger : toast.tone === "success" ? palette.success : palette.brand;
  const foreground = palette.isDark ? "#1B1017" : "#FFFFFF";

  return (
    <View
      pointerEvents="box-none"
      style={{ position: "absolute", left: 0, right: 0, top: insets.top + spacing.sm, paddingHorizontal: spacing.lg }}
    >
      <Pressable
        accessibilityLiveRegion="assertive"
        accessibilityRole="alert"
        onPress={() => dismissToast(toast.id)}
        style={[
          {
            backgroundColor: background,
            borderRadius: radii.md,
            paddingVertical: spacing.md,
            paddingHorizontal: spacing.lg,
            flexDirection: "row",
            alignItems: "center",
            gap: spacing.md
          },
          elevation(palette, 2)
        ]}
      >
        <Text style={{ color: foreground, flex: 1 }} variant="label" weight="medium">
          {toast.message}
        </Text>
        {toast.actionLabel && toast.onAction ? (
          <Pressable
            accessibilityRole="button"
            hitSlop={10}
            onPress={() => {
              dismissToast(toast.id);
              toast.onAction?.();
            }}
          >
            <Text style={{ color: foreground, textDecorationLine: "underline" }} variant="label" weight="bold">
              {toast.actionLabel}
            </Text>
          </Pressable>
        ) : null}
      </Pressable>
    </View>
  );
}
