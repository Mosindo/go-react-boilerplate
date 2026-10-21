import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { MIN_TOUCH, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type SheetAction = { label: string; destructive?: boolean; onPress: () => void; testID?: string };

type Props = { visible: boolean; title?: string; actions: SheetAction[]; onClose: () => void };

/** Bottom sheet of actions; works the same on iOS, Android and the web. */
export function ActionSheet({ visible, title, actions, onClose }: Props) {
  const theme = useTheme();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable style={[styles.backdrop, { backgroundColor: theme.overlay }]} onPress={onClose} accessibilityLabel="Fermer le menu">
        <View style={[styles.sheet, { backgroundColor: theme.surface }]} onStartShouldSetResponder={() => true}>
          {title ? (
            <Text variant="label" tone="muted" center>
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              key={action.label}
              testID={action.testID}
              accessibilityRole="button"
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [styles.action, { borderColor: theme.border, opacity: pressed ? 0.7 : 1 }]}
            >
              <Text style={{ fontWeight: "600" }} tone={action.destructive ? "danger" : "default"} center>
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" onPress={onClose} style={styles.action}>
            <Text tone="muted" center>
              Annuler
            </Text>
          </Pressable>
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 520,
    borderTopLeftRadius: radius.lg,
    borderTopRightRadius: radius.lg,
    padding: spacing.lg,
    gap: spacing.xs
  },
  action: { minHeight: MIN_TOUCH, justifyContent: "center", borderBottomWidth: StyleSheet.hairlineWidth, borderColor: "transparent" }
});
