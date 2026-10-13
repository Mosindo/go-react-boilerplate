import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { Button } from "./Button";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type ConfirmDialogProps = {
  visible: boolean;
  title: string;
  message?: string;
  confirmLabel: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children?: React.ReactNode;
};

export function ConfirmDialog({
  visible,
  title,
  message,
  confirmLabel,
  destructive,
  loading,
  onConfirm,
  onCancel,
  children
}: ConfirmDialogProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onCancel} transparent visible={visible}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable style={StyleSheet.absoluteFill} onPress={onCancel} accessibilityLabel="Fermer" />
        <View accessibilityViewIsModal style={[styles.dialog, { backgroundColor: colors.surface }]}>
          <Text accessibilityRole="header" variant="heading" weight="bold">
            {title}
          </Text>
          {message ? <Text tone="muted">{message}</Text> : null}
          {children}
          <View style={styles.buttons}>
            <Button label="Annuler" onPress={onCancel} variant="outline" />
            <Button label={confirmLabel} loading={loading} onPress={onConfirm} variant={destructive ? "destructive" : "primary"} />
          </View>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  dialog: { width: "100%", maxWidth: 420, borderRadius: radii.xl, padding: spacing.xl, gap: spacing.md },
  buttons: { flexDirection: "row", justifyContent: "flex-end", gap: spacing.sm, flexWrap: "wrap", paddingTop: spacing.sm }
});
