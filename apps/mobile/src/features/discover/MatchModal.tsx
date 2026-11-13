import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { MatchInfo } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, Button, Text } from "../../ui";

type Props = {
  match: MatchInfo | null;
  myPhotoUrl?: string | null;
  myName: string;
  onMessage: () => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  const { colors, spacing } = useTheme();
  if (!match) {
    return null;
  }
  return (
    <Modal visible transparent animationType="fade" onRequestClose={onClose}>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]} testID="match-modal" accessibilityViewIsModal>
        <View style={{ gap: spacing.xl, alignItems: "center", width: "100%", maxWidth: 420 }}>
          <Text variant="display" align="center" style={styles.white} accessibilityRole="header">
            C’est réciproque !
          </Text>
          <Text align="center" style={styles.whiteMuted}>
            Vous et {match.user.firstName} vous plaisez mutuellement.
          </Text>
          <View style={styles.avatars}>
            <View style={[styles.ring, { borderColor: colors.primary }]}>
              <Avatar url={myPhotoUrl} name={myName} size={112} />
            </View>
            <View style={[styles.ring, styles.overlap, { borderColor: colors.primary }]}>
              <Avatar url={match.user.photos[0]?.url} name={match.user.firstName} size={112} />
            </View>
          </View>
          <View style={{ gap: spacing.sm, alignSelf: "stretch" }}>
            <Button
              label={`Écrire à ${match.user.firstName}`}
              icon="chatbubble-ellipses-outline"
              onPress={onMessage}
              testID="match-message"
            />
            <Button label="Continuer à découvrir" variant="secondary" onPress={onClose} testID="match-close" />
          </View>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24 },
  white: { color: "#FFFFFF" },
  whiteMuted: { color: "rgba(255,255,255,0.85)" },
  avatars: { flexDirection: "row", alignItems: "center" },
  ring: { borderWidth: 4, borderRadius: 64, padding: 3 },
  overlap: { marginLeft: -24 }
});
