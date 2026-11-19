import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, Modal, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { radii, shadow, spacing } from "../theme/tokens";
import { Button } from "./Button";
import { Icon } from "./Icon";
import { Text } from "./Text";

type ToastTone = "info" | "success" | "error";
type Toast = { id: number; message: string; tone: ToastTone };

type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
};

type FeedbackValue = {
  toast: (message: string, tone?: ToastTone) => void;
  confirm: (options: ConfirmOptions) => Promise<boolean>;
};

const FeedbackContext = createContext<FeedbackValue | null>(null);

/** Toasts and confirmation dialogs that work identically on iOS, Android and web. */
export function FeedbackProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<Toast | null>(null);
  const [dialog, setDialog] = useState<(ConfirmOptions & { resolve: (v: boolean) => void }) | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);

  const showToast = useCallback((message: string, tone: ToastTone = "info") => {
    counter.current += 1;
    setToast({ id: counter.current, message, tone });
  }, []);

  useEffect(() => {
    if (!toast) return;
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 220, useNativeDriver: true }).start(() => setToast(null));
    }, 3200);
    return () => clearTimeout(timer);
  }, [toast, opacity]);

  const confirm = useCallback((options: ConfirmOptions) => new Promise<boolean>((resolve) => setDialog({ ...options, resolve })), []);

  const close = (value: boolean) => {
    dialog?.resolve(value);
    setDialog(null);
  };

  const value = useMemo(() => ({ toast: showToast, confirm }), [showToast, confirm]);
  const toneIcon = toast?.tone === "success" ? "checkmark-circle" : toast?.tone === "error" ? "alert-circle" : "information-circle";
  const toneColor = toast?.tone === "success" ? colors.success : toast?.tone === "error" ? colors.danger : colors.primary;

  return (
    <FeedbackContext.Provider value={value}>
      {children}
      {toast ? (
        <Animated.View
          pointerEvents="none"
          style={[styles.toastWrap, { opacity, top: insets.top + spacing.sm }]}
          accessibilityLiveRegion="polite"
        >
          <View style={[styles.toast, shadow, { backgroundColor: colors.surfaceRaised, borderColor: colors.border }]}>
            <Icon name={toneIcon} size={20} color={toneColor} />
            <Text variant="label" style={styles.toastText} testID="toast">
              {toast.message}
            </Text>
          </View>
        </Animated.View>
      ) : null}
      <Modal transparent visible={!!dialog} animationType="fade" onRequestClose={() => close(false)}>
        <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={() => close(false)} accessibilityLabel="Fermer">
          <Pressable style={[styles.dialog, { backgroundColor: colors.surfaceRaised }]} onPress={() => undefined}>
            <Text variant="heading">{dialog?.title}</Text>
            {dialog?.message ? <Text tone="muted">{dialog.message}</Text> : null}
            <View style={styles.dialogActions}>
              <Button label={dialog?.cancelLabel ?? "Annuler"} variant="secondary" onPress={() => close(false)} style={styles.flex} />
              <Button
                label={dialog?.confirmLabel ?? "Confirmer"}
                variant={dialog?.destructive ? "danger" : "primary"}
                onPress={() => close(true)}
                style={styles.flex}
                testID="confirm-dialog-ok"
              />
            </View>
          </Pressable>
        </Pressable>
      </Modal>
    </FeedbackContext.Provider>
  );
}

export function useFeedback(): FeedbackValue {
  const ctx = useContext(FeedbackContext);
  if (!ctx) throw new Error("useFeedback must be used within FeedbackProvider");
  return ctx;
}

const styles = StyleSheet.create({
  toastWrap: { position: "absolute", left: spacing.md, right: spacing.md, alignItems: "center", zIndex: 1000 },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    maxWidth: 520,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    borderRadius: radii.pill,
    borderWidth: 1
  },
  toastText: { flexShrink: 1 },
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  dialog: { width: "100%", maxWidth: 400, borderRadius: radii.lg, padding: spacing.lg, gap: spacing.sm },
  dialogActions: { flexDirection: "row", gap: spacing.sm, marginTop: spacing.sm },
  flex: { flex: 1 }
});
