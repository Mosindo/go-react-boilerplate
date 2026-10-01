import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Button, Text, radii, spacing, useTheme } from "../shared/ui";

export type SheetAction = { label: string; onPress: () => void; destructive?: boolean };

type ActionSheetProps = {
  visible: boolean;
  title?: string;
  actions: SheetAction[];
  onClose: () => void;
};

/** Bottom sheet menu: unlike Alert it is not limited to three buttons on Android. */
export function ActionSheet({ visible, title, actions, onClose }: ActionSheetProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer le menu" onPress={onClose} style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <View style={[styles.sheet, { backgroundColor: colors.surface }]}>
          {title ? (
            <Text align="center" tone="muted" variant="label">
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Button
              key={action.label}
              label={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              variant={action.destructive ? "destructive" : "secondary"}
            />
          ))}
          <Button label="Annuler" onPress={onClose} variant="ghost" />
        </View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl, padding: spacing.lg, gap: spacing.sm, paddingBottom: spacing.xxl }
});
