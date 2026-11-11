import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { MatchPayload } from "../../api/types";
import { useMyProfile } from "../../hooks/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, Button, Text } from "../../ui";

type Props = {
  match: MatchPayload | null;
  onMessage: (m: MatchPayload) => void;
  onClose: () => void;
};

export function MatchModal({ match, onMessage, onClose }: Props) {
  const { colors, radii } = useTheme();
  const me = useMyProfile();
  return (
    <Modal transparent visible={!!match} animationType="fade" onRequestClose={onClose}>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]}>
        {match ? (
          <View
            testID="match-modal"
            style={[styles.card, { backgroundColor: colors.surfaceRaised, borderRadius: radii.xl }]}
          >
            <Text variant="overline" tone="primary" align="center">
              C’est réciproque
            </Text>
            <Text variant="display" align="center">
              Nouveau match !
            </Text>
            <View style={styles.avatars}>
              <View style={[styles.ring, { borderColor: colors.surfaceRaised }]}>
                <Avatar url={me.data?.photos[0]?.url} size={104} label="Toi" />
              </View>
              <View style={[styles.ring, styles.overlap, { borderColor: colors.surfaceRaised }]}>
                <Avatar url={match.user.photos[0]?.url} size={104} label={match.user.firstName} />
              </View>
            </View>
            <Text tone="muted" align="center">
              Toi et {match.user.firstName} vous plaisez mutuellement. Pourquoi ne pas lui écrire le premier message ?
            </Text>
            <Button
              testID="match-send-message"
              label={`Écrire à ${match.user.firstName}`}
              icon="chatbubble-outline"
              onPress={() => onMessage(match)}
            />
            <Button testID="match-continue" label="Continuer à découvrir" variant="ghost" onPress={onClose} />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: 20 },
  card: { width: "100%", maxWidth: 420, padding: 24, gap: 16 },
  avatars: { flexDirection: "row", justifyContent: "center", marginVertical: 8 },
  ring: { borderWidth: 4, borderRadius: 60 },
  overlap: { marginLeft: -24 }
});
