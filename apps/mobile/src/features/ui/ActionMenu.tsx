import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Text, radii, spacing } from "../../shared/ui";
import { usePalette } from "./palette";

export type MenuAction = { key: string; label: string; destructive?: boolean; onPress: () => void };

type Props = { visible: boolean; actions: MenuAction[]; onClose: () => void };

export function ActionMenu({ visible, actions, onClose }: Props) {
  const p = usePalette();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable
        accessibilityLabel="Close menu"
        onPress={onClose}
        style={[styles.backdrop, { backgroundColor: p.scrim }]}
      >
        <View style={[styles.sheet, { backgroundColor: p.surface }]} testID="action-menu">
          {actions.map((action) => (
            <Pressable
              accessibilityRole="button"
              key={action.key}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={styles.item}
              testID={`menu-${action.key}`}
            >
              <Text tone={action.destructive ? "danger" : "default"} weight="semibold">
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" onPress={onClose} style={styles.item}>
            <Text tone="muted">Cancel</Text>
          </Pressable>
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end", padding: spacing.lg },
  sheet: { borderRadius: radii.xl, paddingVertical: spacing.sm },
  item: { paddingHorizontal: spacing.xl, paddingVertical: spacing.lg }
});
