import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import type { Match } from "../../api/types";
import { spacing, useTheme } from "../../theme";
import { Avatar, Button, Text } from "../../ui";

export function MatchModal({ match, myPhoto, onMessage, onClose }: { match: Match | null; myPhoto?: string; onMessage: () => void; onClose: () => void }) {
  const { colors } = useTheme();
  const scale = useRef(new Animated.Value(0.85)).current;
  // Keep rendering the last match while the modal fades out.
  const lastMatch = useRef<Match | null>(null);
  if (match) lastMatch.current = match;
  const shown = match ?? lastMatch.current;
  useEffect(() => {
    if (match) {
      scale.setValue(0.85);
      Animated.spring(scale, { toValue: 1, useNativeDriver: true, friction: 6 }).start();
    }
  }, [match, scale]);

  return (
    <Modal visible={!!match} transparent animationType="fade" onRequestClose={onClose}>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]}>
        <Animated.View style={[styles.sheet, { backgroundColor: colors.surface, transform: [{ scale }] }]} accessibilityViewIsModal>
          <View style={styles.avatars}>
            <Avatar url={myPhoto} size={92} label="Vous" />
            <View style={[styles.heart, { backgroundColor: colors.primary }]}>
              <Text tone="onPrimary" variant="heading">
                ♥
              </Text>
            </View>
            <Avatar url={shown?.user.photos[0]?.url} size={92} label={shown?.user.firstName} />
          </View>
          <Text variant="title" serif center accessibilityRole="header">
            C’est réciproque !
          </Text>
          <Text tone="muted" center>
            {shown?.user.firstName} et vous vous plaisez. Lancez la conversation tant que l’étincelle est là.
          </Text>
          <Button label="Envoyer un message" icon="chatbubble-ellipses" onPress={onMessage} testID="match-message" style={styles.full} />
          <Button label="Continuer à découvrir" variant="ghost" onPress={onClose} testID="match-close" />
        </Animated.View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  sheet: { width: "100%", maxWidth: 400, borderRadius: 28, padding: spacing.xl, gap: spacing.md, alignItems: "center" },
  avatars: { flexDirection: "row", alignItems: "center", marginBottom: spacing.sm },
  heart: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center", marginHorizontal: -12, zIndex: 2 },
  full: { alignSelf: "stretch" }
});
