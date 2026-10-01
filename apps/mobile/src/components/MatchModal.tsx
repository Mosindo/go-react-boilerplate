import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { MatchSummary } from "../api/types";
import { Avatar, Button, Text, radii, spacing, useTheme } from "../shared/ui";

type MatchModalProps = {
  match: MatchSummary | null;
  onMessage: (match: MatchSummary) => void;
  onClose: () => void;
};

export function MatchModal({ match, onMessage, onClose }: MatchModalProps) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={match !== null}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        {match ? (
          <View accessibilityViewIsModal style={[styles.card, { backgroundColor: colors.surface }]}>
            <Text align="center" tone="primary" variant="eyebrow" weight="bold">
              Réciproque
            </Text>
            <Text align="center" variant="title">
              C&apos;est un match !
            </Text>
            <Avatar name={match.user.firstName} path={match.user.photos[0]?.url} size="lg" />
            <Text align="center" tone="muted">
              Vous et {match.user.firstName} vous plaisez mutuellement. Qui ose le premier message ?
            </Text>
            <Button label={`Écrire à ${match.user.firstName}`} onPress={() => onMessage(match)} />
            <Button label="Continuer à découvrir" onPress={onClose} variant="ghost" />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: { width: "100%", maxWidth: 380, borderRadius: radii.xl, padding: spacing.xl, alignItems: "center", gap: spacing.md }
});
