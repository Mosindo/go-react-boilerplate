import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radius, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; message: string; kind: ToastKind } | null;

type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel: string;
  cancelLabel?: string;
  destructive?: boolean;
};

type FeedbackApi = {
  toast: (message: string, kind?: ToastKind) => void;
  confirm: (options: ConfirmOptions) => Promise<boolean>;
};

const FeedbackContext = createContext<FeedbackApi | null>(null);

export function FeedbackProvider({ children }: { children: React.ReactNode }) {
  const theme = useTheme();
  const [toastState, setToast] = useState<ToastState>(null);
  const [dialog, setDialog] = useState<(ConfirmOptions & { resolve: (v: boolean) => void }) | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const counter = useRef(0);

  const toast = useCallback((message: string, kind: ToastKind = "info") => {
    counter.current += 1;
    setToast({ id: counter.current, message, kind });
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setToast(null), 3500);
  }, []);

  const confirm = useCallback(
    (options: ConfirmOptions) => new Promise<boolean>((resolve) => setDialog({ ...options, resolve })),
    []
  );

  const close = (value: boolean) => {
    dialog?.resolve(value);
    setDialog(null);
  };

  const api = useMemo(() => ({ toast, confirm }), [toast, confirm]);
  const toastColor =
    toastState?.kind === "error" ? theme.danger : toastState?.kind === "success" ? theme.success : theme.text;

  return (
    <FeedbackContext.Provider value={api}>
      {children}
      {toastState ? (
        <View pointerEvents="none" style={styles.toastWrap}>
          <View
            accessibilityRole="alert"
            accessibilityLiveRegion="polite"
            style={[styles.toast, { backgroundColor: theme.surface, borderColor: toastColor }]}
          >
            <Text variant="bodyStrong" color={toastColor} center>
              {toastState.message}
            </Text>
          </View>
        </View>
      ) : null}
      <Modal visible={!!dialog} transparent animationType="fade" onRequestClose={() => close(false)}>
        <Pressable style={[styles.backdrop, { backgroundColor: theme.overlay }]} onPress={() => close(false)}>
          <Pressable
            style={[styles.card, { backgroundColor: theme.surface }]}
            onPress={() => undefined}
            accessibilityViewIsModal
          >
            <Text variant="heading">{dialog?.title}</Text>
            {dialog?.message ? <Text muted>{dialog.message}</Text> : null}
            <View style={styles.actions}>
              <Button
                label={dialog?.cancelLabel ?? "Annuler"}
                variant="ghost"
                small
                onPress={() => close(false)}
              />
              <Button
                label={dialog?.confirmLabel ?? "OK"}
                variant={dialog?.destructive ? "danger" : "primary"}
                small
                onPress={() => close(true)}
                testID="confirm-accept"
              />
            </View>
          </Pressable>
        </Pressable>
      </Modal>
    </FeedbackContext.Provider>
  );
}

export function useFeedback(): FeedbackApi {
  const ctx = useContext(FeedbackContext);
  if (!ctx) throw new Error("useFeedback must be used inside FeedbackProvider");
  return ctx;
}

const styles = StyleSheet.create({
  toastWrap: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: 96, alignItems: "center" },
  toast: {
    borderWidth: 1.5,
    borderRadius: radius.md,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    maxWidth: 480,
    shadowColor: "#000",
    shadowOpacity: 0.15,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 4 },
    elevation: 6
  },
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: { width: "100%", maxWidth: 400, borderRadius: radius.lg, padding: spacing.xl, gap: spacing.md },
  actions: { flexDirection: "row", justifyContent: "flex-end", gap: spacing.sm, marginTop: spacing.sm }
});
