import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type SheetProps = {
  visible: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
};

/** Bottom sheet modal used for short tasks (report, confirmations...). */
export function Sheet({ visible, title, onClose, children }: SheetProps) {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.backdrop, { backgroundColor: theme.colors.overlay }]} />
        <View
          style={[
            styles.sheet,
            {
              backgroundColor: theme.colors.background,
              paddingBottom: insets.bottom + 16,
              borderTopLeftRadius: theme.radii.xl,
              borderTopRightRadius: theme.radii.xl
            }
          ]}
        >
          <View style={styles.header}>
            <Text accessibilityRole="header" style={styles.flex} variant="title">
              {title}
            </Text>
            <IconButton icon="close" label="Fermer" onPress={onClose} />
          </View>
          <ScrollView keyboardShouldPersistTaps="handled">{children}</ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  backdrop: { ...StyleSheet.absoluteFillObject },
  sheet: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    maxHeight: "88%",
    paddingHorizontal: 20,
    paddingTop: 12,
    width: "100%",
    maxWidth: 640,
    marginHorizontal: "auto",
    alignSelf: "center"
  },
  header: { flexDirection: "row", alignItems: "center", paddingVertical: 8 }
});
