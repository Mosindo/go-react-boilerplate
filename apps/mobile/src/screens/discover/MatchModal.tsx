import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { SwipeResult } from "../../api/types";
import { colors, spacing } from "../../theme";
import { Avatar, Button, Text } from "../../ui";

type MatchModalProps = {
  match: NonNullable<SwipeResult["match"]> | null;
  myPhoto?: string | null;
  myName: string;
  onMessage: () => void;
  onClose: () => void;
};

export function MatchModal({ match: current, myPhoto, myName, onMessage, onClose }: MatchModalProps) {
  const scale = useRef(new Animated.Value(0.8)).current;
  // Keep rendering the last match while the modal fades out.
  const lastMatch = useRef(current);
  if (current) {
    lastMatch.current = current;
  }
  const match = current ?? lastMatch.current;
  useEffect(() => {
    if (current) {
      scale.setValue(0.8);
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    }
  }, [current, scale]);

  return (
    <Modal visible={!!current} transparent animationType="fade" onRequestClose={onClose}>
      <View style={styles.backdrop} testID="match-modal">
        {match ? (
          <Animated.View style={[styles.content, { transform: [{ scale }] }]}>
            <Ionicons name="sparkles" size={36} color={colors.gold} />
            <Text variant="display" align="center">
              C’est réciproque !
            </Text>
            <Text variant="body" color="textMuted" align="center">
              Vous et {match.profile.firstName} vous plaisez mutuellement.
            </Text>
            <View style={styles.avatars}>
              <Avatar path={myPhoto} name={myName} size={104} ring />
              <View style={styles.heart}>
                <Ionicons name="heart" size={22} color={colors.onPrimary} />
              </View>
              <Avatar path={match.profile.photos[0]?.url} name={match.profile.firstName} size={104} ring />
            </View>
            <Button
              label={`Écrire à ${match.profile.firstName}`}
              icon="chatbubble-ellipses-outline"
              onPress={onMessage}
              style={styles.button}
              testID="match-message"
            />
            <Button label="Continuer à découvrir" variant="ghost" onPress={onClose} style={styles.button} testID="match-continue" />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: "rgba(21,17,26,0.94)",
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
    marginVertical: spacing.lg
  },
  heart: {
    width: 44,
    height: 44,
    borderRadius: 22,
    marginHorizontal: -14,
    zIndex: 1,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center"
  },
  button: {
    alignSelf: "stretch"
  }
});
