import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import type { MatchSummary, Photo as PhotoType } from "../api/types";
import { Button, Photo, Text, colors, spacing } from "../shared/ui";

type Props = {
  match: MatchSummary | null;
  myPhoto?: PhotoType;
  onMessage: (match: MatchSummary) => void;
  onClose: () => void;
};

export function MatchCelebration({ match, myPhoto, onClose, onMessage }: Props) {
  const scale = useRef(new Animated.Value(0.8)).current;
  useEffect(() => {
    if (match) {
      scale.setValue(0.8);
      Animated.spring(scale, { toValue: 1, friction: 6, useNativeDriver: true }).start();
    }
  }, [match, scale]);

  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={match !== null}>
      <View style={styles.backdrop} testID="match-celebration">
        {match ? (
          <Animated.View style={[styles.content, { transform: [{ scale }] }]}>
            <Text align="center" tone="primary" variant="eyebrow">
              C&apos;est réciproque
            </Text>
            <Text align="center" variant="display">
              Match !
            </Text>
            <Text align="center" tone="muted">
              Toi et {match.user.firstName} vous êtes plu. Lance la conversation quand tu veux.
            </Text>
            <View style={styles.photos}>
              <Photo style={[styles.photo, styles.left]} url={myPhoto?.url} />
              <Photo placeholderLabel={match.user.firstName.slice(0, 1)} style={[styles.photo, styles.right]} url={match.user.photo?.url} />
            </View>
            <Button
              fullWidth
              label={`Écrire à ${match.user.firstName}`}
              onPress={() => onMessage(match)}
              size="lg"
              testID="match-message"
            />
            <Button fullWidth label="Continuer à découvrir" onPress={onClose} testID="match-close" variant="ghost" />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: "rgba(10, 8, 7, 0.94)",
    justifyContent: "center",
    padding: spacing.xl
  },
  content: {
    width: "100%",
    maxWidth: 420,
    alignSelf: "center",
    alignItems: "center",
    gap: spacing.md
  },
  photos: {
    flexDirection: "row",
    marginVertical: spacing.xl
  },
  photo: {
    width: 128,
    height: 160,
    borderRadius: 24,
    borderWidth: 3,
    borderColor: colors.background
  },
  left: {
    transform: [{ rotate: "-7deg" }, { translateX: 12 }]
  },
  right: {
    transform: [{ rotate: "7deg" }, { translateX: -12 }],
    borderColor: colors.primary
  }
});
