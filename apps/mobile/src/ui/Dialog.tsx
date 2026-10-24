import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radius, spacing, useTheme } from "../theme/theme";
import { Button } from "./Button";
import { Text } from "./Text";

export type DialogAction = {
  label: string;
  value: string;
  variant?: "primary" | "secondary" | "ghost" | "danger";
};

type DialogInput = { title: string; message?: string; actions: DialogAction[] };
type DialogValue = {
  /** Resolves with the chosen action value, or null when dismissed. */
  choose: (input: DialogInput) => Promise<string | null>;
  confirm: (input: {
    title: string;
    message?: string;
    confirmLabel: string;
    destructive?: boolean;
  }) => Promise<boolean>;
};

const DialogContext = createContext<DialogValue | null>(null);

/** Cross-platform replacement for Alert.alert (which does nothing on the web build). */
export function DialogProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const [dialog, setDialog] = useState<DialogInput | null>(null);
  const resolver = useRef<((value: string | null) => void) | null>(null);

  const choose = useCallback(
    (input: DialogInput) =>
      new Promise<string | null>((resolve) => {
        resolver.current = resolve;
        setDialog(input);
      }),
    [],
  );

  const close = (value: string | null) => {
    resolver.current?.(value);
    resolver.current = null;
    setDialog(null);
  };

  const value = useMemo<DialogValue>(
    () => ({
      choose,
      confirm: async ({ title, message, confirmLabel, destructive }) =>
        (await choose({
          title,
          message,
          actions: [
            { label: confirmLabel, value: "yes", variant: destructive ? "danger" : "primary" },
            { label: "Annuler", value: "no", variant: "ghost" },
          ],
        })) === "yes",
    }),
    [choose],
  );

  return (
    <DialogContext.Provider value={value}>
      {children}
      <Modal transparent visible={!!dialog} animationType="fade" onRequestClose={() => close(null)}>
        <Pressable
          style={[styles.backdrop, { backgroundColor: colors.overlay }]}
          onPress={() => close(null)}
        >
          <Pressable
            testID="dialog"
            accessibilityViewIsModal
            style={[styles.card, { backgroundColor: colors.surface }]}
            onPress={() => undefined}
          >
            <Text variant="heading">{dialog?.title}</Text>
            {dialog?.message ? <Text muted>{dialog.message}</Text> : null}
            <View style={styles.actions}>
              {dialog?.actions.map((a) => (
                <Button
                  key={a.value}
                  testID={`dialog-${a.value}`}
                  title={a.label}
                  variant={a.variant ?? "secondary"}
                  onPress={() => close(a.value)}
                />
              ))}
            </View>
          </Pressable>
        </Pressable>
      </Modal>
    </DialogContext.Provider>
  );
}

export function useDialog(): DialogValue {
  const ctx = useContext(DialogContext);
  if (!ctx) throw new Error("useDialog must be used inside DialogProvider");
  return ctx;
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: {
    width: "100%",
    maxWidth: 420,
    borderRadius: radius.lg,
    padding: spacing.xl,
    gap: spacing.md,
  },
  actions: { gap: spacing.sm, marginTop: spacing.sm },
});
