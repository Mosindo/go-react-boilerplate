import { Ionicons } from "@expo/vector-icons";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Text } from "../shared/ui";
import { radii, spacing, useTheme } from "../theme";

export type MenuAction = { key: string; label: string; icon: keyof typeof Ionicons.glyphMap; destructive?: boolean; onPress: () => void };

/** Bottom sheet listing actions (more reliable and more accessible than long-press menus). */
export function ActionMenu({ visible, actions, onClose }: { visible: boolean; actions: MenuAction[]; onClose: () => void }) {
  const { colors } = useTheme();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose} accessibilityViewIsModal>
      <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={onClose} accessibilityLabel="Fermer le menu" accessibilityRole="button">
        <View style={[styles.sheet, { backgroundColor: colors.surface }]} testID="action-menu">
          {actions.map((action) => (
            <Pressable
              key={action.key}
              testID={`action-${action.key}`}
              accessibilityRole="button"
              accessibilityLabel={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [styles.item, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
            >
              <Ionicons name={action.icon} size={22} color={action.destructive ? colors.danger : colors.text} />
              <Text variant="bodyStrong" tone={action.destructive ? "danger" : "default"}>
                {action.label}
              </Text>
            </Pressable>
          ))}
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end", alignItems: "center" },
  sheet: { width: "100%", maxWidth: 560, borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl, padding: spacing.sm, paddingBottom: spacing.xl },
  item: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, minHeight: 56, borderRadius: radii.md }
});
