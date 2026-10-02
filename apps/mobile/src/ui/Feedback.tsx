import { Ionicons } from "@expo/vector-icons";
import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";

import { radii, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

type ToastKind = "info" | "success" | "error";
interface ToastState {
  id: number;
  kind: ToastKind;
  message: string;
}

export interface SheetOption {
  label: string;
  onPress: () => void;
  destructive?: boolean;
}

interface SheetState {
  title?: string;
  message?: string;
  options: SheetOption[];
}

interface FeedbackApi {
  toast: (message: string, kind?: ToastKind) => void;
  /** Bottom sheet with actions. `Alert.alert` is not usable on web, so everything goes through here. */
  sheet: (title: string | undefined, options: SheetOption[], message?: string) => void;
  confirm: (opts: {
    title: string;
    message: string;
    confirmLabel: string;
    destructive?: boolean;
  }) => Promise<boolean>;
}

const FeedbackContext = createContext<FeedbackApi | null>(null);

export function useFeedback(): FeedbackApi {
  const ctx = useContext(FeedbackContext);
  if (!ctx) throw new Error("useFeedback must be used inside FeedbackProvider");
  return ctx;
}

export function FeedbackProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const [toast, setToast] = useState<ToastState | null>(null);
  const [sheet, setSheet] = useState<SheetState | null>(null);
  const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const nextId = useRef(1);

  const showToast = useCallback((message: string, kind: ToastKind = "info") => {
    const id = nextId.current++;
    setToast({ id, kind, message });
    if (toastTimer.current) clearTimeout(toastTimer.current);
    toastTimer.current = setTimeout(() => setToast((t) => (t?.id === id ? null : t)), 3500);
  }, []);

  const showSheet = useCallback(
    (title: string | undefined, options: SheetOption[], message?: string) => {
      setSheet({ title, message, options });
    },
    [],
  );

  const confirm = useCallback<FeedbackApi["confirm"]>(
    (opts) =>
      new Promise<boolean>((resolve) => {
        setSheet({
          title: opts.title,
          message: opts.message,
          options: [
            {
              label: opts.confirmLabel,
              destructive: opts.destructive,
              onPress: () => resolve(true),
            },
            { label: "Annuler", onPress: () => resolve(false) },
          ],
        });
      }),
    [],
  );

  const api = useMemo(
    () => ({ toast: showToast, sheet: showSheet, confirm }),
    [showToast, showSheet, confirm],
  );

  const close = () => setSheet(null);

  return (
    <FeedbackContext.Provider value={api}>
      {children}
      {toast ? (
        <View pointerEvents="none" style={styles.toastWrap}>
          <View
            accessibilityLiveRegion="polite"
            style={[
              styles.toast,
              {
                backgroundColor:
                  toast.kind === "error"
                    ? colors.danger
                    : toast.kind === "success"
                      ? colors.success
                      : colors.text,
              },
            ]}
          >
            <Ionicons
              name={
                toast.kind === "error"
                  ? "alert-circle"
                  : toast.kind === "success"
                    ? "checkmark-circle"
                    : "information-circle"
              }
              size={20}
              color={colors.background}
            />
            <Text variant="label" style={{ color: colors.background, flexShrink: 1 }}>
              {toast.message}
            </Text>
          </View>
        </View>
      ) : null}
      <Modal visible={sheet !== null} transparent animationType="fade" onRequestClose={close}>
        <Pressable
          style={[styles.backdrop, { backgroundColor: colors.overlay }]}
          onPress={close}
          accessibilityLabel="Fermer"
        >
          <Pressable
            style={[styles.sheet, { backgroundColor: colors.surface }]}
            onPress={() => undefined}
            accessibilityViewIsModal
          >
            {sheet?.title ? (
              <Text variant="heading" center>
                {sheet.title}
              </Text>
            ) : null}
            {sheet?.message ? (
              <Text tone="muted" center>
                {sheet.message}
              </Text>
            ) : null}
            {sheet?.options.map((option) => (
              <Button
                key={option.label}
                label={option.label}
                variant={
                  option.destructive ? "danger" : option.label === "Annuler" ? "ghost" : "secondary"
                }
                onPress={() => {
                  close();
                  option.onPress();
                }}
              />
            ))}
            {sheet && !sheet.options.some((o) => o.label === "Annuler") ? (
              <Button label="Fermer" variant="ghost" onPress={close} />
            ) : null}
          </Pressable>
        </Pressable>
      </Modal>
    </FeedbackContext.Provider>
  );
}

const styles = StyleSheet.create({
  toastWrap: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    bottom: 96,
    alignItems: "center",
  },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderRadius: radii.pill,
    maxWidth: 480,
  },
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    borderTopLeftRadius: radii.lg,
    borderTopRightRadius: radii.lg,
    padding: spacing.xl,
    gap: spacing.md,
    width: "100%",
    maxWidth: 560,
    alignSelf: "center",
  },
});
