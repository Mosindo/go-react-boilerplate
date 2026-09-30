import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useNavigation } from "@react-navigation/native";
import { EmptyState, ErrorState, IconButton, LoadingState, Screen, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { ApiError, errorMessage } from "../../lib/api/client";
import { discoveryApi } from "../../lib/api/endpoints";
import type { Match, PublicProfile, SwipeResult } from "../../lib/api/types";
import { queryKeys } from "../../lib/queryClient";
import { showToast } from "../../lib/toast";
import type { AppStackParamList } from "../../navigation/types";
import { useProfile } from "../profile/hooks";
import { MatchModal } from "./MatchModal";
import { SwipeDeck, type SwipeDeckHandle, type SwipeDirection } from "./SwipeDeck";
import { swipeEvents } from "./swipeEvents";

const PREFETCH_THRESHOLD = 3;

export default function DiscoverScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const client = useQueryClient();
  const { colors } = useTheme();
  const { data: me } = useProfile();
  const deck = useRef<SwipeDeckHandle>(null);
  const handled = useRef(new Set<string>());
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const queueRef = useRef(queue);
  queueRef.current = queue;
  const [exhausted, setExhausted] = useState(false);
  const [match, setMatch] = useState<Match | null>(null);

  const query = useQuery({ queryKey: queryKeys.discovery, queryFn: () => discoveryApi.next(10), staleTime: 0 });

  // Merge each fetched batch into the local queue, skipping handled profiles.
  useEffect(() => {
    const batch = query.data;
    if (!batch) return;
    const known = new Set(queueRef.current.map((p) => p.userId));
    const fresh = batch.filter((p) => !known.has(p.userId) && !handled.current.has(p.userId));
    // Nothing new in this batch: stop prefetching until the user refreshes.
    setExhausted(fresh.length === 0);
    if (fresh.length) {
      setQueue((current) => [...current, ...fresh.filter((p) => !current.some((c) => c.userId === p.userId))]);
    }
  }, [query.data, query.dataUpdatedAt]);

  useEffect(() => {
    if (queue.length <= PREFETCH_THRESHOLD && !query.isFetching && !exhausted && query.isSuccess) {
      void query.refetch();
    }
  }, [exhausted, query, queue.length]);

  const onResult = useCallback(
    (userId: string, result: SwipeResult | null) => {
      handled.current.add(userId);
      setQueue((current) => current.filter((p) => p.userId !== userId));
      if (result?.matched && result.match) {
        setMatch(result.match);
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    [client]
  );

  useEffect(() => swipeEvents.subscribe(onResult), [onResult]);

  const onSwipe = async (profile: PublicProfile, direction: SwipeDirection) => {
    onResult(profile.userId, null);
    try {
      const result = await discoveryApi.swipe(profile.userId, direction);
      if (result.matched) onResult(profile.userId, result);
    } catch (e) {
      if (!(e instanceof ApiError && e.code === "already_swiped")) {
        showToast(errorMessage(e), "error");
      }
    }
  };

  const resetAndReload = () => {
    setExhausted(false);
    void query.refetch();
  };

  let content: React.ReactNode;
  if (query.isLoading) {
    content = <LoadingState label="Recherche de profils…" />;
  } else if (query.error && queue.length === 0) {
    content = <ErrorState message={errorMessage(query.error)} onRetry={resetAndReload} />;
  } else if (queue.length === 0) {
    content = (
      <EmptyState
        actionLabel="Actualiser"
        icon="compass-outline"
        message="Vous avez vu tous les profils correspondant à vos critères. Élargissez la distance ou la tranche d'âge, ou revenez plus tard."
        onAction={resetAndReload}
        title="Plus personne pour le moment"
      />
    );
  } else {
    content = (
      <>
        <View style={styles.deck}>
          <SwipeDeck onOpenDetails={(p) => navigation.navigate("ProfileDetail", { userId: p.userId, fromDiscovery: true })} onSwipe={onSwipe} profiles={queue} ref={deck} />
        </View>
        <View style={styles.actions}>
          <IconButton filled icon="close" label="Passer" onPress={() => deck.current?.swipe("pass")} size={64} testID="discover-pass" tone="muted" />
          <IconButton
            filled
            icon="information"
            label="Voir le profil complet"
            onPress={() => navigation.navigate("ProfileDetail", { userId: queue[0].userId, fromDiscovery: true })}
            size={48}
            tone="default"
          />
          <IconButton filled icon="heart" label="J'aime" onPress={() => deck.current?.swipe("like")} size={64} testID="discover-like" tone="primary" />
        </View>
      </>
    );
  }

  return (
    <Screen testID="discover-screen">
      <View style={styles.header}>
        <Text style={{ color: colors.primary }} variant="title">
          Lueur
        </Text>
        <IconButton icon="options-outline" label="Préférences de rencontre" onPress={() => navigation.navigate("Preferences")} />
      </View>
      {content}
      <MatchModal
        match={match}
        myName={me?.firstName ?? ""}
        myPhoto={me?.photos[0]}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Chat", { conversationId: m.conversationId, name: m.user.firstName });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingHorizontal: 20, paddingVertical: 8 },
  deck: { flex: 1, marginHorizontal: 12 },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: 28, paddingVertical: 16, paddingBottom: 12 }
});
