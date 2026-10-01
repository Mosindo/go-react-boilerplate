import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radii, spacing, useTheme } from "../theme/theme";
import { Button, Text } from "./ui";

type ToastKind = "info" | "success" | "error";
type ToastState = { id: number; message: string; kind: ToastKind } | null;

type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
};

type FeedbackValue = {
  toast: (message: string, kind?: ToastKind) => void;
  confirm: (options: ConfirmOptions) => Promise<boolean>;
};

const FeedbackContext = createContext<FeedbackValue | null>(null);

export function FeedbackProvider({ children }: { children: React.ReactNode }) {
  const t = useTheme();
  const [toastState, setToast] = useState<ToastState>(null);
  const [dialog, setDialog] = useState<(ConfirmOptions & { resolve: (v: boolean) => void }) | null>(null);
  const seq = useRef(0);

  const toast = useCallback((message: string, kind: ToastKind = "info") => {
    seq.current += 1;
    setToast({ id: seq.current, message, kind });
  }, []);

  useEffect(() => {
    if (!toastState) return;
    const timer = setTimeout(() => setToast(null), 3500);
    return () => clearTimeout(timer);
  }, [toastState]);

  const confirm = useCallback(
    (options: ConfirmOptions) => new Promise<boolean>((resolve) => setDialog({ ...options, resolve })),
    []
  );

  const close = (answer: boolean) => {
    dialog?.resolve(answer);
    setDialog(null);
  };

  const value = useMemo(() => ({ toast, confirm }), [toast, confirm]);
  const bg = toastState?.kind === "error" ? t.danger : toastState?.kind === "success" ? t.success : t.text;

  return (
    <FeedbackContext.Provider value={value}>
      {children}
      {toastState ? (
        <View pointerEvents="none" style={styles.toastWrap}>
          <View
            accessibilityRole="alert"
            accessibilityLiveRegion="polite"
            style={[styles.toast, { backgroundColor: bg }]}
          >
            <Text variant="label" color={t.isDark && toastState.kind === "info" ? t.background : "#FFFFFF"}>
              {toastState.message}
            </Text>
          </View>
        </View>
      ) : null}
      <Modal transparent visible={!!dialog} animationType="fade" onRequestClose={() => close(false)}>
        <Pressable style={[styles.backdrop, { backgroundColor: t.overlay }]} onPress={() => close(false)}>
          <Pressable style={[styles.dialog, { backgroundColor: t.surface }]} onPress={() => undefined}>
            <Text variant="heading" style={{ marginBottom: spacing.sm }}>
              {dialog?.title}
            </Text>
            {dialog?.message ? (
              <Text muted style={{ marginBottom: spacing.lg }}>
                {dialog.message}
              </Text>
            ) : null}
            <Button
              label={dialog?.confirmLabel ?? "Confirmer"}
              variant={dialog?.destructive ? "danger" : "primary"}
              onPress={() => close(true)}
              testID="confirm-yes"
            />
            <Button
              label={dialog?.cancelLabel ?? "Annuler"}
              variant="ghost"
              onPress={() => close(false)}
              style={{ marginTop: spacing.sm }}
              testID="confirm-no"
            />
          </Pressable>
        </Pressable>
      </Modal>
    </FeedbackContext.Provider>
  );
}

export function useFeedback(): FeedbackValue {
  const ctx = useContext(FeedbackContext);
  if (!ctx) throw new Error("useFeedback must be used inside <FeedbackProvider>");
  return ctx;
}

const styles = StyleSheet.create({
  toastWrap: { position: "absolute", left: 0, right: 0, bottom: 96, alignItems: "center", paddingHorizontal: spacing.lg },
  toast: { borderRadius: radii.pill, paddingVertical: spacing.md, paddingHorizontal: spacing.xl, maxWidth: 420 },
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  dialog: { width: "100%", maxWidth: 400, borderRadius: radii.lg, padding: spacing.xl }
});
