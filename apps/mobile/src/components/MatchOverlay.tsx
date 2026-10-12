import { Ionicons } from "@expo/vector-icons";
import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Animated, Modal, StyleSheet, View } from "react-native";
import type { MatchSummary } from "../api/types";
import { useMatches } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import { radii, spacing, useTheme } from "../theme";
import { Avatar, Button, Text } from "../ui";

type MatchApi = {
  /** Celebrate a match; ignores a match that was already celebrated this session. */
  celebrate: (matchId: string) => void;
};

const MatchContext = createContext<MatchApi>({ celebrate: () => undefined });

type Props = { children: React.ReactNode; onOpenChat: (match: MatchSummary) => void; selfPhoto?: string };

export function MatchProvider({ children, onOpenChat, selfPhoto }: Props) {
  const { colors } = useTheme();
  const matches = useMatches();
  const { subscribe } = useRealtime();
  const celebrated = useRef(new Set<string>());
  const [pendingId, setPendingId] = useState<string | null>(null);
  const scale = useRef(new Animated.Value(0.8)).current;

  const celebrate = useCallback((matchId: string) => {
    if (celebrated.current.has(matchId)) {
      return;
    }
    celebrated.current.add(matchId);
    setPendingId(matchId);
  }, []);

  useEffect(() => subscribe((event) => event.type === "match.new" && celebrate(event.data.matchId)), [subscribe, celebrate]);

  const match = pendingId ? matches.data?.find((m) => m.id === pendingId) : undefined;
  useEffect(() => {
    if (pendingId && !match && !matches.isFetching) {
      void matches.refetch();
    }
  }, [pendingId, match, matches]);

  useEffect(() => {
    if (match) {
      scale.setValue(0.8);
      Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
    }
  }, [match, scale]);

  const api = useMemo(() => ({ celebrate }), [celebrate]);
  const close = () => setPendingId(null);

  return (
    <MatchContext.Provider value={api}>
      {children}
      <Modal visible={!!match} transparent animationType="fade" onRequestClose={close}>
        <View style={[styles.backdrop, { backgroundColor: colors.scrim }]}>
          {match ? (
            <Animated.View style={[styles.sheet, { backgroundColor: colors.surface, transform: [{ scale }] }]} accessibilityViewIsModal>
              <Ionicons name="heart" size={40} color={colors.primary} />
              <Text variant="display" tone="primary" style={styles.center}>
                C'est un match !
              </Text>
              <Text tone="muted" style={styles.center}>
                Vous et {match.profile.firstName} vous plaisez mutuellement.
              </Text>
              <View style={styles.avatars}>
                <Avatar uri={selfPhoto} name="Moi" size={84} />
                <Avatar uri={match.profile.photos[0]?.thumbUrl} name={match.profile.firstName} size={84} />
              </View>
              <Button
                label={`Écrire à ${match.profile.firstName}`}
                onPress={() => {
                  close();
                  onOpenChat(match);
                }}
                style={styles.full}
              />
              <Button label="Continuer à découvrir" variant="ghost" onPress={close} />
            </Animated.View>
          ) : null}
        </View>
      </Modal>
    </MatchContext.Provider>
  );
}

export function useMatchCelebration(): MatchApi {
  return useContext(MatchContext);
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  sheet: { width: "100%", maxWidth: 380, borderRadius: radii.xl, padding: spacing.xl, alignItems: "center", gap: spacing.md },
  center: { textAlign: "center" },
  avatars: { flexDirection: "row", gap: spacing.lg, marginVertical: spacing.md },
  full: { alignSelf: "stretch" }
});
