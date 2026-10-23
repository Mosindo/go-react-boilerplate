import React, { createContext, useCallback, useContext, useMemo, useState } from "react";
import { Modal, Pressable, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { t } from "../i18n/fr";
import { createStyles } from "../theme/ThemeProvider";
import { ScalePressable } from "./Pressable";
import { Text } from "./Text";

export type SheetAction = {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  testID?: string;
};

export type SheetOptions = {
  title?: string;
  message?: string;
  actions: SheetAction[];
  cancelLabel?: string;
};

type DialogApi = { showSheet: (options: SheetOptions) => void };
const DialogContext = createContext<DialogApi>({ showSheet: () => {} });

const useStyles = createStyles(({ colors, spacing, radius, contentMaxWidth }) => ({
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay,
    justifyContent: "flex-end",
    alignItems: "center"
  },
  sheet: {
    width: "100%",
    maxWidth: contentMaxWidth,
    backgroundColor: colors.surface,
    borderTopLeftRadius: radius.lg,
    borderTopRightRadius: radius.lg,
    padding: spacing.lg,
    gap: spacing.sm
  },
  header: { paddingBottom: spacing.sm, gap: spacing.xs },
  action: {
    minHeight: 48,
    borderRadius: radius.md,
    backgroundColor: colors.surfaceAlt,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: spacing.lg
  }
}));

export function DialogProvider({ children }: { children: React.ReactNode }) {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  const [options, setOptions] = useState<SheetOptions | null>(null);
  const close = useCallback(() => setOptions(null), []);
  const showSheet = useCallback((next: SheetOptions) => setOptions(next), []);
  const api = useMemo(() => ({ showSheet }), [showSheet]);

  return (
    <DialogContext.Provider value={api}>
      {children}
      <Modal animationType="fade" onRequestClose={close} transparent visible={!!options}>
        <View style={styles.backdrop}>
          <Pressable
            accessibilityLabel={t.common.close}
            accessibilityRole="button"
            onPress={close}
            style={{ flex: 1, width: "100%" }}
          />
          <View
            accessibilityViewIsModal
            style={[styles.sheet, { paddingBottom: insets.bottom + 16 }]}
          >
            {options?.title || options?.message ? (
              <View style={styles.header}>
                {options.title ? (
                  <Text variant="heading" center>
                    {options.title}
                  </Text>
                ) : null}
                {options.message ? (
                  <Text tone="muted" center>
                    {options.message}
                  </Text>
                ) : null}
              </View>
            ) : null}
            {options?.actions.map((action) => (
              <ScalePressable
                accessibilityRole="button"
                key={action.label}
                onPress={() => {
                  close();
                  action.onPress();
                }}
                testID={action.testID}
              >
                <View style={styles.action}>
                  <Text variant="label" tone={action.destructive ? "danger" : "primary"}>
                    {action.label}
                  </Text>
                </View>
              </ScalePressable>
            ))}
            <ScalePressable accessibilityRole="button" onPress={close} testID="sheet-cancel">
              <View style={styles.action}>
                <Text variant="label" tone="muted">
                  {options?.cancelLabel ?? t.common.cancel}
                </Text>
              </View>
            </ScalePressable>
          </View>
        </View>
      </Modal>
    </DialogContext.Provider>
  );
}

export function useDialog(): DialogApi & {
  confirm: (o: {
    title: string;
    message?: string;
    confirmLabel: string;
    destructive?: boolean;
    onConfirm: () => void;
  }) => void;
} {
  const api = useContext(DialogContext);
  return useMemo(
    () => ({
      ...api,
      confirm: ({ title, message, confirmLabel, destructive, onConfirm }) =>
        api.showSheet({
          title,
          message,
          actions: [
            { label: confirmLabel, destructive, onPress: onConfirm, testID: "sheet-confirm" }
          ]
        })
    }),
    [api]
  );
}
