import React, { useEffect, useRef } from "react";
import { Animated, Platform, Modal, StyleSheet, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useMyProfile } from "../../hooks/queries";
import { navigationRef } from "../../navigation/navigationRef";
import { colors, radii, spacing } from "../../theme/tokens";
import { AppText } from "../AppText";
import { Avatar } from "../Avatar";
import { Button } from "../Button";
import { dismissMatch, useCelebratedMatch } from "./matchCelebration";

export function MatchCelebration() {
  const match = useCelebratedMatch();
  const me = useMyProfile(!!match);
  const scale = useRef(new Animated.Value(0.85)).current;

  useEffect(() => {
    if (match) {
      scale.setValue(0.85);
      Animated.spring(scale, { toValue: 1, friction: 6, useNativeDriver: Platform.OS !== "web" }).start();
    }
  }, [match, scale]);

  if (!match) return null;
  const openChat = () => {
    dismissMatch();
    if (navigationRef.isReady()) {
      navigationRef.navigate("Chat", {
        conversationId: match.conversationId,
        matchId: match.id,
        userId: match.user.userId,
        name: match.user.firstName,
        photoUrl: match.user.photos[0]?.url ?? null
      });
    }
  };

  return (
    <Modal transparent visible animationType="fade" onRequestClose={dismissMatch}>
      <View style={styles.backdrop}>
        <Animated.View style={[styles.card, { transform: [{ scale }] }]} testID="match-celebration">
          <Ionicons name="sparkles" size={30} color={colors.accent} />
          <AppText variant="display" align="center">
            C’est un match !
          </AppText>
          <AppText color={colors.textMuted} align="center">
            {match.user.firstName} et toi vous êtes plu mutuellement.
          </AppText>
          <View style={styles.avatars}>
            <Avatar url={me.data?.photos[0]?.url} name={me.data?.profile?.firstName ?? "Moi"} size={96} ring />
            <View style={styles.heart}>
              <Ionicons name="heart" size={22} color={colors.onPrimary} />
            </View>
            <Avatar url={match.user.photos[0]?.url} name={match.user.firstName} size={96} ring />
          </View>
          <Button label={`Écrire à ${match.user.firstName}`} icon="chatbubble-ellipses-outline" onPress={openChat} style={styles.full} testID="match-open-chat" />
          <Button label="Continuer à découvrir" variant="ghost" onPress={dismissMatch} style={styles.full} testID="match-dismiss" />
        </Animated.View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  card: {
    width: "100%",
    maxWidth: 420,
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.xl,
    borderRadius: radii.xl,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  avatars: { flexDirection: "row", alignItems: "center", marginVertical: spacing.md },
  heart: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center",
    marginHorizontal: -12,
    zIndex: 2
  },
  full: { alignSelf: "stretch" }
});
