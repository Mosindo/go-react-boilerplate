import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type DialogProps = {
  visible: boolean;
  onClose: () => void;
  title: string;
  message?: string;
  children?: React.ReactNode;
  testID?: string;
};

/** Centred modal dialog. Used instead of Alert so behaviour is identical on native and web. */
export function Dialog({ visible, onClose, title, message, children, testID }: DialogProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable
          accessibilityViewIsModal
          onPress={() => undefined}
          style={[styles.card, { backgroundColor: colors.surface }]}
          testID={testID}
        >
          <Text accessibilityRole="header" variant="heading">
            {title}
          </Text>
          {message ? <Text tone="muted">{message}</Text> : null}
          {children}
        </Pressable>
      </Pressable>
    </Modal>
  );
}

type ConfirmDialogProps = {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({ visible, title, message, confirmLabel, destructive, loading, onConfirm, onCancel }: ConfirmDialogProps) {
  return (
    <Dialog message={message} onClose={onCancel} title={title} visible={visible}>
      <View style={styles.actions}>
        <Button
          label={confirmLabel}
          loading={loading}
          onPress={onConfirm}
          testID="confirm-dialog-confirm"
          variant={destructive ? "danger" : "primary"}
        />
        <Button label="Annuler" onPress={onCancel} variant="ghost" />
      </View>
    </Dialog>
  );
}

export type SheetAction = { label: string; icon: IconName; onPress: () => void; destructive?: boolean; testID?: string };

export function ActionSheet({
  visible,
  onClose,
  actions,
  title
}: {
  visible: boolean;
  onClose: () => void;
  actions: SheetAction[];
  title?: string;
}) {
  const { colors } = useTheme();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.sheetBackdrop, { backgroundColor: colors.overlay }]}>
        <Pressable onPress={() => undefined} style={[styles.sheet, { backgroundColor: colors.surface }]}>
          {title ? (
            <Text style={styles.sheetTitle} tone="muted" variant="label">
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              accessibilityRole="button"
              key={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [styles.sheetRow, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
              testID={action.testID}
            >
              <Icon color={action.destructive ? colors.danger : colors.text} name={action.icon} />
              <Text tone={action.destructive ? "danger" : "default"} variant="bodyStrong">
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Button label="Fermer" onPress={onClose} variant="secondary" />
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  card: { width: "100%", maxWidth: 420, borderRadius: radii.xl, padding: spacing.xl, gap: spacing.md },
  actions: { gap: spacing.xs, marginTop: spacing.xs },
  sheetBackdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    width: "100%",
    maxWidth: 560,
    alignSelf: "center",
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.lg,
    paddingBottom: spacing.xxl,
    gap: spacing.xxs
  },
  sheetTitle: { paddingHorizontal: spacing.sm, paddingBottom: spacing.xs },
  sheetRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.sm,
    borderRadius: radii.md
  }
});
