import React from "react";
import { Modal, Platform, StyleSheet, View } from "react-native";
import type { CardSummary } from "../../api/types";
import { useTheme } from "../../theme";
import { Avatar, Button, Text } from "../../ui";

type Props = {
  match: { user: CardSummary; conversationId: string } | null;
  myPhotoUrl?: string | null;
  myName: string;
  onMessage: (conversationId: string) => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  const { colors, spacing } = useTheme();
  return (
    <Modal animationType={Platform.OS === "web" ? "none" : "fade"} onRequestClose={onClose} transparent visible={!!match}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay, padding: spacing.xl }]}>
        {match ? (
          <View
            accessibilityViewIsModal
            style={[styles.card, { backgroundColor: colors.surfaceRaised, borderColor: colors.border, padding: spacing.xl, gap: spacing.md }]}
          >
            <Text center tone="accent" variant="eyebrow">
              C’est réciproque
            </Text>
            <Text accessibilityRole="header" center testID="match-title" variant="display">
              C’est un match !
            </Text>
            <View style={styles.avatars}>
              <Avatar name={myName} ring size={96} uri={myPhotoUrl} />
              <View style={{ marginLeft: -18 }}>
                <Avatar name={match.user.firstName} ring size={96} uri={match.user.photoUrl} />
              </View>
            </View>
            <Text center tone="muted">
              Toi et {match.user.firstName} vous plaisez mutuellement. Lance la conversation !
            </Text>
            <Button
              icon="chatbubble-ellipses-outline"
              label={`Écrire à ${match.user.firstName}`}
              onPress={() => onMessage(match.conversationId)}
              testID="match-message"
            />
            <Button label="Continuer à découvrir" onPress={onClose} variant="ghost" />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "center" },
  card: { borderRadius: 28, borderWidth: 1, width: "100%", maxWidth: 420, alignSelf: "center" },
  avatars: { flexDirection: "row", justifyContent: "center", marginVertical: 8 }
});
