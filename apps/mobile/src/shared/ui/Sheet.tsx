import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { IconButton } from "./IconButton";
import { Text } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type SheetProps = {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  root: { flex: 1, justifyContent: "flex-end" as const },
  backdrop: {
    ...({ position: "absolute", top: 0, left: 0, right: 0, bottom: 0 } as const),
    backgroundColor: t.colors.overlay
  },
  sheet: {
    maxHeight: "88%" as const,
    backgroundColor: t.colors.backgroundElevated,
    borderTopLeftRadius: t.radii.xl,
    borderTopRightRadius: t.radii.xl,
    paddingTop: t.spacing.md,
    paddingHorizontal: t.spacing.lg
  },
  header: {
    flexDirection: "row" as const,
    alignItems: "center" as const,
    justifyContent: "space-between" as const,
    marginBottom: t.spacing.sm
  },
  title: { flex: 1 },
  body: { gap: t.spacing.md, paddingBottom: t.spacing.lg }
});

/** Bottom sheet built on the RN Modal (no extra dependency). */
export function Sheet({ children, onClose, testID, title, visible }: SheetProps) {
  const styles = useThemedStyles(makeStyles);
  const insets = useSafeAreaInsets();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        style={styles.root}
      >
        <Pressable
          accessibilityLabel="Close"
          accessibilityRole="button"
          onPress={onClose}
          style={styles.backdrop}
        />
        <View
          accessibilityViewIsModal
          style={[styles.sheet, { paddingBottom: insets.bottom + 8 }]}
          testID={testID}
        >
          <View style={styles.header}>
            <Text accessibilityRole="header" style={styles.title} variant="heading" weight="bold">
              {title ?? ""}
            </Text>
            <IconButton accessibilityLabel="Close" glyph="×" onPress={onClose} />
          </View>
          <ScrollView contentContainerStyle={styles.body} keyboardShouldPersistTaps="handled">
            {children}
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}
