import React, { useEffect, useRef } from "react";
import { Animated, Modal, Platform, StyleSheet, View } from "react-native";
import type { MatchInfo } from "../api/types";
import { Button, Text, colors, radii, spacing } from "../shared/ui";
import { PhotoImage } from "./PhotoImage";

type MatchOverlayProps = {
  match: MatchInfo | null;
  myPhotoPath?: string | null;
  myName?: string;
  onMessage: (match: MatchInfo) => void;
  onClose: () => void;
};

export function MatchOverlay({ match, myPhotoPath, myName, onMessage, onClose }: MatchOverlayProps) {
  const scale = useRef(new Animated.Value(0.85)).current;

  useEffect(() => {
    if (match) {
      scale.setValue(0.85);
      Animated.spring(scale, { toValue: 1, friction: 6, useNativeDriver: Platform.OS !== "web" }).start();
    }
  }, [match, scale]);

  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={Boolean(match)}>
      <View style={styles.backdrop} testID="match-overlay">
        {match ? (
          <Animated.View style={[styles.sheet, { transform: [{ scale }] }]}>
            <Text style={styles.title} tone="primary" weight="bold">
              C'est un match !
            </Text>
            <Text style={styles.subtitle} tone="muted">
              Vous et {match.user.firstName} vous plaisez mutuellement.
            </Text>
            <View style={styles.photos}>
              <PhotoImage fallbackLabel={myName} path={myPhotoPath} style={styles.photo} />
              <PhotoImage fallbackLabel={match.user.firstName} path={match.user.photos[0]?.thumbUrl} style={[styles.photo, styles.photoOverlap]} />
            </View>
            <Button label={`Écrire à ${match.user.firstName}`} onPress={() => onMessage(match)} size="lg" testID="match-message" />
            <Button label="Continuer à découvrir" onPress={onClose} testID="match-close" variant="ghost" />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.scrim, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  sheet: {
    width: "100%",
    maxWidth: 420,
    backgroundColor: colors.background,
    borderRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md,
    alignItems: "stretch"
  },
  title: { fontSize: 34, lineHeight: 40, textAlign: "center" },
  subtitle: { textAlign: "center" },
  photos: { flexDirection: "row", justifyContent: "center", marginVertical: spacing.md },
  photo: { width: 112, height: 112, borderRadius: 56, borderWidth: 4, borderColor: colors.background },
  photoOverlap: { marginLeft: -24 }
});
