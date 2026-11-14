import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { Match, Photo } from "../../api/types";
import { spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, Button, Icon, Text } from "../../ui";

type Props = {
  match: Match | null;
  myPhoto?: Photo | null;
  onMessage: (match: Match) => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhoto, onMessage, onClose }: Props) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={!!match}>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]}>
        {match ? (
          <View accessibilityViewIsModal style={styles.content} testID="match-modal">
            <Icon color={colors.primary} name="sparkles" size={34} />
            <Text center style={styles.white} variant="display">
              C’est réciproque !
            </Text>
            <Text center style={styles.soft}>
              Vous et {match.user.firstName} vous plaisez. À vous de faire le premier pas.
            </Text>
            <View style={styles.avatars}>
              <View style={[styles.ring, { borderColor: colors.primary }]}>
                <Avatar label="Votre photo" photo={myPhoto} size={112} />
              </View>
              <View style={[styles.ring, styles.overlap, { borderColor: colors.primary }]}>
                <Avatar label={`Photo de ${match.user.firstName}`} photo={match.user.photos[0]} size={112} />
              </View>
            </View>
            <View style={styles.actions}>
              <Button icon="chatbubble-ellipses-outline" label="Envoyer un message" onPress={() => onMessage(match)} size="lg" testID="match-message" />
              <Button label="Continuer à découvrir" onPress={onClose} style={styles.ghost} variant="ghost" />
            </View>
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  content: { alignItems: "center", gap: spacing.sm, width: "100%", maxWidth: 420 },
  white: { color: "#fff" },
  soft: { color: "rgba(255,255,255,0.85)" },
  avatars: { flexDirection: "row", marginVertical: spacing.xl },
  ring: { borderWidth: 3, borderRadius: 64, padding: 3 },
  overlap: { marginLeft: -24 },
  actions: { width: "100%", gap: spacing.xs },
  ghost: { backgroundColor: "rgba(255,255,255,0.08)" }
});
