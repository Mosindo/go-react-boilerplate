import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { matchingApi } from "../../api/matching";
import type { PublicProfile, SwipeAction, SwipeResult } from "../../api/types";
import { refreshLocationIfStale } from "../../components/location";
import { queryKeys } from "../../hooks/queryClient";
import { useMyProfile, useSetMyProfile } from "../../hooks/queries";
import { markMatchCelebrated } from "../../hooks/useRealtime";
import type { AppStackParamList } from "../../navigation/types";
import { colors, spacing } from "../../theme";
import { EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, showToast } from "../../ui";
import { MatchModal } from "./MatchModal";
import { SwipeDeck, type SwipeDeckHandle } from "./SwipeDeck";

export function DiscoverScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const qc = useQueryClient();
  const me = useMyProfile();
  const setMyProfile = useSetMyProfile();
  const deckRef = useRef<SwipeDeckHandle>(null);
  const [handled, setHandled] = useState<Set<string>>(() => new Set());
  const [match, setMatch] = useState<SwipeResult["match"]>(null);

  const discover = useQuery({ queryKey: queryKeys.discover, queryFn: () => matchingApi.discover(10), staleTime: 60_000 });

  const deck = useMemo(() => (discover.data ?? []).filter((p) => !handled.has(p.userId)), [discover.data, handled]);

  const swipe = useMutation({
    mutationFn: ({ profile, action }: { profile: PublicProfile; action: SwipeAction }) => matchingApi.swipe(profile.userId, action),
    onSuccess: (result) => {
      if (result.match) {
        markMatchCelebrated(result.match.conversationId);
        setMatch(result.match);
        void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    onError: (error) => showToast(errorMessage(error, "Votre choix n'a pas pu être enregistré."), "error")
  });

  const swipePending = swipe.isPending;

  // Fetch the next batch before the deck runs out. Only refetch once part of
  // the current batch was answered (and recorded), otherwise the server has
  // nothing new and refetching would loop.
  const batchConsumed = (discover.data ?? []).some((p) => handled.has(p.userId));
  const { refetch, isFetching } = discover;
  useEffect(() => {
    if (batchConsumed && deck.length <= 2 && !isFetching && !swipePending) {
      void refetch();
    }
  }, [batchConsumed, deck.length, isFetching, swipePending, refetch]);

  const deckEmpty = deck.length === 0;
  useFocusEffect(
    useCallback(() => {
      void refreshLocationIfStale(me.data).then((updated) => {
        if (updated) {
          setMyProfile(updated);
        }
      });
      if (deckEmpty) {
        void refetch();
      }
    }, [me.data, setMyProfile, deckEmpty, refetch])
  );

  const onSwiped = (profile: PublicProfile, action: SwipeAction) => {
    setHandled((prev) => new Set(prev).add(profile.userId));
    swipe.mutate({ profile, action });
  };

  const top = deck[0];

  let body: React.ReactNode;
  if (discover.isLoading) {
    body = <LoadingState label="Recherche de personnes près de vous…" />;
  } else if (discover.error) {
    body = <ErrorState message={errorMessage(discover.error)} onRetry={() => void discover.refetch()} />;
  } else if (!top) {
    body = (
      <EmptyState
        icon="compass-outline"
        title="Personne de nouveau pour l'instant"
        body="Revenez un peu plus tard ou élargissez vos critères de distance et d'âge."
        actionLabel={discover.isFetching ? undefined : "Ajuster mes préférences"}
        onAction={() => navigation.navigate("Preferences")}
        testID="discover-empty"
      />
    );
  } else {
    body = (
      <>
        <SwipeDeck
          ref={deckRef}
          profiles={deck}
          onSwiped={onSwiped}
          onOpen={(p) => navigation.navigate("ProfileDetail", { userId: p.userId })}
        />
        <View style={styles.actions}>
          <IconButton
            icon="close"
            label={`Passer ${top.firstName}`}
            onPress={() => deckRef.current?.swipe("pass")}
            size={64}
            color={colors.textMuted}
            background={colors.surfaceHigh}
            testID="discover-pass"
          />
          <IconButton
            icon="heart"
            label={`Aimer ${top.firstName}`}
            onPress={() => deckRef.current?.swipe("like")}
            size={72}
            color={colors.onPrimary}
            background={colors.primary}
            testID="discover-like"
          />
        </View>
      </>
    );
  }

  return (
    <Screen edges={["top"]} testID="discover-screen">
      <Header
        title="Découvrir"
        large
        right={<IconButton icon="options-outline" label="Préférences" onPress={() => navigation.navigate("Preferences")} />}
      />
      <View style={styles.body}>{body}</View>
      <MatchModal
        match={match}
        myName={me.data?.profile?.firstName ?? ""}
        myPhoto={me.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={() => {
          const conversationId = match?.conversationId;
          setMatch(null);
          if (conversationId) {
            navigation.navigate("Chat", { conversationId });
          }
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: {
    flex: 1,
    paddingTop: spacing.xs,
    paddingBottom: spacing.md
  },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xl,
    paddingTop: spacing.md
  }
});
