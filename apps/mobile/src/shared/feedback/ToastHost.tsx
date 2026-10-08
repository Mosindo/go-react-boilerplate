import React, { useEffect } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Notice } from "../ui";
import { dismissToast, useToasts, type Toast } from "./store";
import { spacing } from "../ui";

function ToastRow({ toast }: { toast: Toast }) {
  useEffect(() => {
    const timer = setTimeout(() => dismissToast(toast.id), toast.tone === "error" ? 6000 : 3500);
    return () => clearTimeout(timer);
  }, [toast.id, toast.tone]);
  return (
    <Pressable accessibilityHint="Appuyez pour fermer" onPress={() => dismissToast(toast.id)}>
      <Notice message={toast.message} tone={toast.tone} />
    </Pressable>
  );
}

/** Renders transient feedback above the tab bar; mount once at the app root. */
export function ToastHost() {
  const toasts = useToasts();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) return null;
  return (
    <View pointerEvents="box-none" style={[styles.host, { bottom: insets.bottom + 72 }]}>
      {toasts.map((toast) => (
        <ToastRow key={toast.id} toast={toast} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: spacing.lg, right: spacing.lg, gap: spacing.sm, alignSelf: "center", maxWidth: 520 }
});
