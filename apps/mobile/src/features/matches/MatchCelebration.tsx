import React, { useEffect } from "react";
import { Animated, Modal, StyleSheet, View, useAnimatedValue } from "react-native";
import { queryKeys, type ProfileState } from "../../api";
import { useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Text } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";
import { navigationRef } from "../../navigation/ref";
import { dismissMatch, useAnnouncedMatch } from "./matchStore";

export function MatchCelebration() {
  const match = useAnnouncedMatch();
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const scale = useAnimatedValue(0.7);
  const opacity = useAnimatedValue(0);

  useEffect(() => {
    if (!match) {
      return;
    }
    scale.setValue(0.7);
    opacity.setValue(0);
    Animated.parallel([
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }),
      Animated.timing(opacity, { toValue: 1, duration: 220, useNativeDriver: true })
    ]).start();
  }, [match, scale, opacity]);

  if (!match) {
    return null;
  }

  const mine = queryClient.getQueryData<ProfileState>(queryKeys.profile)?.profile?.photos[0]?.thumbUrl ?? null;
  const theirThumb = match.profile.photos[0]?.thumbUrl ?? null;

  const sendMessage = () => {
    dismissMatch();
    if (navigationRef.isReady()) {
      navigationRef.navigate("Chat", {
        conversationId: match.conversationId,
        userId: match.profile.userId,
        firstName: match.profile.firstName,
        thumbUrl: theirThumb,
        matchId: match.matchId
      });
    }
  };

  return (
    <Modal animationType="fade" onRequestClose={dismissMatch} statusBarTranslucent transparent visible>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]} testID="match-modal">
        <Animated.View style={[styles.card, { backgroundColor: colors.surface, opacity, transform: [{ scale }] }]}>
          <Text color="primary" style={styles.centered} variant="title">
            C&apos;est un match !
          </Text>
          <View style={styles.avatars}>
            <Avatar size={96} thumbUrl={mine} />
            <Avatar name={match.profile.firstName} size={96} thumbUrl={theirThumb} />
          </View>
          <Text color="textMuted" style={styles.centered}>
            {`Vous et ${match.profile.firstName} vous plaisez mutuellement.`}
          </Text>
          <View style={styles.buttons}>
            <Button label="Envoyer un message" onPress={sendMessage} testID="match-send-message" />
            <Button label="Continuer à explorer" onPress={dismissMatch} testID="match-dismiss" variant="ghost" />
          </View>
        </Animated.View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: {
    width: "100%",
    maxWidth: 380,
    borderRadius: radius.lg,
    padding: spacing.xl,
    gap: spacing.lg,
    alignItems: "stretch"
  },
  centered: { textAlign: "center" },
  avatars: { flexDirection: "row", justifyContent: "center", gap: spacing.lg },
  buttons: { gap: spacing.sm }
});
