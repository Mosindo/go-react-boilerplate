import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { Match } from "../api/types";
import { Button, Text, colors, radii, spacing } from "../shared/ui";
import { PersonAvatar } from "./PersonAvatar";

type MatchOverlayProps = {
  match: Match | null;
  onChat: (match: Match) => void;
  onDismiss: () => void;
};

export function MatchOverlay({ match, onChat, onDismiss }: MatchOverlayProps) {
  return (
    <Modal animationType="fade" onRequestClose={onDismiss} transparent visible={match !== null}>
      <View style={styles.backdrop}>
        {match ? (
          <View accessibilityViewIsModal style={styles.card} testID="match-overlay">
            <Text tone="primary" variant="eyebrow" weight="bold">
              Un match !
            </Text>
            <Text style={styles.center} variant="title" weight="bold">
              Vous plaisez à {match.user.firstName}
            </Text>
            <PersonAvatar name={match.user.firstName} photo={match.user.photo} size={112} />
            <Text style={styles.center} tone="muted">
              C&apos;est réciproque. Engagez la conversation, c&apos;est gratuit et sans limite.
            </Text>
            <View style={styles.actions}>
              <Button fullWidth label="Envoyer un message" onPress={() => onChat(match)} size="lg" testID="match-chat" />
              <Button fullWidth label="Continuer à découvrir" onPress={onDismiss} variant="ghost" testID="match-dismiss" />
            </View>
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: {
    width: "100%",
    maxWidth: 400,
    borderRadius: radii.xl,
    backgroundColor: colors.backgroundElevated,
    padding: spacing.xl,
    alignItems: "center",
    gap: spacing.lg
  },
  center: { textAlign: "center" },
  actions: { width: "100%", gap: spacing.xs }
});
