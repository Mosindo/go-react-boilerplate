import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, Modal, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../theme";
import { Button, Icon, Text } from "../ui";

type ToastTone = "info" | "success" | "error";
type Toast = { id: number; message: string; tone: ToastTone };

export type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel: string;
  cancelLabel?: string;
  destructive?: boolean;
};

type FeedbackContextValue = {
  toast: (message: string, tone?: ToastTone) => void;
  confirm: (options: ConfirmOptions) => Promise<boolean>;
};

const FeedbackContext = createContext<FeedbackContextValue | null>(null);

// Lets non-React code (the API client) surface errors.
let externalToast: ((message: string, tone?: ToastTone) => void) | null = null;
export function showToast(message: string, tone: ToastTone = "error") {
  externalToast?.(message, tone);
}

export function FeedbackProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const [toastState, setToastState] = useState<Toast | null>(null);
  const [dialog, setDialog] = useState<(ConfirmOptions & { resolve: (v: boolean) => void }) | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const counter = useRef(0);

  const toast = useCallback((message: string, tone: ToastTone = "info") => {
    counter.current += 1;
    setToastState({ id: counter.current, message, tone });
  }, []);

  useEffect(() => {
    externalToast = toast;
    return () => {
      externalToast = null;
    };
  }, [toast]);

  useEffect(() => {
    if (!toastState) {
      return;
    }
    Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true }).start();
    const timer = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 180, useNativeDriver: true }).start(() => setToastState(null));
    }, 3200);
    return () => clearTimeout(timer);
  }, [opacity, toastState]);

  const confirm = useCallback((options: ConfirmOptions) => new Promise<boolean>((resolve) => setDialog({ ...options, resolve })), []);

  const closeDialog = (value: boolean) => {
    dialog?.resolve(value);
    setDialog(null);
  };

  const value = useMemo(() => ({ toast, confirm }), [toast, confirm]);
  const toneIcon =
    toastState?.tone === "success" ? "checkmark-circle" : toastState?.tone === "error" ? "alert-circle" : "information-circle";

  return (
    <FeedbackContext.Provider value={value}>
      {children}
      {toastState ? (
        <Animated.View
          accessibilityLiveRegion="polite"
          accessibilityRole="alert"
          pointerEvents="none"
          style={[styles.toast, { opacity, bottom: insets.bottom + 96, backgroundColor: colors.text }]}
          testID="toast"
        >
          <Icon
            color={toastState.tone === "error" ? "danger" : toastState.tone === "success" ? "success" : "background"}
            name={toneIcon}
            size={20}
          />
          <Text color="background" style={styles.toastText} variant="label">
            {toastState.message}
          </Text>
        </Animated.View>
      ) : null}
      <Modal animationType="fade" onRequestClose={() => closeDialog(false)} transparent visible={!!dialog}>
        <Pressable
          accessibilityLabel="Fermer"
          onPress={() => closeDialog(false)}
          style={[styles.backdrop, { backgroundColor: colors.overlay }]}
        >
          <Pressable style={[styles.dialog, { backgroundColor: colors.surfaceRaised }]} testID="confirm-dialog">
            <Text variant="heading">{dialog?.title}</Text>
            {dialog?.message ? <Text color="textMuted">{dialog.message}</Text> : null}
            <View style={styles.actions}>
              <Button
                label={dialog?.cancelLabel ?? "Annuler"}
                onPress={() => closeDialog(false)}
                testID="confirm-cancel"
                variant="secondary"
              />
              <Button
                label={dialog?.confirmLabel ?? "OK"}
                onPress={() => closeDialog(true)}
                testID="confirm-accept"
                variant={dialog?.destructive ? "danger" : "primary"}
              />
            </View>
          </Pressable>
        </Pressable>
      </Modal>
    </FeedbackContext.Provider>
  );
}

export function useFeedback(): FeedbackContextValue {
  const ctx = useContext(FeedbackContext);
  if (!ctx) {
    throw new Error("useFeedback must be used within FeedbackProvider");
  }
  return ctx;
}

const styles = StyleSheet.create({
  toast: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    alignSelf: "center",
    maxWidth: 520,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  toastText: { flex: 1 },
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  dialog: { width: "100%", maxWidth: 400, borderRadius: radii.lg, padding: spacing.xl, gap: spacing.sm },
  actions: { gap: spacing.xs, marginTop: spacing.sm }
});
