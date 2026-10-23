import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState
} from "react";
import { Animated, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { createStyles } from "../theme/ThemeProvider";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error";
type ToastItem = { id: number; message: string; kind: ToastKind };
type ToastApi = {
  show: (message: string, kind?: ToastKind) => void;
  success: (message: string) => void;
  error: (message: string) => void;
};

const ToastContext = createContext<ToastApi>({
  show: () => {},
  success: () => {},
  error: () => {}
});

const useStyles = createStyles(({ colors, spacing, radius, contentMaxWidth }) => ({
  host: {
    position: "absolute",
    left: 0,
    right: 0,
    alignItems: "center",
    paddingHorizontal: spacing.lg
  },
  toast: {
    maxWidth: contentMaxWidth,
    width: "100%",
    borderRadius: radius.md,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    backgroundColor: colors.primary
  },
  success: { backgroundColor: colors.success },
  error: { backgroundColor: colors.danger }
}));

const DURATION_MS = 3600;

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  const [item, setItem] = useState<ToastItem | null>(null);
  const progress = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    counter.current += 1;
    setItem({ id: counter.current, message, kind });
  }, []);

  useEffect(() => {
    if (!item) return;
    progress.setValue(0);
    Animated.timing(progress, { toValue: 1, duration: 220, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(progress, { toValue: 0, duration: 220, useNativeDriver: true }).start(
        ({ finished }) => {
          if (finished) setItem((current) => (current?.id === item.id ? null : current));
        }
      );
    }, DURATION_MS);
    return () => clearTimeout(timer);
  }, [item, progress]);

  const api = useMemo<ToastApi>(
    () => ({
      show,
      success: (message) => show(message, "success"),
      error: (message) => show(message, "error")
    }),
    [show]
  );

  return (
    <ToastContext.Provider value={api}>
      {children}
      {item ? (
        <Animated.View
          pointerEvents="none"
          style={[
            styles.host,
            {
              bottom: insets.bottom + 72,
              opacity: progress,
              transform: [
                { translateY: progress.interpolate({ inputRange: [0, 1], outputRange: [16, 0] }) }
              ]
            }
          ]}
        >
          <View
            accessibilityLiveRegion="polite"
            accessibilityRole="alert"
            style={[
              styles.toast,
              item.kind === "success" ? styles.success : null,
              item.kind === "error" ? styles.error : null
            ]}
          >
            <Text tone="onPrimary">{item.message}</Text>
          </View>
        </Animated.View>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  return useContext(ToastContext);
}
