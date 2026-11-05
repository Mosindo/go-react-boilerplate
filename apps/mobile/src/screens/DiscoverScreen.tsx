import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { discoveryApi } from "../api/endpoints";
import type { Candidate, SwipeAction } from "../api/types";
import { BackCard, TopCard, type DeckCardHandle } from "../features/SwipeDeck";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { useRealtime } from "../realtime/RealtimeProvider";
import { useTheme } from "../theme";
import { EmptyState, ErrorState, LoadingState, Screen, Text, useToast } from "../ui";

const REFILL_AT = 3;

export function DiscoverScreen() {
  const t = useTheme();
  const toast = useToast();
  const qc = useQueryClient();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const { showMatch } = useRealtime();
  const topRef = useRef<DeckCardHandle>(null);
  const [deck, setDeck] = useState<Candidate[]>([]);
  const seen = useRef(new Set<string>());
  // True once a refill returned nothing new: stop auto-refetching until the user asks.
  const exhausted = useRef(false);

  const feed = useQuery({
    queryKey: queryKeys.feed,
    queryFn: () => discoveryApi.feed(10),
    staleTime: 0,
    gcTime: 0
  });

  // Merge fetched pages into the local deck without ever re-adding a handled card.
  useEffect(() => {
    if (!feed.data) return;
    setDeck((cur) => {
      const have = new Set(cur.map((c) => c.id));
      const fresh = feed.data.filter((c) => !have.has(c.id) && !seen.current.has(c.id));
      exhausted.current = fresh.length === 0;
      return fresh.length ? [...cur, ...fresh] : cur;
    });
  }, [feed.data, feed.dataUpdatedAt]);

  useEffect(() => {
    if (!exhausted.current && deck.length <= REFILL_AT && !feed.isFetching && feed.data) {
      void feed.refetch();
    }
  }, [deck.length, feed]);

  const refresh = useCallback(() => {
    exhausted.current = false;
    void feed.refetch();
  }, [feed]);

  useEffect(
    () =>
      navigation.addListener("focus", () => {
        if (exhausted.current) refresh();
      }),
    [navigation, refresh]
  );

  const onSwiped = useCallback(
    async (card: Candidate, action: SwipeAction) => {
      seen.current.add(card.id);
      setDeck((cur) => cur.filter((c) => c.id !== card.id));
      try {
        const res = await discoveryApi.swipe(card.id, action);
        if (res.matched && !res.alreadySwiped && res.conversationId) {
          showMatch({
            matchId: res.matchId ?? "",
            conversationId: res.conversationId,
            peerId: card.id,
            peerName: card.firstName
          });
          void qc.invalidateQueries({ queryKey: queryKeys.conversations });
        }
      } catch (err) {
        // Put the card back so nothing is silently lost.
        seen.current.delete(card.id);
        setDeck((cur) => [card, ...cur]);
        toast.show(errorMessage(err), "error");
      }
    },
    [qc, showMatch, toast]
  );

  const top = deck[0];
  const next = deck[1];

  if (feed.isLoading && deck.length === 0) {
    return (
      <Screen edges={["top"]}>
        <LoadingState label="Recherche de profils…" />
      </Screen>
    );
  }
  if (feed.isError && deck.length === 0) {
    return (
      <Screen edges={["top"]}>
        <ErrorState message={errorMessage(feed.error)} onRetry={() => void feed.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={styles.header}>
        <Text variant="title">Découvrir</Text>
      </View>
      {top ? (
        <>
          <View style={styles.deck}>
            {next ? <BackCard key={`b-${next.id}`} card={next} /> : null}
            <TopCard
              key={top.id}
              ref={topRef}
              card={top}
              onSwiped={(a) => void onSwiped(top, a)}
              onOpen={() => navigation.navigate("UserProfile", { userId: top.id, canSwipe: true })}
            />
          </View>
          <View style={styles.actions}>
            <RoundButton
              icon="close"
              label="Passer"
              color={t.textMuted}
              onPress={() => topRef.current?.swipe("pass")}
            />
            <RoundButton
              icon="information"
              label="Voir le profil"
              small
              color={t.text}
              onPress={() => navigation.navigate("UserProfile", { userId: top.id, canSwipe: true })}
            />
            <RoundButton
              icon="heart"
              label="J'aime"
              color={t.primary}
              onPress={() => topRef.current?.swipe("like")}
            />
          </View>
        </>
      ) : (
        <EmptyState
          icon="compass-outline"
          title="Vous avez tout vu pour l'instant"
          message="Élargissez vos préférences ou revenez plus tard : de nouveaux profils arrivent chaque jour."
          actionLabel="Actualiser"
          onAction={refresh}
        />
      )}
    </Screen>
  );
}

function RoundButton({
  icon,
  label,
  color,
  onPress,
  small
}: {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  color: string;
  onPress: () => void;
  small?: boolean;
}) {
  const t = useTheme();
  const size = small ? 52 : 68;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: t.surface,
          borderColor: t.border,
          opacity: pressed ? 0.8 : 1
        }
      ]}
    >
      <Ionicons name={icon} size={small ? 24 : 32} color={color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: 20, paddingVertical: 8 },
  deck: { flex: 1, marginHorizontal: 14, marginBottom: 14 },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: 24,
    paddingBottom: 14
  },
  round: { borderWidth: 1, alignItems: "center", justifyContent: "center" }
});
