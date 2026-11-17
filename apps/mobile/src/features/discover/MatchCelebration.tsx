import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Match } from "../../api/types";
import { colors, spacing } from "../../theme";
import { Avatar, Button, Text } from "../../ui";

type Props = {
  match: Match | null;
  myPhotoUrl?: string;
  myName: string;
  onMessage: (match: Match) => void;
  onClose: () => void;
};

export function MatchCelebration({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  const scale = useRef(new Animated.Value(0.6)).current;
  useEffect(() => {
    if (match) {
      scale.setValue(0.6);
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    }
  }, [match, scale]);

  return (
    <Modal visible={!!match} transparent animationType="fade" onRequestClose={onClose}>
      <View style={styles.backdrop} testID="match-modal">
        {match ? (
          <Animated.View style={[styles.content, { transform: [{ scale }] }]}>
            <Ionicons name="sparkles" size={34} color={colors.gold} />
            <Text variant="display" center>
              C’est réciproque !
            </Text>
            <Text tone="muted" center>
              {match.profile.firstName} et toi vous plaisez. Pourquoi ne pas lui écrire ?
            </Text>
            <View style={styles.avatars}>
              <Avatar url={myPhotoUrl} name={myName} size={104} ring />
              <View style={styles.heart}>
                <Ionicons name="heart" size={22} color={colors.accentText} />
              </View>
              <Avatar url={match.profile.photos[0]?.url} name={match.profile.firstName} size={104} ring />
            </View>
            <Button label="Envoyer un message" testID="match-send-message" onPress={() => onMessage(match)} style={styles.button} />
            <Button label="Continuer à découvrir" variant="ghost" testID="match-continue" onPress={onClose} style={styles.button} />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay,
    alignItems: "center",
    justifyContent: "center",
    padding: spacing.xl
  },
  content: {
    width: "100%",
    maxWidth: 420,
    alignItems: "center",
    gap: spacing.md
  },
  avatars: {
    flexDirection: "row",
    alignItems: "center",
    marginVertical: spacing.xl
  },
  heart: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: colors.accent,
    alignItems: "center",
    justifyContent: "center",
    marginHorizontal: -14,
    zIndex: 2
  },
  button: {
    alignSelf: "stretch"
  }
});
