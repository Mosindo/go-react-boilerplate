import React from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { MIN_TOUCH_TARGET, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type ActionSheetOption = {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  testID?: string;
};

type Props = {
  visible: boolean;
  title?: string;
  options: ActionSheetOption[];
  onClose: () => void;
};

export function ActionSheet({ visible, title, options, onClose }: Props) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} statusBarTranslucent transparent visible={visible}>
      <Pressable
        accessibilityLabel="Fermer le menu"
        accessibilityRole="button"
        onPress={onClose}
        style={[styles.backdrop, { backgroundColor: colors.overlay }]}
      >
        <SafeAreaView edges={["bottom"]} style={styles.safe}>
          <View style={[styles.sheet, { backgroundColor: colors.surface }]}>
            {title ? (
              <Text color="textMuted" style={styles.title} variant="caption">
                {title}
              </Text>
            ) : null}
            {options.map((option) => (
              <Pressable
                accessibilityRole="button"
                key={option.label}
                onPress={() => {
                  onClose();
                  option.onPress();
                }}
                style={({ pressed }) => [styles.option, { opacity: pressed ? 0.6 : 1, borderColor: colors.border }]}
                testID={option.testID}
              >
                <Text color={option.destructive ? "danger" : "text"} variant="label">
                  {option.label}
                </Text>
              </Pressable>
            ))}
            <Pressable accessibilityRole="button" onPress={onClose} style={styles.option}>
              <Text color="textMuted" variant="label">
                Annuler
              </Text>
            </Pressable>
          </View>
        </SafeAreaView>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  safe: { paddingHorizontal: spacing.md },
  sheet: { borderRadius: radius.lg, overflow: "hidden", marginBottom: spacing.sm },
  title: { textAlign: "center", padding: spacing.md },
  option: {
    minHeight: MIN_TOUCH_TARGET + 6,
    alignItems: "center",
    justifyContent: "center",
    borderTopWidth: StyleSheet.hairlineWidth
  }
});
