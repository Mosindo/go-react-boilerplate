import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

export type SheetAction = {
  label: string;
  icon: keyof typeof Ionicons.glyphMap;
  destructive?: boolean;
  onPress: () => void;
  testID?: string;
};

type SheetProps = {
  visible: boolean;
  title?: string;
  actions?: SheetAction[];
  onClose: () => void;
  children?: React.ReactNode;
};

/** Cross-platform bottom sheet used for contextual actions and dialogs. */
export function Sheet({ visible, title, actions, onClose, children }: SheetProps) {
  const { colors, spacing, radii } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable
        style={[styles.backdrop, { backgroundColor: colors.overlay }]}
        onPress={onClose}
        accessibilityLabel="Fermer"
      />
      <View
        style={[
          styles.sheet,
          {
            backgroundColor: colors.surface,
            borderTopLeftRadius: radii.xl,
            borderTopRightRadius: radii.xl,
            paddingBottom: insets.bottom + spacing.md,
            paddingHorizontal: spacing.lg,
            gap: spacing.xs
          }
        ]}
        accessibilityViewIsModal
      >
        <View style={[styles.grabber, { backgroundColor: colors.borderStrong }]} />
        {title ? (
          <Text variant="heading" style={{ marginBottom: spacing.xs }}>
            {title}
          </Text>
        ) : null}
        {children}
        {actions?.map((action) => (
          <Pressable
            key={action.label}
            testID={action.testID}
            accessibilityRole="button"
            onPress={() => {
              onClose();
              action.onPress();
            }}
            style={({ pressed }) => [
              styles.action,
              { borderRadius: radii.md, backgroundColor: pressed ? colors.surfaceAlt : "transparent" }
            ]}
          >
            <Ionicons name={action.icon} size={22} color={action.destructive ? colors.danger : colors.text} />
            <Text weight="medium" tone={action.destructive ? "danger" : "default"}>
              {action.label}
            </Text>
          </Pressable>
        ))}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { ...StyleSheet.absoluteFillObject },
  sheet: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    paddingTop: 10,
    maxWidth: 640,
    alignSelf: "center",
    width: "100%"
  },
  grabber: { width: 40, height: 5, borderRadius: 3, alignSelf: "center", marginBottom: 12 },
  action: { flexDirection: "row", alignItems: "center", gap: 14, paddingVertical: 14, paddingHorizontal: 8 }
});
