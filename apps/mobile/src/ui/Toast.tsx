import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

type ToastKind = "info" | "error" | "success";
type ToastApi = { show: (message: string, kind?: ToastKind) => void };

const ToastContext = createContext<ToastApi>({ show: () => undefined });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<{ message: string; kind: ToastKind; id: number } | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    setToast({ message, kind, id: Date.now() });
  }, []);

  useEffect(() => {
    if (!toast) {
      return;
    }
    Animated.timing(opacity, { toValue: 1, duration: 160, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 200, useNativeDriver: true }).start(() => setToast(null));
    }, 3500);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  const api = useMemo(() => ({ show }), [show]);
  const background = toast?.kind === "error" ? colors.danger : toast?.kind === "success" ? colors.success : colors.text;

  return (
    <ToastContext.Provider value={api}>
      {children}
      {toast ? (
        <Animated.View
          accessibilityLiveRegion="polite"
          accessibilityRole="alert"
          pointerEvents="none"
          style={[styles.toast, { opacity, backgroundColor: background, bottom: insets.bottom + 90 }]}
        >
          <Text variant="label" style={{ color: colors.background }}>
            {toast.message}
          </Text>
        </Animated.View>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  toast: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    borderRadius: radii.md,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg
  }
});
