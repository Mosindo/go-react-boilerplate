import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Text } from "./Text";
import { radius, spacing, useColors } from "./theme";

type ToastKind = "info" | "error" | "success";
type ToastState = { id: number; message: string; kind: ToastKind } | null;
type ToastApi = { show: (message: string, kind?: ToastKind) => void };

const ToastContext = createContext<ToastApi>({ show: () => undefined });

export function useToast(): ToastApi {
  return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toast, setToast] = useState<ToastState>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const insets = useSafeAreaInsets();
  const c = useColors();
  const counter = useRef(0);

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    counter.current += 1;
    setToast({ id: counter.current, message, kind });
  }, []);

  useEffect(() => {
    if (!toast) return;
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const t = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 220, useNativeDriver: true }).start(() => setToast((cur) => (cur?.id === toast.id ? null : cur)));
    }, 3200);
    return () => clearTimeout(t);
  }, [toast, opacity]);

  const api = useMemo(() => ({ show }), [show]);
  const bg = toast?.kind === "error" ? c.danger : toast?.kind === "success" ? c.success : c.primary;

  return (
    <ToastContext.Provider value={api}>
      {children}
      {toast ? (
        <View pointerEvents="none" style={[styles.wrap, { bottom: insets.bottom + 90 }]}>
          <Animated.View accessibilityLiveRegion="polite" accessibilityRole="alert" style={[styles.toast, { backgroundColor: bg, opacity }]}>
            <Text variant="label" style={{ color: c.background }}>{toast.message}</Text>
          </Animated.View>
        </View>
      ) : null}
    </ToastContext.Provider>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.lg, right: spacing.lg, alignItems: "center" },
  toast: { borderRadius: radius.pill, paddingHorizontal: spacing.xl, paddingVertical: spacing.md, maxWidth: 420 },
});
