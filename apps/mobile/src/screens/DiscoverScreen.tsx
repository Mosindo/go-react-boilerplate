import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, StyleSheet, Text, View, useWindowDimensions } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { ApiError, errorMessage } from "../api/client";
import type { Card, SwipeAction } from "../api/types";
import { SwipeCard } from "../features/discover/SwipeCard";
import { useMatch } from "../features/match/MatchContext";
import type { MainStackParams } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";
import { Badge, EmptyState, ErrorState, Loading } from "../ui/components";

const REFILL_BELOW = 4;

export function DiscoverScreen() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const queryClient = useQueryClient();
  const { showMatch } = useMatch();
  const { width, height } = useWindowDimensions();
  const [deck, setDeck] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [exhausted, setExhausted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);

  const notifications = useQuery({ queryKey: ["notifications"], queryFn: api.notifications });

  const refill = useCallback(async () => {
    if (fetching.current) {
      return;
    }
    fetching.current = true;
    try {
      const batch = await api.discover(10);
      const fresh = batch.filter((card) => !seen.current.has(card.id));
      fresh.forEach((card) => seen.current.add(card.id));
      setDeck((current) => [...current, ...fresh]);
      setExhausted(fresh.length === 0);
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      fetching.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refill();
  }, [refill]);

  useEffect(() => {
    if (!loading && !exhausted && !error && deck.length < REFILL_BELOW) {
      void refill();
    }
  }, [deck.length, loading, exhausted, error, refill]);

  const decide = useCallback(
    async (card: Card, action: SwipeAction) => {
      setDeck((current) => current.filter((c) => c.id !== card.id));
      setNotice(null);
      try {
        const result = await api.swipe(card.id, action);
        if (result.matched && result.conversationId && result.matchId) {
          showMatch({ user: card, conversationId: result.conversationId, matchId: result.matchId });
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
        }
      } catch (e) {
        if (!(e instanceof ApiError && (e.code === "already_swiped" || e.status === 404))) {
          setNotice(errorMessage(e));
        }
      }
    },
    [queryClient, showMatch]
  );

  const cardWidth = Math.min(width - spacing.xl * 2, 420);
  const cardHeight = Math.min(Math.max(height - 360, 360), 600);
  const top = deck[0];
  const unread = notifications.data?.unreadCount ?? 0;

  return (
    <View style={[styles.flex, { backgroundColor: theme.background }]}>
      <View style={styles.header}>
        <Text style={[styles.brand, { color: theme.primary }]}>Braise</Text>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Notifications, ${unread} non lues`}
          onPress={() => navigation.navigate("Notifications")}
          style={styles.bell}
          testID="open-notifications"
        >
          <Text style={{ fontSize: 22 }}>🔔</Text>
          <View style={styles.bellBadge}>
            <Badge count={unread} />
          </View>
        </Pressable>
      </View>

      {loading ? (
        <Loading label="Recherche de profils…" />
      ) : error && deck.length === 0 ? (
        <ErrorState
          message={error}
          onRetry={() => {
            setLoading(true);
            void refill();
          }}
        />
      ) : !top ? (
        <EmptyState
          title="Vous avez tout vu pour l'instant"
          message="Élargissez vos préférences ou revenez plus tard : de nouveaux profils arrivent régulièrement."
          actionLabel="Actualiser"
          onAction={() => {
            setExhausted(false);
            setLoading(true);
            void refill();
          }}
        />
      ) : (
        <View style={styles.flex}>
          <View style={[styles.deck, { height: cardHeight + spacing.md }]}>
            {deck
              .slice(0, 2)
              .reverse()
              .map((card) => (
                <SwipeCard
                  key={card.id}
                  card={card}
                  interactive={card.id === top.id}
                  width={cardWidth}
                  height={cardHeight}
                  onSwipe={(action) => void decide(card, action)}
                  onOpen={() => navigation.navigate("UserProfile", { user: card })}
                />
              ))}
          </View>
          {notice ? <Text style={[styles.notice, { color: theme.danger }]}>{notice}</Text> : null}
          <View style={styles.actions}>
            <RoundButton label="Passer" symbol="✕" color={theme.danger} onPress={() => void decide(top, "pass")} testID="pass-button" />
            <RoundButton
              label="J'aime"
              symbol="♥"
              color={theme.primary}
              onPress={() => void decide(top, "like")}
              testID="like-button"
              large
            />
          </View>
        </View>
      )}
    </View>
  );
}

function RoundButton({
  label,
  symbol,
  color,
  onPress,
  large,
  testID
}: {
  label: string;
  symbol: string;
  color: string;
  onPress: () => void;
  large?: boolean;
  testID: string;
}) {
  const theme = useTheme();
  const size = large ? 76 : 64;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          borderColor: color,
          backgroundColor: theme.surface,
          opacity: pressed ? 0.8 : 1
        }
      ]}
    >
      <Text style={{ color, fontSize: large ? 34 : 28, fontWeight: "800" }}>{symbol}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.xl,
    paddingTop: spacing.lg,
    paddingBottom: spacing.sm
  },
  brand: { fontSize: 28, fontWeight: "900", letterSpacing: -0.5 },
  bell: { padding: spacing.sm },
  bellBadge: { position: "absolute", top: 0, right: 0 },
  deck: { alignItems: "center", marginTop: spacing.sm },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xxl, marginTop: spacing.lg },
  round: { borderWidth: 2.5, alignItems: "center", justifyContent: "center", borderRadius: radius.pill },
  notice: { textAlign: "center", marginTop: spacing.sm }
});
