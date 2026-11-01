import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Animated, StyleSheet, Text as RNText, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { radius, spacing, useColors } from "../theme";

type Toast = { id: number; message: string; tone: "info" | "error" | "success" };

const ToastContext = createContext<{ show: (message: string, tone?: Toast["tone"]) => void } | null>(null);

/** Retour utilisateur apres une action (succes/erreur), non bloquant. */
export function ToastProvider({ children }: { children: ReactNode }) {
  const colors = useColors();
  const [toast, setToast] = useState<Toast | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);

  const show = useCallback((message: string, tone: Toast["tone"] = "info") => {
    counter.current += 1;
    setToast({ id: counter.current, message, tone });
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
  }, [toast, opacity]);

  const value = useMemo(() => ({ show }), [show]);
  const background = toast?.tone === "error" ? colors.danger : toast?.tone === "success" ? colors.success : colors.text;

  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast ? (
        <SafeAreaView edges={["top"]} pointerEvents="none" style={styles.wrap}>
          <Animated.View
            accessibilityLiveRegion="polite"
            style={[styles.toast, { backgroundColor: background, opacity }]}
          >
            <RNText style={[styles.text, { color: colors.background }]}>{toast.message}</RNText>
          </Animated.View>
        </SafeAreaView>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return ctx;
}

export function Spacer({ size = spacing.lg }: { size?: number }) {
  return <View style={{ height: size }} />;
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", top: 0, left: 0, right: 0, alignItems: "center" },
  toast: {
    marginTop: spacing.sm,
    marginHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    borderRadius: radius.md,
  },
  text: { fontSize: 14, fontWeight: "600", textAlign: "center" },
});
