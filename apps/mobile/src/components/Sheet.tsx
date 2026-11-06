import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { radius, spacing, useTheme } from "../theme";
import { Text } from "./ui";

/** Bottom sheet / dialog used for confirmations and pickers. */
export function Sheet({
  visible,
  onClose,
  title,
  children
}: {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}) {
  const { colors } = useTheme();
  return (
    <Modal
      visible={visible}
      transparent
      animationType="fade"
      onRequestClose={onClose}
      statusBarTranslucent
    >
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable style={StyleSheet.absoluteFill} onPress={onClose} accessibilityLabel="Fermer" />
        <View
          accessibilityViewIsModal
          style={[styles.sheet, { backgroundColor: colors.surface, borderColor: colors.border }]}
        >
          {title ? (
            <Text variant="heading" style={{ marginBottom: spacing.md }}>
              {title}
            </Text>
          ) : null}
          {children}
        </View>
      </View>
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
    borderWidth: 1,
    padding: spacing.xl,
    gap: spacing.md
  }
});
