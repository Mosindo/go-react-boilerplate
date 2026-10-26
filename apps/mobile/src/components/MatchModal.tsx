import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { MatchView } from "../api";
import { radius, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Photo } from "./Photo";
import { Text } from "./Text";

type Props = { match: MatchView | null; onChat: () => void; onClose: () => void };

/** Celebration shown when two people like each other. */
export function MatchModal({ match, onChat, onClose }: Props) {
  const theme = useTheme();
  return (
    <Modal visible={!!match} transparent animationType="fade" onRequestClose={onClose}>
      <View style={[styles.backdrop, { backgroundColor: theme.overlay }]}>
        <View
          style={[styles.card, { backgroundColor: theme.surface }]}
          accessibilityViewIsModal
          testID="match-modal"
        >
          <Text variant="display" color={theme.primary} center>
            C&apos;est un match !
          </Text>
          <Text muted center>
            Vous et {match?.profile.firstName} vous plaisez mutuellement.
          </Text>
          <Photo
            url={match?.profile.photos[0]?.url}
            label={`Photo de ${match?.profile.firstName ?? ""}`}
            style={styles.photo}
          />
          <Button label={`Écrire à ${match?.profile.firstName ?? ""}`} onPress={onChat} testID="match-chat" />
          <Button label="Continuer à découvrir" variant="ghost" onPress={onClose} testID="match-close" />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: { width: "100%", maxWidth: 420, borderRadius: radius.lg, padding: spacing.xl, gap: spacing.md },
  photo: { width: "100%", aspectRatio: 1, borderRadius: radius.lg, overflow: "hidden" }
});
