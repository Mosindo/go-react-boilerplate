import React, { useEffect, useRef } from "react";
import { Animated, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Text, colors, radii, shadows, spacing } from "../ui";
import { clearGlobalError, dismissToast, useGlobalFeedback } from "./store";

/** Renders the global error banner and toasts above the app. */
export function FeedbackHost() {
  const { error, toast } = useGlobalFeedback();
  const insets = useSafeAreaInsets();
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!toast) {
      return;
    }
    opacity.setValue(0);
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const id = toast.id;
    const timer = setTimeout(() => dismissToast(id), 2800);
    return () => clearTimeout(timer);
  }, [opacity, toast]);

  useEffect(() => {
    if (!error) {
      return;
    }
    const timer = setTimeout(clearGlobalError, 6000);
    return () => clearTimeout(timer);
  }, [error]);

  return (
    <View pointerEvents="box-none" style={[StyleSheet.absoluteFill, { paddingTop: insets.top + spacing.sm }]}>
      {error ? (
        <Pressable accessibilityRole="alert" onPress={clearGlobalError} style={[styles.banner, styles.error]} testID="global-error">
          <Text variant="label" weight="semibold">
            {error}
          </Text>
          <Text tone="muted" variant="caption">
            Touchez pour fermer
          </Text>
        </Pressable>
      ) : null}
      {toast ? (
        <Animated.View style={[styles.toastWrap, { opacity }]}>
          <View
            accessibilityLiveRegion="polite"
            style={[styles.banner, toast.tone === "danger" ? styles.error : toast.tone === "success" ? styles.success : null]}
            testID="toast"
          >
            <Text variant="label" weight="semibold">
              {toast.message}
            </Text>
          </View>
        </Animated.View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  banner: {
    marginHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    borderRadius: radii.md,
    backgroundColor: colors.surfaceRaised,
    borderWidth: 1,
    borderColor: colors.border,
    gap: 2,
    ...shadows.floating
  },
  error: {
    borderColor: colors.dangerBorder,
    backgroundColor: "#2B1C1A"
  },
  success: {
    borderColor: colors.primaryBorder
  },
  toastWrap: {
    alignItems: "center",
    marginTop: spacing.sm
  }
});
