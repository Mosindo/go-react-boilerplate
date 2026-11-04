import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import type { SwipeResponse } from "../../api/types";
import { Button, RemoteImage, Text, radius, spacing, useColors } from "../../shared/ui";

type Props = { match: NonNullable<SwipeResponse["match"]> | null; onChat: () => void; onClose: () => void };

export function MatchModal({ match, onChat, onClose }: Props) {
  const c = useColors();
  const scale = useRef(new Animated.Value(0.8)).current;
  useEffect(() => {
    if (match) Animated.spring(scale, { toValue: 1, useNativeDriver: true, friction: 5 }).start();
    else scale.setValue(0.8);
  }, [match, scale]);

  const profile = match?.profile;
  return (
    <Modal visible={match !== null} transparent animationType="fade" onRequestClose={onClose} statusBarTranslucent>
      <View style={[styles.backdrop, { backgroundColor: c.overlay }]}>
        <Animated.View style={[styles.card, { backgroundColor: c.surface, transform: [{ scale }] }]} testID="match-modal">
          <Text variant="label" tone="accent" style={{ letterSpacing: 2, textTransform: "uppercase" }}>Il y a de la place pour une rencontre</Text>
          <Text variant="title" style={{ textAlign: "center" }}>C'est un match !</Text>
          {profile ? (
            <>
              <RemoteImage path={profile.photos[0]?.url} accessibilityLabel={`Photo de ${profile.firstName}`} style={styles.photo} />
              <Text>Vous et {profile.firstName} vous plaisez mutuellement.</Text>
            </>
          ) : null}
          <Button title="Envoyer un message" variant="accent" onPress={onChat} testID="match-chat" style={{ alignSelf: "stretch" }} />
          <Button title="Continuer à découvrir" variant="ghost" onPress={onClose} testID="match-close" />
        </Animated.View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: { width: "100%", maxWidth: 380, borderRadius: radius.lg, padding: spacing.xl, alignItems: "center", gap: spacing.md },
  photo: { width: 140, height: 140, borderRadius: 70 },
});
