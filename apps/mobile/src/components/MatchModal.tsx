import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import { Avatar, Button, Text, radii, spacing, useTheme } from "../shared/ui";

type MatchModalProps = {
  visible: boolean;
  myName: string;
  myPhotoPath?: string;
  otherName: string;
  otherPhotoPath?: string;
  onMessage: () => void;
  onClose: () => void;
};

export function MatchModal({ visible, myName, myPhotoPath, otherName, otherPhotoPath, onMessage, onClose }: MatchModalProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <View accessibilityViewIsModal style={[styles.sheet, { backgroundColor: colors.surface }]} testID="match-modal">
          <Text tone="primary" variant="eyebrow" weight="bold">
            Vous vous plaisez mutuellement
          </Text>
          <Text style={styles.center} variant="title">
            C'est un match !
          </Text>
          <View style={styles.avatars}>
            <Avatar name={myName} photoPath={myPhotoPath} size={92} />
            <Avatar name={otherName} photoPath={otherPhotoPath} size={92} />
          </View>
          <Text style={styles.center} tone="muted">
            Dites bonjour à {otherName} : c'est gratuit, sans limite.
          </Text>
          <View style={styles.actions}>
            <Button label={`Écrire à ${otherName}`} onPress={onMessage} testID="match-modal-message" />
            <Button label="Continuer à découvrir" onPress={onClose} variant="ghost" />
          </View>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  sheet: { width: "100%", maxWidth: 380, borderRadius: radii.xl, padding: spacing.xl, alignItems: "center", gap: spacing.md },
  center: { textAlign: "center" },
  avatars: { flexDirection: "row", gap: spacing.md, marginVertical: spacing.md },
  actions: { width: "100%", gap: spacing.xs, marginTop: spacing.sm }
});
