import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radius, spacing, useTheme } from "../theme/theme";
import { Text } from "./Text";

type ToastInput = { message: string; kind?: "info" | "success" | "error"; onPress?: () => void };
type ToastValue = { show: (toast: ToastInput) => void };

const ToastContext = createContext<ToastValue>({ show: () => undefined });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<(ToastInput & { id: number }) | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const show = useCallback((input: ToastInput) => {
    if (timer.current) clearTimeout(timer.current);
    setToast({ ...input, id: Date.now() });
    timer.current = setTimeout(() => setToast(null), 4000);
  }, []);

  const value = useMemo(() => ({ show }), [show]);
  const bg =
    toast?.kind === "error"
      ? colors.danger
      : toast?.kind === "success"
        ? colors.success
        : colors.text;

  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast ? (
        <View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + spacing.sm }]}>
          <Pressable
            testID="toast"
            accessibilityRole="alert"
            onPress={() => {
              toast.onPress?.();
              setToast(null);
            }}
            style={[styles.toast, { backgroundColor: bg }]}
          >
            <Text variant="label" color={colors.bg}>
              {toast.message}
            </Text>
          </Pressable>
        </View>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastValue {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  wrap: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    alignItems: "center",
    zIndex: 1000,
  },
  toast: {
    borderRadius: radius.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    maxWidth: 520,
  },
});
