import React from "react";
import { Modal, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "../ui";

type Props = {
  visible: boolean;
  title?: string;
  onClose: () => void;
  children: React.ReactNode;
  testID?: string;
};

/** Bottom sheet used for menus and short forms (works on native and web). */
export function Sheet({ visible, title, onClose, children, testID }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <View style={styles.root}>
        <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[StyleSheet.absoluteFill, { backgroundColor: colors.overlay }]} />
        <View style={[styles.sheet, { backgroundColor: colors.surfaceRaised, paddingBottom: insets.bottom + spacing.lg }]} testID={testID}>
          <View style={[styles.grabber, { backgroundColor: colors.borderStrong }]} />
          {title ? <Text variant="heading">{title}</Text> : null}
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            {children}
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    gap: spacing.sm,
    maxHeight: "88%",
    width: "100%",
    maxWidth: 640,
    alignSelf: "center"
  },
  grabber: { width: 40, height: 4, borderRadius: 2, alignSelf: "center", marginBottom: spacing.xs },
  content: { gap: spacing.xs, paddingBottom: spacing.sm }
});

/**
 * Waits for a dismissed modal to finish its exit animation before presenting
 * another one (iOS cannot present two modals at once; on web the second one
 * would otherwise render beneath the first).
 */
export function afterModalDismiss(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, Platform.OS === "ios" ? 400 : 80));
}
