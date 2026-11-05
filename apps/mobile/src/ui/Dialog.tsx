import React, { type ReactNode } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radii, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

type Props = {
  visible: boolean;
  title: string;
  message?: string;
  children?: ReactNode;
  confirmLabel?: string;
  confirmVariant?: "primary" | "danger";
  cancelLabel?: string;
  loading?: boolean;
  onConfirm?: () => void;
  onCancel: () => void;
};

/** Cross-platform confirmation dialog (Alert.alert is a no-op on web). */
export function Dialog({
  visible,
  title,
  message,
  children,
  confirmLabel,
  confirmVariant = "primary",
  cancelLabel = "Annuler",
  loading,
  onConfirm,
  onCancel
}: Props) {
  const t = useTheme();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onCancel}>
      <Pressable
        style={[styles.backdrop, { backgroundColor: t.overlay }]}
        onPress={onCancel}
        accessibilityLabel="Fermer"
      >
        <Pressable
          accessibilityViewIsModal
          style={[styles.card, { backgroundColor: t.surface }]}
          onPress={() => undefined}
        >
          <Text variant="heading">{title}</Text>
          {message ? <Text tone="muted">{message}</Text> : null}
          {children}
          <View style={styles.actions}>
            {onConfirm && confirmLabel ? (
              <Button
                label={confirmLabel}
                variant={confirmVariant}
                loading={loading}
                onPress={onConfirm}
              />
            ) : null}
            <Button label={cancelLabel} variant="ghost" onPress={onCancel} />
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "center", padding: 24 },
  card: { borderRadius: radii.lg, padding: 20, gap: 12 },
  actions: { gap: 4, marginTop: 8 }
});
