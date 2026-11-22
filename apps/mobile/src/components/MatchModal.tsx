import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { Match } from "../api/types";
import { Avatar, Button, Text, colors, radii, spacing } from "../ui";

type Props = {
  match: Match | null;
  myPhotoUrl?: string;
  myName: string;
  onMessage: (match: Match) => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={!!match}>
      <View style={styles.backdrop}>
        {match ? (
          <View accessibilityViewIsModal style={styles.sheet} testID="match-modal">
            <Text tone="accent" variant="eyebrow">
              C’est réciproque
            </Text>
            <Text align="center" variant="display">
              Nouveau match
            </Text>
            <View style={styles.avatars}>
              <Avatar name={myName} ring size={96} url={myPhotoUrl} />
              <View style={styles.spark} />
              <Avatar name={match.user.firstName} ring size={96} url={match.user.photos[0]?.url} />
            </View>
            <Text align="center" tone="muted">
              Vous et {match.user.firstName} vous plaisez. Lancez la conversation !
            </Text>
            <View style={styles.actions}>
              <Button label={`Écrire à ${match.user.firstName}`} onPress={() => onMessage(match)} testID="match-message" />
              <Button label="Continuer à découvrir" onPress={onClose} testID="match-close" variant="ghost" />
            </View>
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay,
    alignItems: "center",
    justifyContent: "center",
    padding: spacing.lg
  },
  sheet: {
    width: "100%",
    maxWidth: 420,
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.xl,
    borderRadius: radii.xl,
    backgroundColor: colors.surface,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.borderStrong
  },
  avatars: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    marginVertical: spacing.sm
  },
  spark: {
    width: 14,
    height: 14,
    borderRadius: 7,
    backgroundColor: colors.accent
  },
  actions: {
    alignSelf: "stretch",
    gap: spacing.xs,
    marginTop: spacing.sm
  }
});
