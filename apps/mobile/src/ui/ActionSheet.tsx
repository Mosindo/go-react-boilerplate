import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, useTheme } from "../theme/theme";
import { Text } from "./Text";

export type SheetAction = { label: string; onPress: () => void; destructive?: boolean };

type Props = { visible: boolean; title?: string; actions: SheetAction[]; onClose: () => void };

export function ActionSheet({ visible, title, actions, onClose }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={onClose} accessibilityLabel="Fermer">
        <View style={[styles.sheet, { backgroundColor: colors.surface, paddingBottom: insets.bottom + 12 }]}>
          {title ? (
            <Text variant="label" muted style={styles.title}>
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
              style={[styles.row, { borderTopColor: colors.border }]}
            >
              <Text variant="body" color={action.destructive ? colors.danger : colors.text}>
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" onPress={onClose} style={[styles.row, { borderTopColor: colors.border }]}>
            <Text variant="label" muted>
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
  sheet: { borderTopLeftRadius: radii.lg, borderTopRightRadius: radii.lg, paddingTop: 8 },
  title: { textAlign: "center", padding: 12 },
  row: { paddingVertical: 16, alignItems: "center", borderTopWidth: StyleSheet.hairlineWidth }
});
