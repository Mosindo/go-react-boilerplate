import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Text, colors, radii, spacing } from "../shared/ui";

export type SheetAction = {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  testID?: string;
};

type ActionSheetProps = {
  visible: boolean;
  title?: string;
  actions: SheetAction[];
  onClose: () => void;
};

/** Bottom sheet used instead of Alert, which caps buttons on Android. */
export function ActionSheet({ visible, title, actions, onClose }: ActionSheetProps) {
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={styles.backdrop}>
        <View style={styles.sheet}>
          {title ? (
            <Text style={styles.title} tone="muted" variant="label" weight="semibold">
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              accessibilityRole="button"
              key={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [styles.action, pressed ? styles.pressed : null]}
              testID={action.testID}
            >
              <Text tone={action.destructive ? "danger" : "default"} weight="semibold">
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" onPress={onClose} style={[styles.action, styles.cancel]}>
            <Text tone="muted" weight="semibold">
              Annuler
            </Text>
          </Pressable>
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, justifyContent: "flex-end" },
  sheet: {
    backgroundColor: colors.backgroundElevated,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.lg,
    paddingBottom: spacing.xxl,
    gap: spacing.xs
  },
  title: { textAlign: "center", marginBottom: spacing.sm },
  action: { paddingVertical: spacing.md, alignItems: "center", borderRadius: radii.md },
  pressed: { backgroundColor: colors.surfaceMuted },
  cancel: { marginTop: spacing.xs, backgroundColor: colors.surfaceMuted }
});
