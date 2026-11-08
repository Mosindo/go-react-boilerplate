import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radii, spacing, useTheme } from "../theme/theme";
import { Button } from "./Button";
import { Text } from "./Text";

type SheetProps = { visible: boolean; onClose: () => void; title?: string; children: React.ReactNode };

/** Bottom sheet built on Modal so it behaves identically on iOS, Android and web. */
export function Sheet({ visible, onClose, title, children }: SheetProps) {
  const { colors } = useTheme();
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable
        style={[styles.backdrop, { backgroundColor: colors.overlay }]}
        onPress={onClose}
        accessibilityLabel="Fermer"
      />
      <View style={[styles.sheet, { backgroundColor: colors.surface }]}>
        {title ? (
          <Text variant="heading" style={styles.title}>
            {title}
          </Text>
        ) : null}
        {children}
      </View>
    </Modal>
  );
}

export type ActionItem = { label: string; onPress: () => void; destructive?: boolean };

export function ActionSheet({
  visible,
  onClose,
  title,
  actions
}: {
  visible: boolean;
  onClose: () => void;
  title?: string;
  actions: ActionItem[];
}) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title}>
      {actions.map((a) => (
        <Button
          key={a.label}
          title={a.label}
          variant={a.destructive ? "danger" : "secondary"}
          style={styles.action}
          onPress={() => {
            onClose();
            a.onPress();
          }}
        />
      ))}
      <Button title="Annuler" variant="ghost" onPress={onClose} />
    </Sheet>
  );
}

type ConfirmProps = {
  visible: boolean;
  title: string;
  message: string;
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
}: ConfirmProps) {
  return (
    <Sheet visible={visible} onClose={onCancel} title={title}>
      <Text tone="textMuted" style={styles.message}>
        {message}
      </Text>
      {children}
      <Button
        title={confirmLabel}
        variant={destructive ? "danger" : "primary"}
        loading={loading}
        onPress={onConfirm}
        style={styles.action}
      />
      <Button title="Annuler" variant="ghost" onPress={onCancel} />
    </Sheet>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1 },
  sheet: {
    borderTopLeftRadius: radii.lg,
    borderTopRightRadius: radii.lg,
    padding: spacing.lg,
    paddingBottom: spacing.xl
  },
  title: { marginBottom: spacing.sm },
  message: { marginBottom: spacing.md },
  action: { marginBottom: spacing.xs }
});
