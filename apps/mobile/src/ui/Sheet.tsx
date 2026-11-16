import React from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { colors, layout, radii, spacing } from "../theme/tokens";
import { Button } from "./Button";
import { Text } from "./Text";

export type SheetProps = {
  visible: boolean;
  title?: string;
  onClose: () => void;
  children: React.ReactNode;
  testID?: string;
};

/** Bottom sheet built on RN Modal: works on iOS, Android and web. */
export function Sheet({ visible, title, onClose, children, testID }: SheetProps) {
  const insets = useSafeAreaInsets();
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <KeyboardAvoidingView style={styles.root} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <Pressable style={StyleSheet.absoluteFill} onPress={onClose} accessibilityLabel="Fermer" accessibilityRole="button">
          <View style={[StyleSheet.absoluteFill, { backgroundColor: colors.scrim }]} />
        </Pressable>
        <View style={[styles.sheet, { paddingBottom: insets.bottom + spacing.lg }]} testID={testID}>
          <View style={styles.handle} />
          {title ? (
            <Text variant="heading" style={styles.title}>
              {title}
            </Text>
          ) : null}
          <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={styles.content}>
            {children}
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

export type ConfirmDialogProps = {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children?: React.ReactNode;
};

export function ConfirmDialog({ visible, title, message, confirmLabel, destructive, loading, onConfirm, onCancel, children }: ConfirmDialogProps) {
  return (
    <Sheet visible={visible} title={title} onClose={onCancel}>
      <Text tone="muted">{message}</Text>
      {children}
      <View style={styles.actions}>
        <Button label={confirmLabel} variant={destructive ? "danger" : "primary"} onPress={onConfirm} loading={loading} testID="confirm-dialog-confirm" />
        <Button label="Annuler" variant="ghost" onPress={onCancel} />
      </View>
    </Sheet>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    backgroundColor: colors.surface,
    borderTopLeftRadius: radii.xxl,
    borderTopRightRadius: radii.xxl,
    paddingHorizontal: spacing.xl,
    paddingTop: spacing.md,
    maxHeight: "88%",
    width: "100%",
    maxWidth: layout.maxContentWidth + 80,
    alignSelf: "center"
  },
  handle: { alignSelf: "center", width: 40, height: 4, borderRadius: 2, backgroundColor: colors.borderStrong, marginBottom: spacing.md },
  title: { marginBottom: spacing.md },
  content: { gap: spacing.md, paddingBottom: spacing.md },
  actions: { gap: spacing.sm, marginTop: spacing.md }
});
