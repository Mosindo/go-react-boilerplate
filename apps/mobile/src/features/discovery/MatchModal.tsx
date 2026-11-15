import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import * as Haptics from "expo-haptics";
import type { MatchSummary, Photo } from "../../api/types";
import { spacing, useTheme } from "../../theme";
import { Avatar, Button, Text } from "../../ui";

type Props = {
  match: MatchSummary | null;
  myPhoto?: Photo;
  myName: string;
  onMessage: (match: MatchSummary) => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhoto, myName, onMessage, onClose }: Props) {
  const { colors } = useTheme();
  const scale = useRef(new Animated.Value(0.6)).current;

  useEffect(() => {
    if (!match) return;
    scale.setValue(0.6);
    Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    void Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success).catch(() => undefined);
  }, [match, scale]);

  return (
    <Modal visible={Boolean(match)} transparent animationType="fade" onRequestClose={onClose}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]} testID="match-modal">
        {match ? (
          <Animated.View style={[styles.content, { transform: [{ scale }] }]}>
            <Text variant="micro" color="primary" align="center">
              C’EST RÉCIPROQUE
            </Text>
            <Text variant="display" align="center" style={styles.title}>
              C’est un match !
            </Text>
            <View style={styles.avatars}>
              <View style={[styles.left, { borderColor: colors.background }]}>
                <Avatar photo={myPhoto} name={myName} size={112} />
              </View>
              <View style={[styles.right, { borderColor: colors.background }]}>
                <Avatar photo={match.user.photos[0]} name={match.user.firstName} size={112} />
              </View>
            </View>
            <Text color="textMuted" align="center" style={styles.body}>
              Vous et {match.user.firstName} vous plaisez mutuellement. Lancez la conversation !
            </Text>
            <Button
              label={`Écrire à ${match.user.firstName}`}
              icon="chatbubble-ellipses-outline"
              onPress={() => onMessage(match)}
              testID="match-message"
            />
            <Button label="Continuer à découvrir" variant="ghost" onPress={onClose} testID="match-close" />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  content: { width: "100%", maxWidth: 380, gap: spacing.sm },
  title: { marginBottom: spacing.lg },
  avatars: { flexDirection: "row", justifyContent: "center", marginBottom: spacing.lg },
  left: { borderWidth: 4, borderRadius: 64, transform: [{ rotate: "-6deg" }, { translateX: 12 }] },
  right: { borderWidth: 4, borderRadius: 64, transform: [{ rotate: "6deg" }, { translateX: -12 }] },
  body: { marginBottom: spacing.md }
});
