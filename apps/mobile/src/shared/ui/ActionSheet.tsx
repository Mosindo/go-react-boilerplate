import React from "react";
import { Modal, Pressable, StyleSheet } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type SheetAction = {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  disabled?: boolean;
};

type ActionSheetProps = {
  visible: boolean;
  title?: string;
  actions: SheetAction[];
  onClose: () => void;
};

/** Bottom sheet of actions. Used instead of Alert, which is unavailable on web. */
export function ActionSheet({ visible, title, actions, onClose }: ActionSheetProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable style={[styles.sheet, { backgroundColor: colors.surface }]}>
          {title ? (
            <Text style={styles.title} tone="muted" variant="label" weight="semibold">
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              accessibilityRole="button"
              disabled={action.disabled}
              key={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [
                styles.action,
                { backgroundColor: pressed ? colors.surfaceMuted : "transparent", opacity: action.disabled ? 0.4 : 1 }
              ]}
            >
              <Text tone={action.destructive ? "danger" : "default"} weight="semibold">
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" onPress={onClose} style={[styles.action, styles.cancel, { borderColor: colors.border }]}>
            <Text tone="muted" weight="semibold">
              Annuler
            </Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 520,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.md,
    paddingBottom: spacing.xl,
    gap: spacing.xxs
  },
  title: { textAlign: "center", paddingVertical: spacing.sm },
  action: { paddingVertical: spacing.md, alignItems: "center", borderRadius: radii.md },
  cancel: { marginTop: spacing.xs, borderTopWidth: StyleSheet.hairlineWidth, borderRadius: 0 }
});
