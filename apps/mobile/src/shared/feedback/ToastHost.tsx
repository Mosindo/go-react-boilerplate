import React, { useEffect } from "react";
import { StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Notice } from "../ui";
import { dismissToast, useToasts, type Toast } from "./store";

function ToastItem({ toast }: { toast: Toast }) {
  useEffect(() => {
    const timer = setTimeout(() => dismissToast(toast.id), 4000);
    return () => clearTimeout(timer);
  }, [toast.id]);
  return <Notice message={toast.message} tone={toast.tone === "danger" ? "danger" : toast.tone === "success" ? "success" : "info"} />;
}

export function ToastHost() {
  const toasts = useToasts();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) {
    return null;
  }
  return (
    <View pointerEvents="none" style={[styles.host, { bottom: insets.bottom + 84 }]}>
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: 16, right: 16, gap: 8 }
});
