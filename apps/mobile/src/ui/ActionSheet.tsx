import React from "react";
import { Modal, Platform, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme";
import { Text } from "./Text";

export type SheetAction = {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  testID?: string;
};

type Props = {
  visible: boolean;
  title?: string;
  message?: string;
  actions: SheetAction[];
  onClose: () => void;
};

/** Cross-platform bottom sheet (Alert with buttons is a no-op on web). */
export function ActionSheet({ visible, title, message, actions, onClose }: Props) {
  const { colors, radii, spacing } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal animationType={Platform.OS === "web" ? "none" : "fade"} onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable
          onPress={() => undefined}
          style={[
            styles.sheet,
            {
              backgroundColor: colors.surfaceRaised,
              borderTopLeftRadius: radii.xl,
              borderTopRightRadius: radii.xl,
              paddingBottom: insets.bottom + spacing.md,
              padding: spacing.lg
            }
          ]}
        >
          {title ? <Text variant="heading">{title}</Text> : null}
          {message ? <Text tone="muted">{message}</Text> : null}
          <View style={{ gap: spacing.xs, marginTop: spacing.sm }}>
            {actions.map((action) => (
              <Pressable
                accessibilityRole="button"
                key={action.label}
                onPress={() => {
                  onClose();
                  action.onPress();
                }}
                style={({ pressed }) => [
                  styles.action,
                  { backgroundColor: pressed ? colors.surfaceAlt : colors.surface, borderRadius: radii.md, borderColor: colors.border }
                ]}
                testID={action.testID}
              >
                <Text style={{ color: action.destructive ? colors.danger : colors.text }} variant="bodyStrong">
                  {action.label}
                </Text>
              </Pressable>
            ))}
            <Pressable accessibilityRole="button" onPress={onClose} style={styles.action}>
              <Text tone="muted" variant="bodyStrong">
                Annuler
              </Text>
            </Pressable>
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { gap: 6, width: "100%", maxWidth: 640, alignSelf: "center" },
  action: { minHeight: 52, alignItems: "center", justifyContent: "center", borderWidth: StyleSheet.hairlineWidth, borderColor: "transparent" }
});
