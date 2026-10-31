import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { MatchItem } from "../api/types";
import { Button, Text, colors, radii, spacing } from "../shared/ui";
import { RemotePhoto } from "./RemotePhoto";

type MatchModalProps = {
  match: MatchItem | null;
  onClose: () => void;
  onChat: (match: MatchItem) => void;
};

export function MatchModal({ match, onClose, onChat }: MatchModalProps) {
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={match !== null}>
      <View style={styles.backdrop}>
        {match ? (
          <View accessibilityViewIsModal style={styles.sheet}>
            <Text style={styles.center} tone="primary" variant="eyebrow" weight="bold">
              Il y a de l'étincelle
            </Text>
            <Text style={styles.center} variant="title" weight="bold">
              C'est un match !
            </Text>
            <RemotePhoto style={styles.photo} url={match.user.photos[0]?.url} />
            <Text style={styles.center} tone="muted">
              Vous et {match.user.firstName} vous plaisez mutuellement. Dites bonjour !
            </Text>
            <Button fullWidth label={`Écrire à ${match.user.firstName}`} onPress={() => onChat(match)} />
            <Button fullWidth label="Continuer à découvrir" onPress={onClose} variant="ghost" />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  sheet: {
    width: "100%",
    maxWidth: 420,
    backgroundColor: colors.background,
    borderRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md,
    alignItems: "center"
  },
  center: { textAlign: "center" },
  photo: { width: 160, height: 160, borderRadius: 80, overflow: "hidden" }
});
