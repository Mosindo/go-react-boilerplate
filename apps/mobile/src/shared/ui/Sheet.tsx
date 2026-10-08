import React, { type ReactNode } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import type { IconName } from "./Button";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type SheetAction = { label: string; icon?: IconName; destructive?: boolean; onPress: () => void };

type SheetProps = { visible: boolean; title?: string; onClose: () => void; actions?: SheetAction[]; children?: ReactNode };

/** Bottom sheet used for action menus and small forms; works the same on native and web. */
export function Sheet({ visible, title, onClose, actions, children }: SheetProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.backdrop, { backgroundColor: colors.overlay }]} />
      <View style={styles.sheetWrap} pointerEvents="box-none">
        <View style={[styles.sheet, { backgroundColor: colors.surface }]}>
          <View style={[styles.handle, { backgroundColor: colors.border }]} />
          {title ? (
            <Text accessibilityRole="header" variant="heading">
              {title}
            </Text>
          ) : null}
          {actions?.map((action) => (
            <Pressable
              accessibilityLabel={action.label}
              accessibilityRole="button"
              key={action.label}
              onPress={() => {
                onClose();
                action.onPress();
              }}
              style={({ pressed }) => [styles.action, pressed ? { opacity: 0.6 } : null]}
            >
              {action.icon ? <Ionicons color={action.destructive ? colors.danger : colors.textMuted} name={action.icon} size={22} /> : null}
              <Text style={{ color: action.destructive ? colors.danger : colors.text }} weight="medium">
                {action.label}
              </Text>
            </Pressable>
          ))}
          {children}
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { ...StyleSheet.absoluteFillObject },
  sheetWrap: { flex: 1, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 560,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    paddingBottom: spacing.xxl,
    gap: spacing.xs
  },
  handle: { alignSelf: "center", width: 40, height: 4, borderRadius: 2, marginBottom: spacing.sm },
  action: { flexDirection: "row", alignItems: "center", gap: spacing.md, minHeight: 52 }
});
