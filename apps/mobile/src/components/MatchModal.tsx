import React, { useEffect, useRef } from "react";
import { Animated, Image, Modal, StyleSheet, View } from "react-native";
import { absoluteUrl } from "../api/client";
import { useRealtime } from "../hooks/useRealtime";
import { Button, Text, colors, radii, spacing } from "../shared/ui";

type MatchModalProps = {
  onOpenChat: (matchId: string, userId: string) => void;
};

/** Full-screen celebration shown when two people like each other. */
export function MatchModal({ onOpenChat }: MatchModalProps) {
  const { newMatch, dismissMatch } = useRealtime();
  const scale = useRef(new Animated.Value(0.85)).current;
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!newMatch) {
      return;
    }
    scale.setValue(0.85);
    opacity.setValue(0);
    Animated.parallel([
      Animated.spring(scale, { toValue: 1, useNativeDriver: true, friction: 6 }),
      Animated.timing(opacity, { toValue: 1, duration: 220, useNativeDriver: true })
    ]).start();
  }, [newMatch, opacity, scale]);

  if (!newMatch) {
    return null;
  }
  const photo = newMatch.profile.photos[0];

  return (
    <Modal animationType="fade" onRequestClose={dismissMatch} transparent visible>
      <View style={styles.backdrop}>
        <Animated.View accessibilityViewIsModal style={[styles.card, { opacity, transform: [{ scale }] }]}>
          <Text tone="primary" variant="eyebrow" weight="bold">
            Nouveau match
          </Text>
          <Text style={styles.title} variant="title" weight="bold">
            C'est un match !
          </Text>
          {photo ? <Image source={{ uri: absoluteUrl(photo.url) }} style={styles.photo} /> : null}
          <Text style={styles.copy} tone="muted">
            Vous et {newMatch.profile.firstName} vous plaisez mutuellement. Qui fera le premier pas ?
          </Text>
          <Button
            fullWidth
            label={`Écrire à ${newMatch.profile.firstName}`}
            onPress={() => {
              const target = newMatch;
              dismissMatch();
              onOpenChat(target.matchId, target.profile.id);
            }}
            testID="match-open-chat"
          />
          <Button fullWidth label="Continuer à découvrir" onPress={dismissMatch} testID="match-dismiss" variant="ghost" />
        </Animated.View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: {
    width: "100%",
    maxWidth: 380,
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.xl,
    padding: spacing.xl,
    alignItems: "center",
    gap: spacing.md
  },
  title: { textAlign: "center" },
  photo: { width: 160, height: 200, borderRadius: radii.lg, backgroundColor: colors.surfaceSubtle },
  copy: { textAlign: "center" }
});
