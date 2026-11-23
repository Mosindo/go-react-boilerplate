import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Button, Text, colors, radii, spacing } from "../ui";

type ConfirmDialogProps = {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel?: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

/** Cross-platform confirmation (React Native's Alert is a no-op on web). */
export function ConfirmDialog({
  cancelLabel = "Annuler",
  confirmLabel,
  destructive,
  loading,
  message,
  onCancel,
  onConfirm,
  title,
  visible
}: ConfirmDialogProps) {
  return (
    <Modal animationType="fade" onRequestClose={onCancel} transparent visible={visible}>
      <Pressable accessibilityLabel={cancelLabel} onPress={onCancel} style={styles.backdrop}>
        <Pressable accessibilityRole="none" style={styles.sheet} testID="confirm-dialog">
          <Text variant="heading">{title}</Text>
          <Text tone="muted">{message}</Text>
          <View style={styles.actions}>
            <Button label={cancelLabel} onPress={onCancel} style={styles.action} variant="secondary" />
            <Button
              label={confirmLabel}
              loading={loading}
              onPress={onConfirm}
              style={styles.action}
              testID="confirm-dialog-confirm"
              variant={destructive ? "danger" : "primary"}
            />
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay,
    justifyContent: "center",
    padding: spacing.xl
  },
  sheet: {
    alignSelf: "center",
    width: "100%",
    maxWidth: 420,
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.xl,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.xl,
    gap: spacing.md
  },
  actions: {
    flexDirection: "row",
    gap: spacing.sm,
    marginTop: spacing.sm
  },
  action: {
    flex: 1
  }
});
