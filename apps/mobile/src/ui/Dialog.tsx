import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { colors, radii, spacing } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

type DialogProps = {
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
  testID?: string;
};

/** Cross-platform confirmation dialog (Alert.alert is not available on web). */
export function Dialog({
  visible,
  title,
  message,
  confirmLabel,
  cancelLabel = "Annuler",
  destructive,
  loading,
  onConfirm,
  onCancel,
  children,
  testID
}: DialogProps) {
  return (
    <Modal transparent visible={visible} animationType="fade" onRequestClose={onCancel}>
      <Pressable style={styles.backdrop} onPress={onCancel} accessibilityLabel="Fermer">
        <Pressable style={styles.card} onPress={() => undefined} testID={testID}>
          <Text variant="heading">{title}</Text>
          {message ? (
            <Text variant="small" color="textMuted">
              {message}
            </Text>
          ) : null}
          {children}
          <View style={styles.actions}>
            <Button label={cancelLabel} variant="ghost" onPress={onCancel} style={styles.action} />
            <Button
              label={confirmLabel}
              variant={destructive ? "danger" : "primary"}
              onPress={onConfirm}
              loading={loading}
              style={styles.action}
              testID={testID ? `${testID}-confirm` : undefined}
            />
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

type SheetProps = {
  visible: boolean;
  title?: string;
  onClose: () => void;
  children: React.ReactNode;
};

/** Bottom sheet used for contextual actions. */
export function Sheet({ visible, title, onClose, children }: SheetProps) {
  return (
    <Modal transparent visible={visible} animationType="slide" onRequestClose={onClose}>
      <Pressable style={styles.sheetBackdrop} onPress={onClose} accessibilityLabel="Fermer">
        <Pressable style={styles.sheet} onPress={() => undefined}>
          <View style={styles.grabber} />
          {title ? <Text variant="heading">{title}</Text> : null}
          {children}
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
    padding: spacing.lg
  },
  card: {
    width: "100%",
    maxWidth: 420,
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.xl,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.lg,
    gap: spacing.sm
  },
  actions: {
    flexDirection: "row",
    gap: spacing.sm,
    marginTop: spacing.xs
  },
  action: {
    flex: 1
  },
  sheetBackdrop: {
    flex: 1,
    backgroundColor: colors.scrim,
    justifyContent: "flex-end"
  },
  sheet: {
    width: "100%",
    maxWidth: 560,
    alignSelf: "center",
    backgroundColor: colors.backgroundElevated,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.lg,
    paddingBottom: spacing.xxl,
    gap: spacing.sm
  },
  grabber: {
    alignSelf: "center",
    width: 40,
    height: 4,
    borderRadius: 2,
    backgroundColor: colors.borderStrong,
    marginBottom: spacing.xs
  }
});
