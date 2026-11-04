import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { radius, spacing, useColors } from "./theme";

type Props = { visible: boolean; title?: string; onClose: () => void; children: React.ReactNode; testID?: string };

/** Bottom sheet / dialog container. Tapping the backdrop or the system back button closes it. */
export function Sheet({ visible, title, onClose, children, testID }: Props) {
  const c = useColors();
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose} statusBarTranslucent>
      <Pressable accessibilityLabel="Fermer" style={[styles.backdrop, { backgroundColor: c.overlay }]} onPress={onClose} />
      <View testID={testID} style={[styles.sheet, { backgroundColor: c.surface }]}>
        {title ? <Text variant="heading" style={styles.title}>{title}</Text> : null}
        {children}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { ...StyleSheet.absoluteFillObject },
  sheet: { position: "absolute", left: 0, right: 0, bottom: 0, borderTopLeftRadius: radius.lg, borderTopRightRadius: radius.lg, padding: spacing.xl, paddingBottom: spacing.xxl, gap: spacing.md, maxHeight: "85%" },
  title: { marginBottom: spacing.xs },
});
