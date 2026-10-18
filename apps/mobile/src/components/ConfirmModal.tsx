import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Button, Text, colors, radii, shadows, spacing } from "../shared/ui";

type ConfirmModalProps = {
  visible: boolean;
  title: string;
  message?: string;
  confirmLabel: string;
  cancelLabel?: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children?: React.ReactNode;
};

/** Cross-platform confirmation dialog (Alert.alert is a no-op on web). */
export function ConfirmModal({
  visible,
  title,
  message,
  confirmLabel,
  cancelLabel = "Annuler",
  destructive = false,
  loading = false,
  onConfirm,
  onCancel,
  children
}: ConfirmModalProps) {
  return (
    <Modal animationType="fade" onRequestClose={onCancel} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onCancel} style={styles.backdrop}>
        <Pressable accessibilityViewIsModal style={styles.sheet} testID="confirm-modal">
          <Text variant="heading" weight="bold">
            {title}
          </Text>
          {message ? <Text tone="muted">{message}</Text> : null}
          {children}
          <View style={styles.actions}>
            <Button
              label={cancelLabel}
              onPress={onCancel}
              variant="outline"
              style={styles.action}
              testID="confirm-cancel"
            />
            <Button
              label={confirmLabel}
              loading={loading}
              onPress={onConfirm}
              style={styles.action}
              testID="confirm-accept"
              variant={destructive ? "destructive" : "primary"}
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
    alignItems: "center",
    justifyContent: "center",
    padding: spacing.xl
  },
  sheet: {
    width: "100%",
    maxWidth: 420,
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md,
    ...shadows.floating
  },
  actions: { flexDirection: "row", gap: spacing.sm, marginTop: spacing.sm },
  action: { flex: 1 }
});
