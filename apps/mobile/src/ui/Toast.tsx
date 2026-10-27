import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState
} from "react";
import { Animated, StyleSheet } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { radii, usePalette } from "../theme";
import { Text } from "./Text";

type ToastKind = "info" | "error" | "success";
type ToastApi = { show: (message: string, kind?: ToastKind) => void };

const ToastContext = createContext<ToastApi>({ show: () => undefined });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const p = usePalette();
  const [toast, setToast] = useState<{ message: string; kind: ToastKind; id: number } | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    counter.current += 1;
    setToast({ message, kind, id: counter.current });
  }, []);

  useEffect(() => {
    if (!toast) return;
    Animated.timing(opacity, { toValue: 1, duration: 160, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 220, useNativeDriver: true }).start(() =>
        setToast((current) => (current?.id === toast.id ? null : current))
      );
    }, 3200);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  const api = useMemo(() => ({ show }), [show]);
  const bg = toast?.kind === "error" ? p.danger : toast?.kind === "success" ? p.success : p.text;

  return (
    <ToastContext.Provider value={api}>
      {children}
      {toast ? (
        <SafeAreaView pointerEvents="none" edges={["top"]} style={styles.host}>
          <Animated.View
            accessibilityLiveRegion="polite"
            style={[styles.toast, { backgroundColor: bg, opacity }]}
          >
            <Text
              variant="label"
              style={{ color: toast.kind === "info" ? p.background : "#FFFFFF" }}
            >
              {toast.message}
            </Text>
          </Animated.View>
        </SafeAreaView>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  host: { position: "absolute", top: 0, left: 0, right: 0, alignItems: "center" },
  toast: {
    marginTop: 8,
    marginHorizontal: 16,
    paddingHorizontal: 18,
    paddingVertical: 12,
    borderRadius: radii.pill
  }
});
