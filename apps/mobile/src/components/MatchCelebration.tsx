import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import type { Match } from "../api/types";
import { spacing, useTheme } from "../theme";
import { Avatar, Button, Icon, Text } from "../ui";

type Props = {
  match: Match | null;
  myPhotoUrl?: string | null;
  myName: string;
  onMessage: (match: Match) => void;
  onClose: () => void;
};

export function MatchCelebration({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  const { colors } = useTheme();
  const scale = useRef(new Animated.Value(0.85)).current;

  useEffect(() => {
    if (match) {
      scale.setValue(0.85);
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    }
  }, [match, scale]);

  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={!!match}>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]} testID="match-celebration">
        {match ? (
          <Animated.View style={[styles.content, { transform: [{ scale }] }]}>
            <Icon color="gold" name="sparkles" size={36} />
            <Text align="center" style={styles.title} variant="display">
              C’est réciproque !
            </Text>
            <Text align="center" style={styles.subtitle}>
              Vous et {match.user.firstName} vous plaisez. À vous de briser la glace.
            </Text>
            <View style={styles.avatars}>
              <Avatar name={myName} ring size={104} uri={myPhotoUrl} />
              <View style={[styles.heart, { backgroundColor: colors.accent }]}>
                <Icon color="textOnAccent" name="heart" size={22} />
              </View>
              <Avatar name={match.user.firstName} ring size={104} uri={match.user.photoUrl} />
            </View>
            <View style={styles.actions}>
              <Button
                icon="chatbubble-ellipses-outline"
                label="Envoyer un message"
                onPress={() => onMessage(match)}
                testID="match-message"
              />
              <Button label="Continuer à découvrir" onPress={onClose} style={styles.ghost} testID="match-continue" variant="ghost" />
            </View>
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  content: { width: "100%", maxWidth: 420, alignItems: "center", gap: spacing.md },
  title: { color: "#FFFFFF" },
  subtitle: { color: "rgba(255,255,255,0.85)", maxWidth: 320 },
  avatars: { flexDirection: "row", alignItems: "center", marginVertical: spacing.lg },
  heart: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center", marginHorizontal: -14, zIndex: 2 },
  actions: { alignSelf: "stretch", gap: spacing.xs },
  ghost: { borderColor: "transparent" }
});
