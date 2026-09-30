import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import { Avatar, Button, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import type { Match, Photo } from "../../lib/api/types";

type Props = {
  match: Match | null;
  myPhoto?: Photo | null;
  myName: string;
  onMessage: (match: Match) => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhoto, myName, onMessage, onClose }: Props) {
  const { colors } = useTheme();
  const scale = useRef(new Animated.Value(0.85)).current;

  useEffect(() => {
    if (match) {
      scale.setValue(0.85);
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    }
  }, [match, scale]);

  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={match !== null}>
      <View style={[styles.backdrop, { backgroundColor: colors.scrim }]}>
        {match ? (
          <Animated.View style={[styles.content, { transform: [{ scale }] }]} testID="match-modal">
            <Text align="center" style={styles.white} variant="overline">
              INTÉRÊT RÉCIPROQUE
            </Text>
            <Text align="center" style={styles.white} variant="display">
              C’est un match !
            </Text>
            <Text align="center" style={styles.whiteMuted}>
              Vous et {match.user.firstName} vous plaisez mutuellement.
            </Text>
            <View style={styles.avatars}>
              <Avatar name={myName} ring size={104} uri={myPhoto?.url} />
              <View style={styles.overlap}>
                <Avatar name={match.user.firstName} ring size={104} uri={match.user.photo?.url} />
              </View>
            </View>
            <Button fullWidth icon="chatbubble-ellipses" label="Envoyer un message" onPress={() => onMessage(match)} testID="match-message" />
            <Button fullWidth label="Continuer à découvrir" onPress={onClose} style={styles.secondary} variant="ghost" />
          </Animated.View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24 },
  content: { width: "100%", maxWidth: 420, gap: 14, alignItems: "center" },
  white: { color: "#fff" },
  whiteMuted: { color: "rgba(255,255,255,0.85)" },
  avatars: { flexDirection: "row", marginVertical: 24 },
  overlap: { marginLeft: -20 },
  secondary: { borderColor: "transparent" }
});
