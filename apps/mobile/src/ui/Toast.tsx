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
import { SafeAreaView } from "react-native-safe-area-context";
import { radii, useTheme } from "../theme";
import { Text } from "./Text";

type ToastKind = "info" | "error";
type ToastValue = { show: (message: string, kind?: ToastKind) => void };

const ToastContext = createContext<ToastValue>({ show: () => undefined });

export function ToastProvider({ children }: { children: ReactNode }) {
  const t = useTheme();
  const [toast, setToast] = useState<{ message: string; kind: ToastKind; id: number } | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    counter.current += 1;
    setToast({ message, kind, id: counter.current });
  }, []);

  useEffect(() => {
    if (!toast) return;
    Animated.timing(opacity, { toValue: 1, duration: 150, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 250, useNativeDriver: true }).start(() =>
        setToast((cur) => (cur?.id === toast.id ? null : cur))
      );
    }, 3200);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  const value = useMemo(() => ({ show }), [show]);
  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast ? (
        <SafeAreaView pointerEvents="none" edges={["bottom"]} style={styles.host}>
          <Animated.View
            accessibilityLiveRegion="polite"
            accessibilityRole="alert"
            style={[
              styles.toast,
              { opacity, backgroundColor: toast.kind === "error" ? t.danger : t.text }
            ]}
          >
            <Text variant="label" style={{ color: t.bg }}>
              {toast.message}
            </Text>
          </Animated.View>
        </SafeAreaView>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastValue {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: 0, right: 0, bottom: 72, alignItems: "center" },
  toast: { borderRadius: radii.pill, paddingHorizontal: 18, paddingVertical: 12, maxWidth: "90%" }
});
