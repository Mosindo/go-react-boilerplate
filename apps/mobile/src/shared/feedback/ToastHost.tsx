import React, { useEffect } from "react";
import { Animated, Pressable, StyleSheet, useAnimatedValue } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "../ui/Text";
import { dismissToast, useToast } from "./toast";

const DISPLAY_MS = 4000;

export function ToastHost() {
  const toast = useToast();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const opacity = useAnimatedValue(0);

  useEffect(() => {
    if (!toast) {
      return undefined;
    }
    opacity.setValue(0);
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timeoutId = setTimeout(dismissToast, DISPLAY_MS);
    return () => clearTimeout(timeoutId);
  }, [toast, opacity]);

  if (!toast) {
    return null;
  }

  return (
    <Animated.View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + spacing.sm, opacity }]}>
      <Pressable
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        onPress={dismissToast}
        style={[styles.toast, { backgroundColor: toast.kind === "error" ? colors.danger : colors.text }]}
        testID="toast"
      >
        <Text style={{ color: toast.kind === "error" ? colors.onDanger : colors.background }} variant="label">
          {toast.message}
        </Text>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.lg, right: spacing.lg },
  toast: {
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    minHeight: 44,
    justifyContent: "center"
  }
});
