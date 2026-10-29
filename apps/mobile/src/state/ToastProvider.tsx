import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState
} from "react";
import { Animated, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; message: string; kind: ToastKind };

type ToastApi = { show: (message: string, kind?: ToastKind) => void };

const ToastContext = createContext<ToastApi>({ show: () => undefined });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toast, setToast] = useState<ToastState | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);
  const insets = useSafeAreaInsets();
  const theme = useTheme();

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    counter.current += 1;
    setToast({ id: counter.current, message, kind });
  }, []);

  useEffect(() => {
    if (!toast) {
      return;
    }
    opacity.setValue(0);
    const animation = Animated.sequence([
      Animated.timing(opacity, { toValue: 1, duration: 160, useNativeDriver: true }),
      Animated.delay(3200),
      Animated.timing(opacity, { toValue: 0, duration: 240, useNativeDriver: true })
    ]);
    animation.start(({ finished }) => {
      if (finished) {
        setToast(null);
      }
    });
    return () => animation.stop();
  }, [toast, opacity]);

  const api = useMemo(() => ({ show }), [show]);
  const background =
    toast?.kind === "error"
      ? theme.danger
      : toast?.kind === "success"
        ? theme.success
        : theme.accent;

  return (
    <ToastContext.Provider value={api}>
      {children}
      {toast ? (
        <Animated.View
          accessibilityLiveRegion="polite"
          pointerEvents="none"
          style={[styles.wrap, { top: insets.top + spacing.sm, opacity }]}
        >
          <View style={[styles.toast, { backgroundColor: background }]}>
            <Text style={styles.text}>{toast.message}</Text>
          </View>
        </Animated.View>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  return useContext(ToastContext);
}

const styles = StyleSheet.create({
  wrap: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    alignItems: "center",
    zIndex: 100
  },
  toast: {
    borderRadius: radii.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    maxWidth: 480
  },
  text: { color: "#FFFFFF", fontSize: fontSizes.sm, fontWeight: "600", textAlign: "center" }
});
