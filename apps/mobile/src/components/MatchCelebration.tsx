import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, Modal, Pressable, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { Match } from "../api/types";
import { useAuth } from "../hooks/useAuth";
import { onIncomingMatch } from "../hooks/useRealtime";
import { openChat } from "../navigation/navigationRef";
import { spacing, useTheme } from "../theme";
import { Avatar, Button, Text } from "../ui";

const CelebrationContext = createContext<(match: Match) => void>(() => undefined);

export function useCelebrateMatch() {
  return useContext(CelebrationContext);
}

/** Full-screen "it's mutual" moment, shown after our own like or when the other person likes back. */
export function MatchCelebrationProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const { profile } = useAuth();
  const [match, setMatch] = useState<Match | null>(null);
  const scale = useRef(new Animated.Value(0.8)).current;

  const celebrate = useCallback((m: Match) => setMatch(m), []);
  useEffect(() => onIncomingMatch(celebrate), [celebrate]);

  useEffect(() => {
    if (match) {
      scale.setValue(0.8);
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    }
  }, [match, scale]);

  const myPhoto = profile?.photos[0]?.url ?? null;
  const value = useMemo(() => celebrate, [celebrate]);

  return (
    <CelebrationContext.Provider value={value}>
      {children}
      <Modal animationType="fade" onRequestClose={() => setMatch(null)} transparent visible={!!match}>
        {match ? (
          <LinearGradient colors={[colors.primary, colors.accent]} end={{ x: 1, y: 1 }} start={{ x: 0, y: 0 }} style={styles.root}>
            <Animated.View style={[styles.content, { transform: [{ scale }] }]} testID="match-modal">
              <Text center style={styles.kicker} tone="inverse" variant="overline">
                Nouveau match
              </Text>
              <Text center tone="inverse" variant="display">
                C’est réciproque !
              </Text>
              <Text center style={styles.sub} tone="inverse">
                Vous et {match.profile.firstName} vous plaisez mutuellement.
              </Text>
              <View style={styles.avatars}>
                <View style={[styles.avatarWrap, styles.left]}>
                  <Avatar name={profile?.firstName ?? "Moi"} size={120} url={myPhoto} />
                </View>
                <View style={[styles.avatarWrap, styles.right]}>
                  <Avatar name={match.profile.firstName} size={120} url={match.profile.photos[0]?.url} />
                </View>
              </View>
              <View style={styles.actions}>
                <Button
                  icon="chatbubble-ellipses"
                  label={`Écrire à ${match.profile.firstName}`}
                  onPress={() => {
                    const conversationId = match.conversationId;
                    setMatch(null);
                    openChat(conversationId);
                  }}
                  style={styles.primary}
                  testID="match-modal-message"
                  variant="secondary"
                />
                <Pressable accessibilityRole="button" onPress={() => setMatch(null)} style={styles.ghost} testID="match-modal-close">
                  <Text center tone="inverse" variant="bodyStrong">
                    Continuer à découvrir
                  </Text>
                </Pressable>
              </View>
            </Animated.View>
          </LinearGradient>
        ) : null}
      </Modal>
    </CelebrationContext.Provider>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  content: { width: "100%", maxWidth: 420, alignItems: "center", gap: spacing.sm },
  kicker: { opacity: 0.85 },
  sub: { opacity: 0.92 },
  avatars: { flexDirection: "row", marginVertical: spacing.xxl },
  avatarWrap: { borderRadius: 66, borderWidth: 4, borderColor: "#FFFFFF" },
  left: { transform: [{ rotate: "-8deg" }], marginRight: -18 },
  right: { transform: [{ rotate: "8deg" }] },
  actions: { width: "100%", gap: spacing.xs },
  primary: { backgroundColor: "#FFFFFF", borderColor: "#FFFFFF" },
  ghost: { paddingVertical: spacing.md, alignItems: "center" }
});
