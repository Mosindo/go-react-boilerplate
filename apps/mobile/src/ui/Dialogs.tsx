import React, { type ReactNode } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { MIN_TOUCH, radii, spacing, useColors } from "../theme";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type ConfirmDialogProps = {
  visible: boolean;
  title: string;
  message?: string;
  confirmLabel: string;
  cancelLabel?: string;
  destructive?: boolean;
  loading?: boolean;
  confirmDisabled?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children?: ReactNode;
};

/** Cross-platform confirm dialog (Alert.alert is limited on web). */
export function ConfirmDialog({
  visible,
  title,
  message,
  confirmLabel,
  cancelLabel = "Cancel",
  destructive,
  loading,
  confirmDisabled,
  onConfirm,
  onCancel,
  children
}: ConfirmDialogProps) {
  const colors = useColors();
  return (
    <Modal animationType="fade" onRequestClose={onCancel} transparent visible={visible}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <View
          accessibilityViewIsModal
          style={[styles.dialog, { backgroundColor: colors.surface, borderColor: colors.border }]}
        >
          <Text variant="title">{title}</Text>
          {message ? <Text tone="muted">{message}</Text> : null}
          {children}
          <View style={styles.buttons}>
            <Button
              disabled={confirmDisabled}
              label={confirmLabel}
              loading={loading}
              onPress={onConfirm}
              variant={destructive ? "danger" : "primary"}
            />
            <Button disabled={loading} label={cancelLabel} onPress={onCancel} variant="ghost" />
          </View>
        </View>
      </View>
    </Modal>
  );
}

export type SheetAction = {
  label: string;
  icon?: IconName;
  destructive?: boolean;
  onPress: () => void;
};

type ActionSheetProps = {
  visible: boolean;
  title?: string;
  actions: SheetAction[];
  onClose: () => void;
};

export function ActionSheet({ visible, title, actions, onClose }: ActionSheetProps) {
  const colors = useColors();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable
        accessibilityLabel="Close menu"
        accessibilityRole="button"
        onPress={onClose}
        style={[styles.sheetBackdrop, { backgroundColor: colors.overlay }]}
      >
        <Pressable
          accessible={false}
          style={[styles.sheet, { backgroundColor: colors.surface, borderColor: colors.border }]}
        >
          {title ? (
            <Text center tone="muted" variant="small">
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              key={action.label}
              accessibilityLabel={action.label}
              accessibilityRole="button"
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [styles.sheetRow, pressed && { backgroundColor: colors.surfaceMuted }]}
            >
              {action.icon ? (
                <Icon color={action.destructive ? colors.danger : colors.text} name={action.icon} size={22} />
              ) : null}
              <Text tone={action.destructive ? "danger" : "default"} weight="medium">
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Button label="Cancel" onPress={onClose} variant="ghost" />
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  dialog: {
    width: "100%",
    maxWidth: 420,
    borderRadius: radii.lg,
    borderWidth: 1,
    padding: spacing.lg,
    gap: spacing.sm
  },
  buttons: { gap: spacing.xs, marginTop: spacing.xs },
  sheetBackdrop: { flex: 1, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 520,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    borderWidth: 1,
    padding: spacing.md,
    paddingBottom: spacing.lg,
    gap: spacing.xxs
  },
  sheetRow: {
    minHeight: MIN_TOUCH + 4,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.sm,
    borderRadius: radii.md
  }
});
