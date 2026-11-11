import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import { Animated, Modal, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { Text } from "./Text";

type Toast = { id: number; message: string; tone: "default" | "error" | "success" };

type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel: string;
  destructive?: boolean;
};

type FeedbackApi = {
  toast: (message: string, tone?: Toast["tone"]) => void;
  confirm: (options: ConfirmOptions) => Promise<boolean>;
};

const FeedbackContext = createContext<FeedbackApi>({
  toast: () => undefined,
  confirm: async () => false
});

/**
 * Toasts and confirmation dialogs. React Native's Alert has no buttons on the
 * web, so confirmations are rendered with a Modal everywhere.
 */
export function FeedbackProvider({ children }: { children: React.ReactNode }) {
  const { colors, radii, spacing } = useTheme();
  const insets = useSafeAreaInsets();
  const [toast, setToast] = useState<Toast | null>(null);
  const [dialog, setDialog] = useState<(ConfirmOptions & { resolve: (v: boolean) => void }) | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;

  const showToast = useCallback((message: string, tone: Toast["tone"] = "default") => {
    setToast({ id: Date.now(), message, tone });
  }, []);

  useEffect(() => {
    if (!toast) {
      return;
    }
    Animated.timing(opacity, { toValue: 1, duration: 160, useNativeDriver: true }).start();
    const t = setTimeout(() => {
      Animated.timing(opacity, { toValue: 0, duration: 200, useNativeDriver: true }).start(() => setToast(null));
    }, 3200);
    return () => clearTimeout(t);
  }, [opacity, toast]);

  const confirm = useCallback(
    (options: ConfirmOptions) => new Promise<boolean>((resolve) => setDialog({ ...options, resolve })),
    []
  );

  const close = (value: boolean) => {
    dialog?.resolve(value);
    setDialog(null);
  };

  const toastBg = toast?.tone === "error" ? colors.danger : toast?.tone === "success" ? colors.success : colors.text;

  return (
    <FeedbackContext.Provider value={{ toast: showToast, confirm }}>
      {children}
      {toast ? (
        <Animated.View
          pointerEvents="none"
          style={[styles.toast, { opacity, top: insets.top + 12, backgroundColor: toastBg, borderRadius: radii.md }]}
          accessibilityLiveRegion="polite"
          accessibilityRole="alert"
        >
          <Text style={{ color: colors.background }} variant="label" align="center">
            {toast.message}
          </Text>
        </Animated.View>
      ) : null}
      <Modal transparent visible={!!dialog} animationType="fade" onRequestClose={() => close(false)}>
        <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={() => close(false)}>
          <Pressable
            style={[
              styles.dialog,
              { backgroundColor: colors.surfaceRaised, borderRadius: radii.lg, padding: spacing.lg }
            ]}
            accessibilityViewIsModal
          >
            <Text variant="heading">{dialog?.title}</Text>
            {dialog?.message ? <Text tone="muted">{dialog.message}</Text> : null}
            <View style={styles.actions}>
              <Button label="Annuler" variant="secondary" size="md" style={styles.flex} onPress={() => close(false)} />
              <Button
                testID="confirm-dialog-confirm"
                label={dialog?.confirmLabel ?? "OK"}
                variant={dialog?.destructive ? "danger" : "primary"}
                size="md"
                style={styles.flex}
                onPress={() => close(true)}
              />
            </View>
          </Pressable>
        </Pressable>
      </Modal>
    </FeedbackContext.Provider>
  );
}

export function useFeedback(): FeedbackApi {
  return useContext(FeedbackContext);
}

const styles = StyleSheet.create({
  toast: {
    position: "absolute",
    left: 20,
    right: 20,
    paddingVertical: 14,
    paddingHorizontal: 18,
    alignSelf: "center",
    maxWidth: 520
  },
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24 },
  dialog: { width: "100%", maxWidth: 420, gap: 12 },
  actions: { flexDirection: "row", gap: 10, marginTop: 8 },
  flex: { flex: 1 }
});
