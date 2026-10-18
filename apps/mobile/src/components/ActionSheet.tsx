import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Button, colors, radii, spacing } from "../shared/ui";

export type SheetAction = {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  testID?: string;
};

type ActionSheetProps = {
  visible: boolean;
  actions: SheetAction[];
  onClose: () => void;
};

export function ActionSheet({ visible, actions, onClose }: ActionSheetProps) {
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer le menu" onPress={onClose} style={styles.backdrop}>
        <Pressable style={styles.sheet} testID="action-sheet">
          <View style={styles.handle} />
          {actions.map((action) => (
            <Button
              key={action.label}
              label={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              testID={action.testID}
              variant={action.destructive ? "destructive" : "outline"}
            />
          ))}
          <Button label="Annuler" onPress={onClose} variant="ghost" />
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 560,
    backgroundColor: colors.background,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.lg,
    paddingBottom: spacing.xxl,
    gap: spacing.sm
  },
  handle: { alignSelf: "center", width: 40, height: 4, borderRadius: 2, backgroundColor: colors.borderStrong, marginBottom: spacing.sm }
});
