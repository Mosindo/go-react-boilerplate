import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import { Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, useTheme } from "../theme/theme";
import { Text } from "./Text";

type ToastKind = "info" | "error";
type ToastContextValue = { show: (message: string, kind?: ToastKind) => void };
const ToastContext = createContext<ToastContextValue>({ show: () => undefined });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<{ message: string; kind: ToastKind; id: number } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    if (timer.current) {
      clearTimeout(timer.current);
    }
    setToast({ message, kind, id: Date.now() });
    timer.current = setTimeout(() => setToast(null), 3500);
  }, []);

  useEffect(
    () => () => {
      if (timer.current) {
        clearTimeout(timer.current);
      }
    },
    []
  );

  return (
    <ToastContext.Provider value={{ show }}>
      {children}
      {toast ? (
        <Pressable
          accessibilityRole="alert"
          onPress={() => setToast(null)}
          style={[styles.toast, { top: insets.top + 8, backgroundColor: toast.kind === "error" ? colors.danger : colors.accent }]}
        >
          <Text variant="label" color="#FFFFFF">
            {toast.message}
          </Text>
        </Pressable>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  toast: { position: "absolute", left: 16, right: 16, padding: 14, borderRadius: radii.md, zIndex: 100, elevation: 10 }
});
