import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { colors, layout, radii, spacing } from "../theme/tokens";
import { AppText } from "./AppText";
import { IconButton } from "./IconButton";

type Props = { visible: boolean; title: string; onClose: () => void; children: ReactNode };

export function Sheet({ visible, title, onClose, children }: Props) {
  const insets = useSafeAreaInsets();
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <Pressable style={styles.backdrop} onPress={onClose} accessibilityLabel="Fermer" />
        <View style={[styles.sheet, { paddingBottom: insets.bottom + spacing.md }]}>
          <View style={styles.header}>
            <AppText variant="heading" style={styles.title}>
              {title}
            </AppText>
            <IconButton icon="close" label="Fermer" size={36} onPress={onClose} />
          </View>
          <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={styles.content}>
            {children}
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1, justifyContent: "flex-end" },
  backdrop: { ...StyleSheet.absoluteFillObject, backgroundColor: colors.overlay },
  sheet: {
    maxHeight: "88%",
    width: "100%",
    maxWidth: layout.maxContentWidth + 40,
    alignSelf: "center",
    backgroundColor: colors.surface,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    borderWidth: 1,
    borderColor: colors.border
  },
  header: { flexDirection: "row", alignItems: "center", padding: spacing.lg, paddingBottom: spacing.sm },
  title: { flex: 1 },
  content: { paddingHorizontal: spacing.lg, gap: spacing.md, paddingBottom: spacing.md }
});
