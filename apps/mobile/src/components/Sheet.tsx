import React from "react";
import { Modal, Pressable, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  visible: boolean;
  title: string;
  onClose: () => void;
  children: React.ReactNode;
};

/** Bottom sheet built on Modal: no extra gesture dependency. */
export function Sheet({ visible, title, onClose, children }: Props) {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable
        accessibilityLabel="Fermer"
        onPress={onClose}
        style={[styles.backdrop, { backgroundColor: theme.overlay }]}
      />
      <View
        style={[
          styles.sheet,
          { backgroundColor: theme.background, paddingBottom: insets.bottom + spacing.lg }
        ]}
      >
        <View style={[styles.handle, { backgroundColor: theme.border }]} />
        <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
          {title}
        </Text>
        {children}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1 },
  sheet: {
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.lg,
    maxHeight: "85%"
  },
  handle: { alignSelf: "center", width: 40, height: 4, borderRadius: 2, marginBottom: spacing.md },
  title: { fontSize: fontSizes.lg, fontWeight: "700", marginBottom: spacing.md }
});
