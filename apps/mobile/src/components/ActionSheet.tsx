import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { MIN_TOUCH, radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

export type SheetAction = { label: string; onPress: () => void; destructive?: boolean; testID?: string };

export function ActionSheet({
  visible,
  title,
  actions,
  onClose
}: {
  visible: boolean;
  title?: string;
  actions: SheetAction[];
  onClose: () => void;
}) {
  const { colors } = useTheme();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={onClose} accessibilityLabel="Close menu">
        <View style={[styles.sheet, { backgroundColor: colors.surface }]}>
          {title ? (
            <Text variant="caption" tone="muted" style={styles.title}>
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              key={action.label}
              accessibilityRole="button"
              testID={action.testID}
              style={styles.row}
              onPress={() => {
                onClose();
                action.onPress();
              }}
            >
              <Text style={{ color: action.destructive ? colors.danger : colors.text, fontWeight: "600" }}>{action.label}</Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" style={styles.row} onPress={onClose}>
            <Text tone="muted">Cancel</Text>
          </Pressable>
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { borderTopLeftRadius: radii.lg, borderTopRightRadius: radii.lg, padding: spacing.lg, paddingBottom: spacing.xl },
  title: { textAlign: "center", marginBottom: spacing.sm },
  row: { minHeight: MIN_TOUCH + 4, justifyContent: "center", alignItems: "center" }
});
