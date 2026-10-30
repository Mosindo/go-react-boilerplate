import React, { type ReactNode } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { radii, spacing, useTheme } from "./theme";

type SheetProps = {
  visible: boolean;
  title?: string;
  onClose: () => void;
  children: ReactNode;
};

/** Bottom sheet used for menus and confirmations (Alert is unreliable on web). */
export function Sheet({ visible, title, onClose, children }: SheetProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable
        accessibilityLabel="Fermer"
        onPress={onClose}
        style={[styles.backdrop, { backgroundColor: colors.overlay }]}
      >
        <Pressable style={[styles.sheet, { backgroundColor: colors.surface }]} onPress={() => undefined}>
          {title ? (
            <Text variant="heading" style={styles.title}>
              {title}
            </Text>
          ) : null}
          <ScrollView bounces={false} contentContainerStyle={styles.body}>
            {children}
          </ScrollView>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

type SheetActionProps = { label: string; onPress: () => void; danger?: boolean; testID?: string };

export function SheetAction({ label, onPress, danger = false, testID }: SheetActionProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.action, { backgroundColor: pressed ? colors.surfaceAlt : "transparent" }]}
      testID={testID}
    >
      <Text variant="body" tone={danger ? "danger" : "default"}>
        {label}
      </Text>
    </Pressable>
  );
}

export function SheetBody({ children }: { children: ReactNode }) {
  return <View style={styles.custom}>{children}</View>;
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 560,
    maxHeight: "85%",
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    paddingTop: spacing.lg,
    paddingBottom: spacing.xl
  },
  title: { paddingHorizontal: spacing.xl, paddingBottom: spacing.sm },
  body: { paddingHorizontal: spacing.md },
  action: { minHeight: 52, justifyContent: "center", paddingHorizontal: spacing.md, borderRadius: radii.md },
  custom: { paddingHorizontal: spacing.md, gap: spacing.md, paddingBottom: spacing.sm }
});
