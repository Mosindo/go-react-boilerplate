import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Button, Text } from "../shared/ui";
import { radius, spacing, useTheme } from "../theme";

export type SheetAction = { label: string; destructive?: boolean; onPress: () => void };

/** Bottom sheet menu. Used instead of Alert because Android alerts cap at three buttons. */
export function ActionSheet({
  visible,
  title,
  actions,
  onClose
}: {
  visible: boolean;
  title: string;
  actions: SheetAction[];
  onClose: () => void;
}) {
  const theme = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <View style={[styles.backdrop, { backgroundColor: theme.overlay }]}>
        <Pressable
          accessibilityLabel="Close menu"
          accessibilityRole="button"
          onPress={onClose}
          style={styles.dismiss}
        />
        <View style={[styles.sheet, { backgroundColor: theme.background }]}>
          <Text tone="muted" variant="label">
            {title}
          </Text>
          {actions.map((action) => (
            <Button
              key={action.label}
              label={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              variant={action.destructive ? "danger" : "secondary"}
            />
          ))}
          <Button label="Cancel" onPress={onClose} variant="ghost" />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  dismiss: { flex: 1 },
  sheet: { padding: spacing.lg, gap: spacing.sm, borderTopLeftRadius: radius.lg, borderTopRightRadius: radius.lg }
});
