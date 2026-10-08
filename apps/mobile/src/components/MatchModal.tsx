import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import { useProfile } from "../hooks/data";
import { useRealtime } from "../hooks/useRealtime";
import { navigationRef } from "../navigation/ref";
import { Button, PhotoImage, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

/** Celebration shown when a match happens, from either the swipe response or a push. */
export function MatchModal() {
  const { colors } = useTheme();
  const { incomingMatch, dismissMatch } = useRealtime();
  const { data: profile } = useProfile(incomingMatch?.userId ?? "");

  if (!incomingMatch) return null;

  const openChat = () => {
    const match = incomingMatch;
    dismissMatch();
    if (navigationRef.isReady() && profile) {
      navigationRef.navigate("Chat", {
        conversationId: match.conversationId,
        matchId: match.matchId,
        user: { id: profile.id, firstName: profile.firstName, photoPath: profile.photos[0]?.thumbUrl }
      });
    }
  };

  return (
    <Modal animationType="fade" onRequestClose={dismissMatch} transparent visible>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]}>
        <View accessibilityViewIsModal style={[styles.card, { backgroundColor: colors.surface }]}>
          <Text accessibilityRole="header" tone="primary" variant="title">
            C'est un match !
          </Text>
          {profile ? (
            <>
              <PhotoImage label={profile.firstName} path={profile.photos[0]?.thumbUrl} style={styles.photo} />
              <Text align="center" tone="muted">
                Vous et {profile.firstName} vous plaisez mutuellement.
              </Text>
            </>
          ) : null}
          <Button fullWidth icon="chatbubble-ellipses-outline" onPress={openChat} size="lg">
            Envoyer un message
          </Button>
          <Button fullWidth onPress={dismissMatch} variant="ghost">
            Continuer à découvrir
          </Button>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: { width: "100%", maxWidth: 380, borderRadius: radii.xl, padding: spacing.xl, alignItems: "center", gap: spacing.md },
  photo: { width: 160, height: 160, borderRadius: 80 }
});
