import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../theme/theme";
import { Text } from "./Text";

type ToastKind = "info" | "error" | "success";
type ToastApi = { show: (message: string, kind?: ToastKind) => void };

const ToastContext = createContext<ToastApi>({ show: () => undefined });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const [toast, setToast] = useState<{ id: number; message: string; kind: ToastKind } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    if (timer.current) clearTimeout(timer.current);
    setToast({ id: Date.now(), message, kind });
    timer.current = setTimeout(() => setToast(null), 3500);
  }, []);

  const api = useMemo(() => ({ show }), [show]);
  const bg =
    toast?.kind === "error" ? colors.danger : toast?.kind === "success" ? colors.success : colors.text;

  return (
    <ToastContext.Provider value={api}>
      {children}
      {toast ? (
        <SafeAreaView pointerEvents="none" edges={["top"]} style={styles.host}>
          <View
            style={[styles.toast, { backgroundColor: bg }]}
            accessibilityLiveRegion="polite"
            accessibilityRole="alert"
          >
            <Text variant="label" style={{ color: colors.background }}>
              {toast.message}
            </Text>
          </View>
        </SafeAreaView>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  host: {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    alignItems: "center",
    paddingHorizontal: spacing.md
  },
  toast: { borderRadius: radii.md, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, maxWidth: 480 }
});
