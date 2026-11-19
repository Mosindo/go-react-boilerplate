import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/tokens";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export type SheetAction = { label: string; icon: IconName; onPress: () => void; destructive?: boolean; testID?: string };

type Props = { visible: boolean; title?: string; actions: SheetAction[]; onClose: () => void };

export function ActionSheet({ visible, title, actions, onClose }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal transparent visible={visible} animationType="slide" onRequestClose={onClose}>
      <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={onClose} accessibilityLabel="Fermer le menu">
        <Pressable
          onPress={() => undefined}
          style={[styles.sheet, { backgroundColor: colors.surfaceRaised, paddingBottom: insets.bottom + spacing.md }]}
        >
          <View style={[styles.grabber, { backgroundColor: colors.borderStrong }]} />
          {title ? (
            <Text variant="label" tone="muted" align="center" style={styles.title}>
              {title}
            </Text>
          ) : null}
          {actions.map((a) => (
            <Pressable
              key={a.label}
              accessibilityRole="button"
              testID={a.testID}
              onPress={() => {
                onClose();
                a.onPress();
              }}
              style={({ pressed }) => [styles.action, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
            >
              <Icon name={a.icon} size={21} color={a.destructive ? colors.danger : colors.text} />
              <Text tone={a.destructive ? "danger" : "default"}>{a.label}</Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" onPress={onClose} style={styles.action}>
            <Icon name="close" size={21} color={colors.textMuted} />
            <Text tone="muted">Annuler</Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    borderTopLeftRadius: radii.lg,
    borderTopRightRadius: radii.lg,
    paddingTop: spacing.xs,
    paddingHorizontal: spacing.md,
    width: "100%",
    maxWidth: 640,
    alignSelf: "center"
  },
  grabber: { width: 40, height: 4, borderRadius: 2, alignSelf: "center", marginBottom: spacing.sm },
  title: { marginBottom: spacing.xs },
  action: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingVertical: 14,
    paddingHorizontal: spacing.xs,
    borderRadius: radii.sm
  }
});
