import React from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  visible: boolean;
  title: string;
  onClose: () => void;
  children: React.ReactNode;
};

/** Bottom sheet modal used for reports, confirmations and pickers. */
export function Sheet({ visible, title, onClose, children }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.fill}>
        <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.fill, { backgroundColor: colors.overlay }]} />
        <View style={[styles.sheet, { backgroundColor: colors.surface, paddingBottom: insets.bottom + spacing.md }]}>
          <View style={styles.header}>
            <Text accessibilityRole="header" style={styles.title} variant="heading">
              {title}
            </Text>
            <IconButton icon="close" label="Fermer" onPress={onClose} size={38} />
          </View>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            {children}
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  sheet: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    maxHeight: "88%",
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    paddingTop: spacing.sm,
    width: "100%",
    maxWidth: 640,
    alignSelf: "center"
  },
  header: { flexDirection: "row", alignItems: "center", paddingHorizontal: spacing.lg },
  title: { flex: 1 },
  content: { paddingHorizontal: spacing.lg, paddingBottom: spacing.md, gap: spacing.md }
});
