import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import type { MatchSummary, Photo } from "../../api/types";
import { useReducedMotion } from "../../lib/motion";
import { radii, spacing, useColors } from "../../theme";
import { Button, Icon, ProfilePhoto, Text } from "../../ui";

type Props = {
  match: MatchSummary | null;
  myPhoto: Photo | null;
  myName: string;
  onMessage: (match: MatchSummary) => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhoto, myName, onMessage, onClose }: Props) {
  const colors = useColors();
  const reduced = useReducedMotion();
  const scale = useRef(new Animated.Value(reduced ? 1 : 0.8)).current;
  const opacity = useRef(new Animated.Value(reduced ? 1 : 0)).current;
  const matchId = match?.id;

  useEffect(() => {
    if (!matchId) return;
    if (reduced) {
      scale.setValue(1);
      opacity.setValue(1);
      return;
    }
    scale.setValue(0.8);
    opacity.setValue(0);
    Animated.parallel([
      Animated.spring(scale, { toValue: 1, useNativeDriver: true, bounciness: 10 }),
      Animated.timing(opacity, { toValue: 1, duration: 180, useNativeDriver: true })
    ]).start();
  }, [matchId, reduced, scale, opacity]);

  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={match !== null}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        {match ? (
          <Animated.View
            accessibilityViewIsModal
            style={[
              styles.card,
              { backgroundColor: colors.surface, borderColor: colors.border, opacity, transform: [{ scale }] }
            ]}
          >
            <Icon color={colors.primary} name="heart" size={40} />
            <Text accessibilityLiveRegion="assertive" center variant="heading">
              It&apos;s a match!
            </Text>
            <Text center tone="muted">
              You and {match.user.displayName} liked each other.
            </Text>
            <View style={styles.photos}>
              <ProfilePhoto name={myName} photo={myPhoto} round size="thumb" style={styles.photo} />
              <ProfilePhoto
                name={match.user.displayName}
                photo={match.user.photo}
                round
                size="thumb"
                style={styles.photo}
              />
            </View>
            <View style={styles.buttons}>
              <Button label="Send a message" onPress={() => onMessage(match)} />
              <Button label="Keep swiping" onPress={onClose} variant="ghost" />
            </View>
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  card: {
    width: "100%",
    maxWidth: 400,
    alignItems: "center",
    gap: spacing.sm,
    padding: spacing.lg,
    borderRadius: radii.xl,
    borderWidth: 1
  },
  photos: { flexDirection: "row", gap: spacing.sm, marginVertical: spacing.sm },
  photo: { width: 96, height: 96 },
  buttons: { alignSelf: "stretch", gap: spacing.xs }
});
