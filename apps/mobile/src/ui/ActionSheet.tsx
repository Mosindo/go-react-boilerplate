import React from "react";
import { Modal, Pressable, StyleSheet, Text, View } from "react-native";
import { radius, spacing, useTheme } from "../theme";

export type SheetAction = { label: string; destructive?: boolean; onPress: () => void };

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
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable accessibilityLabel="Fermer" style={[styles.backdrop, { backgroundColor: theme.overlay }]} onPress={onClose}>
        <View style={[styles.sheet, { backgroundColor: theme.surface }]}>
          <Text style={[styles.title, { color: theme.textMuted }]}>{title}</Text>
          {actions.map((action) => (
            <Pressable
              key={action.label}
              accessibilityRole="button"
              testID={`action-${action.label}`}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={styles.item}
            >
              <Text style={{ color: action.destructive ? theme.danger : theme.text, fontSize: 17, fontWeight: "600" }}>{action.label}</Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" onPress={onClose} style={styles.item}>
            <Text style={{ color: theme.textMuted, fontSize: 17 }}>Annuler</Text>
          </Pressable>
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { borderTopLeftRadius: radius.lg, borderTopRightRadius: radius.lg, padding: spacing.lg, paddingBottom: spacing.xl },
  title: { textAlign: "center", fontSize: 13, fontWeight: "700", marginBottom: spacing.sm },
  item: { minHeight: 48, justifyContent: "center", alignItems: "center" }
});
