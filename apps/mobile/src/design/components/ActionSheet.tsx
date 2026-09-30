import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../ThemeProvider";
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

/** Cross-platform bottom sheet for choices and confirmations. */
export function ActionSheet({ visible, title, message, actions, onClose }: Props) {
  const { colors, radii } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable
          accessibilityViewIsModal
          style={[
            styles.sheet,
            { backgroundColor: colors.surfaceRaised, borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl, paddingBottom: insets.bottom + 12 }
          ]}
        >
          {title ? (
            <Text align="center" variant="heading">
              {title}
            </Text>
          ) : null}
          {message ? (
            <Text align="center" tone="muted">
              {message}
            </Text>
          ) : null}
          <View style={styles.actions}>
            {actions.map((action) => (
              <Pressable
                accessibilityRole="button"
                key={action.label}
                onPress={() => {
                  onClose();
                  action.onPress();
                }}
                style={({ pressed }) => [styles.action, { backgroundColor: pressed ? colors.surfaceMuted : colors.surface, borderRadius: radii.md }]}
                testID={action.testID}
              >
                <Text style={{ color: action.destructive ? colors.danger : colors.text }} variant="label">
                  {action.label}
                </Text>
              </Pressable>
            ))}
            <Pressable accessibilityRole="button" onPress={onClose} style={styles.action}>
              <Text tone="muted" variant="label">
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
  sheet: { padding: 20, gap: 8, width: "100%", maxWidth: 640, alignSelf: "center" },
  actions: { gap: 8, marginTop: 8 },
  action: { minHeight: 52, alignItems: "center", justifyContent: "center", paddingHorizontal: 16 }
});
