import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AccessibilityInfo, Animated, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type ToastTone = "default" | "error" | "success";
type ToastState = { id: number; message: string; tone: ToastTone } | null;
type ToastApi = { show: (message: string, tone?: ToastTone) => void };

const ToastContext = createContext<ToastApi | null>(null);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const { colors, radii, spacing } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<ToastState>(null);
  const opacity = useRef(new Animated.Value(0)).current;

  const show = useCallback((message: string, tone: ToastTone = "default") => {
    setToast({ id: Date.now(), message, tone });
    AccessibilityInfo.announceForAccessibility(message);
  }, []);

  useEffect(() => {
    if (!toast) {
      return;
    }
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 220, useNativeDriver: true }).start(() => setToast(null));
    }, 3200);
    return () => clearTimeout(timer);
  }, [opacity, toast]);

  const api = useMemo(() => ({ show }), [show]);
  const bg = toast?.tone === "error" ? colors.danger : toast?.tone === "success" ? colors.accent : colors.text;

  return (
    <ToastContext.Provider value={api}>
      {children}
      {toast ? (
        <Animated.View
          pointerEvents="none"
          testID="toast"
          style={[
            styles.toast,
            { opacity, top: insets.top + spacing.sm, backgroundColor: bg, borderRadius: radii.lg, padding: spacing.md }
          ]}
        >
          <Text weight="medium" style={{ color: colors.background }}>
            {toast.message}
          </Text>
        </Animated.View>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return ctx;
}

const styles = StyleSheet.create({
  toast: { position: "absolute", left: 16, right: 16, maxWidth: 560, alignSelf: "center", zIndex: 1000, elevation: 10 }
});
