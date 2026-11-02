import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, Pressable, StyleSheet, Text } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { fontSizes, radii, spacing, useTheme } from "../theme/theme";

type Kind = "info" | "error" | "success";
type ToastValue = { show: (message: string, kind?: Kind) => void };

const ToastContext = createContext<ToastValue>({ show: () => undefined });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<{ message: string; kind: Kind; id: number } | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;

  const show = useCallback((message: string, kind: Kind = "info") => {
    setToast({ message, kind, id: Date.now() });
  }, []);

  useEffect(() => {
    if (!toast) return;
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 220, useNativeDriver: true }).start(() => setToast(null));
    }, 3200);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  const value = useMemo(() => ({ show }), [show]);
  const background = toast?.kind === "error" ? colors.danger : toast?.kind === "success" ? colors.success : colors.text;

  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast ? (
        <Animated.View
          accessibilityLiveRegion="polite"
          pointerEvents="box-none"
          style={[styles.wrap, { top: insets.top + spacing.sm, opacity }]}
        >
          <Pressable accessibilityRole="alert" onPress={() => setToast(null)} style={[styles.toast, { backgroundColor: background }]}>
            <Text style={[styles.text, { color: colors.background }]}>{toast.message}</Text>
          </Pressable>
        </Animated.View>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastValue {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.lg, right: spacing.lg, alignItems: "center", zIndex: 100 },
  toast: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md, borderRadius: radii.md, maxWidth: 480 },
  text: { fontSize: fontSizes.sm, fontWeight: "600", textAlign: "center" }
});
