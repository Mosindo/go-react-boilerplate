import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode
} from "react";
import { Animated, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { Text } from "./Text";
import { radii, spacing, useTheme } from "./theme";

type Kind = "info" | "error";
type ToastContextValue = { show: (message: string, kind?: Kind) => void };

const ToastContext = createContext<ToastContextValue>({ show: () => undefined });

export function useToast(): ToastContextValue {
  return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<{ message: string; kind: Kind; id: number } | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);

  const show = useCallback((message: string, kind: Kind = "info") => {
    counter.current += 1;
    setToast({ message, kind, id: counter.current });
  }, []);

  useEffect(() => {
    if (!toast) {
      return undefined;
    }
    Animated.timing(opacity, { toValue: 1, duration: 160, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 200, useNativeDriver: true }).start(({ finished }) => {
        if (finished) {
          setToast((current) => (current?.id === toast.id ? null : current));
        }
      });
    }, 3200);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  const value = useMemo(() => ({ show }), [show]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast ? (
        <Animated.View
          accessibilityLiveRegion="polite"
          pointerEvents="none"
          style={[
            styles.toast,
            {
              opacity,
              top: insets.top + spacing.sm,
              backgroundColor: toast.kind === "error" ? colors.danger : colors.text
            }
          ]}
        >
          <Text variant="label" style={{ color: toast.kind === "error" ? "#FFFFFF" : colors.background }}>
            {toast.message}
          </Text>
        </Animated.View>
      ) : null}
    </ToastContext.Provider>
  );
}

const styles = StyleSheet.create({
  toast: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    alignSelf: "center",
    maxWidth: 520,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    borderRadius: radii.md,
    zIndex: 1000
  }
});
