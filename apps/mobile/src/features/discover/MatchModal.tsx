import React, { useEffect, useRef } from "react";
import { Animated, Modal, Platform, StyleSheet, View } from "react-native";
import type { MatchSummary } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/palette";
import { Avatar, Button, Text } from "../../ui";

type Props = {
  match: MatchSummary | null;
  myPhotoUrl?: string;
  myName: string;
  onMessage: (match: MatchSummary) => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  const { colors } = useTheme();
  const scale = useRef(new Animated.Value(0.6)).current;

  useEffect(() => {
    if (match) {
      scale.setValue(0.6);
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    }
  }, [match, scale]);

  return (
    <Modal visible={!!match} transparent animationType={Platform.OS === "web" ? "none" : "fade"} onRequestClose={onClose} statusBarTranslucent>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]} testID="match-modal">
        {match ? (
          <Animated.View style={[styles.content, { transform: [{ scale }] }]}>
            <Text variant="overline" tone="inverse" style={{ color: colors.accent }}>
              C’EST RÉCIPROQUE
            </Text>
            <Text variant="display" tone="inverse" align="center" accessibilityRole="header">
              Nouveau match !
            </Text>
            <Text tone="inverse" align="center" style={styles.sub}>
              Toi et {match.user.firstName} vous plaisez mutuellement.
            </Text>
            <View style={styles.avatars}>
              <Avatar url={myPhotoUrl} name={myName} size={112} ring style={styles.left} />
              <Avatar url={match.user.photos[0]?.url} name={match.user.firstName} size={112} ring style={styles.right} />
            </View>
            <Button label={`Écrire à ${match.user.firstName}`} size="lg" icon="chatbubble-ellipses-outline" fullWidth onPress={() => onMessage(match)} testID="match-message" />
            <Button label="Continuer à découvrir" variant="inverse" fullWidth onPress={onClose} testID="match-close" />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  content: { width: "100%", maxWidth: 420, alignItems: "center", gap: spacing.sm },
  sub: { opacity: 0.9 },
  avatars: { flexDirection: "row", marginVertical: spacing.xl },
  left: { transform: [{ rotate: "-6deg" }], marginRight: -18 },
  right: { transform: [{ rotate: "6deg" }] }
});
