import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "../ui";

export type MenuAction = { label: string; onPress: () => void; destructive?: boolean };

type Props = { visible: boolean; title?: string; actions: MenuAction[]; onClose: () => void };

/** Bottom sheet menu: works the same on iOS, Android and web (Alert is limited to 3 buttons). */
export function ActionMenu({ visible, title, actions, onClose }: Props) {
  const { colors } = useTheme();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={onClose} accessibilityLabel="Fermer le menu">
        <View style={[styles.sheet, { backgroundColor: colors.surface }]} accessibilityViewIsModal>
          {title ? (
            <Text variant="label" tone="muted" style={styles.title}>
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              key={action.label}
              accessibilityRole="button"
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [styles.item, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
            >
              <Text style={{ color: action.destructive ? colors.danger : colors.text, fontWeight: "600" }}>{action.label}</Text>
            </Pressable>
          ))}
          <Pressable
            accessibilityRole="button"
            onPress={onClose}
            style={[styles.item, { borderTopColor: colors.border, borderTopWidth: StyleSheet.hairlineWidth }]}
          >
            <Text tone="muted" style={{ fontWeight: "600" }}>
              Annuler
            </Text>
          </Pressable>
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl, paddingVertical: spacing.sm, paddingBottom: spacing.xl },
  title: { textAlign: "center", paddingVertical: spacing.md },
  item: { paddingVertical: spacing.lg, paddingHorizontal: spacing.xl, alignItems: "center" }
});
