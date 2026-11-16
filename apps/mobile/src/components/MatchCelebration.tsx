import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import type { Match } from "../api/types";
import { spacing } from "../theme/tokens";
import { Avatar, Button, Text } from "../ui";

type Props = {
  match: Match | null;
  myPhotoUrl?: string | null;
  myName: string;
  onMessage: (match: Match) => void;
  onClose: () => void;
};

export function MatchCelebration({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  const scale = useRef(new Animated.Value(0.8)).current;
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!match) {
      return;
    }
    scale.setValue(0.8);
    opacity.setValue(0);
    Animated.parallel([
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }),
      Animated.timing(opacity, { toValue: 1, duration: 220, useNativeDriver: true })
    ]).start();
  }, [match, scale, opacity]);

  return (
    <Modal visible={!!match} transparent animationType="fade" onRequestClose={onClose}>
      <View style={styles.backdrop}>
        {match ? (
          <Animated.View style={[styles.content, { opacity, transform: [{ scale }] }]} testID="match-celebration">
            <Text variant="label" tone="accent" align="center">
              C&apos;EST RÉCIPROQUE
            </Text>
            <Text variant="display" align="center">
              Toi et {match.user.firstName}
            </Text>
            <Text tone="muted" align="center">
              Vous vous êtes plu mutuellement. Lance la conversation !
            </Text>
            <View style={styles.avatars}>
              <Avatar uri={myPhotoUrl} name={myName} size={112} ring />
              <View style={styles.overlap}>
                <Avatar uri={match.user.photos[0]?.url} name={match.user.firstName} size={112} ring />
              </View>
            </View>
            <Button label="Envoyer un message" icon="chatbubble-ellipses" onPress={() => onMessage(match)} testID="match-send-message" />
            <Button label="Continuer à découvrir" variant="ghost" onPress={onClose} />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: "rgba(10, 7, 6, 0.94)", justifyContent: "center", padding: spacing.xxl },
  content: { gap: spacing.lg, width: "100%", maxWidth: 420, alignSelf: "center" },
  avatars: { flexDirection: "row", justifyContent: "center", marginVertical: spacing.lg },
  overlap: { marginLeft: -24 }
});
