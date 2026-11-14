import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, Pressable, StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Toast = { id: number; title: string; body?: string; tone: "info" | "success" | "error"; onPress?: () => void };
type ToastInput = Omit<Toast, "id" | "tone"> & { tone?: Toast["tone"] };

const ToastContext = createContext<(t: ToastInput) => void>(() => undefined);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toast, setToast] = useState<Toast | null>(null);
  const seq = useRef(0);
  const show = useCallback((t: ToastInput) => {
    seq.current += 1;
    setToast({ tone: "info", ...t, id: seq.current });
  }, []);
  return (
    <ToastContext.Provider value={show}>
      {children}
      {toast ? <ToastView key={toast.id} onDone={() => setToast(null)} toast={toast} /> : null}
    </ToastContext.Provider>
  );
}

export function useToast() {
  return useContext(ToastContext);
}

function ToastView({ toast, onDone }: { toast: Toast; onDone: () => void }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const anim = useRef(new Animated.Value(0)).current;
  const icon: IconName = { info: "sparkles-outline", success: "checkmark-circle-outline", error: "alert-circle-outline" }[toast.tone] as IconName;
  const tint = { info: colors.primary, success: colors.success, error: colors.danger }[toast.tone];

  useEffect(() => {
    Animated.spring(anim, { toValue: 1, useNativeDriver: true, friction: 8 }).start();
    const timer = setTimeout(() => {
      Animated.timing(anim, { toValue: 0, duration: 180, useNativeDriver: true }).start(onDone);
    }, 3800);
    return () => clearTimeout(timer);
  }, [anim, onDone]);

  const value = useMemo(
    () => ({
      opacity: anim,
      transform: [{ translateY: anim.interpolate({ inputRange: [0, 1], outputRange: [-30, 0] }) }]
    }),
    [anim]
  );

  return (
    <Animated.View pointerEvents="box-none" style={[styles.wrap, { top: insets.top + spacing.xs }, value]}>
      <Pressable
        accessibilityLiveRegion="polite"
        accessibilityRole="alert"
        onPress={() => {
          toast.onPress?.();
          onDone();
        }}
        style={[styles.toast, { backgroundColor: colors.surfaceRaised, borderColor: colors.border, shadowColor: colors.shadow }]}
      >
        <Icon color={tint} name={icon} size={22} />
        <Animated.View style={styles.text}>
          <Text variant="label">{toast.title}</Text>
          {toast.body ? (
            <Text numberOfLines={2} tone="muted" variant="caption">
              {toast.body}
            </Text>
          ) : null}
        </Animated.View>
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { position: "absolute", left: spacing.md, right: spacing.md, alignItems: "center", zIndex: 1000 },
  toast: {
    width: "100%",
    maxWidth: 520,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    padding: spacing.md,
    borderRadius: radii.lg,
    borderWidth: StyleSheet.hairlineWidth,
    shadowOpacity: 0.16,
    shadowRadius: 20,
    shadowOffset: { width: 0, height: 8 },
    elevation: 8
  },
  text: { flex: 1, gap: 2 }
});
