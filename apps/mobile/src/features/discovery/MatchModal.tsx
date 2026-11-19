import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { Match } from "../../api/types";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/tokens";
import { Avatar, Button, Text } from "../../ui";

type Props = { match: Match | null; onClose: () => void; onMessage: (match: Match) => void };

export function MatchModal({ match, onClose, onMessage }: Props) {
  const { colors } = useTheme();
  const { profile } = useSession();
  if (!match) return null;
  return (
    <Modal visible animationType="fade" transparent onRequestClose={onClose}>
      <View style={[styles.root, { backgroundColor: colors.scrim }]}>
        <View style={[styles.card, { backgroundColor: colors.surfaceRaised }]} accessibilityViewIsModal testID="match-modal">
          <Text variant="caption" tone="primary" align="center" style={styles.kicker}>
            NOUVEAU MATCH
          </Text>
          <Text variant="display" align="center">
            C’est réciproque !
          </Text>
          <View style={styles.avatars}>
            <Avatar path={profile?.photos[0]?.url} name={profile?.firstName ?? "Vous"} size={96} ring />
            <View style={styles.overlap}>
              <Avatar path={match.user.photos[0]?.url} name={match.user.firstName} size={96} ring />
            </View>
          </View>
          <Text tone="muted" align="center">
            Vous et {match.user.firstName} vous plaisez mutuellement. Lancez la conversation !
          </Text>
          <Button
            label={`Écrire à ${match.user.firstName}`}
            icon="chatbubble-ellipses-outline"
            onPress={() => onMessage(match)}
            testID="match-message"
          />
          <Button label="Continuer à découvrir" variant="ghost" onPress={onClose} testID="match-close" />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  card: { width: "100%", maxWidth: 420, borderRadius: 32, padding: spacing.xl, gap: spacing.md },
  kicker: { letterSpacing: 2 },
  avatars: { flexDirection: "row", justifyContent: "center", marginVertical: spacing.sm },
  overlap: { marginLeft: -18 }
});
