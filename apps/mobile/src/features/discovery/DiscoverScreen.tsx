import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { ApiError, errorMessage } from "../../api/client";
import { discoveryApi, notificationsApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { ProfileCard } from "../../api/types";
import type { AppStackParamList } from "../../navigation/types";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { EmptyState, ErrorState, Header, Icon, IconButton, LoadingState, Screen, Text, useFeedback } from "../../ui";
import { SwipeDeck, type Decision, type SwipeDeckHandle } from "./SwipeDeck";
import { onSwiped, useSwipe } from "./useSwipe";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export function DiscoverScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const { profile } = useSession();
  const { toast } = useFeedback();
  const deck = useRef<SwipeDeckHandle>(null);
  const [queue, setQueue] = useState<ProfileCard[]>([]);
  const seen = useRef(new Set<string>());
  const exhausted = useRef(false);
  const swipe = useSwipe();

  const batch = useQuery({ queryKey: queryKeys.discovery, queryFn: () => discoveryApi.next(10), staleTime: 0 });
  const unread = useQuery({ queryKey: queryKeys.unreadNotifications, queryFn: notificationsApi.unreadCount });

  // Merge freshly fetched candidates into the local queue without duplicates.
  useEffect(() => {
    if (!batch.data) return;
    setQueue((current) => {
      const fresh = batch.data.filter((c) => !seen.current.has(c.userId) && !current.some((q) => q.userId === c.userId));
      // Stop prefetching when a batch brings nothing new (avoids refetch loops).
      exhausted.current = fresh.length === 0;
      return fresh.length ? [...current, ...fresh] : current;
    });
  }, [batch.data]);

  useEffect(
    () =>
      onSwiped((userId) => {
        seen.current.add(userId);
        setQueue((q) => q.filter((c) => c.userId !== userId));
      }),
    []
  );

  const { refetch, isFetching } = batch;
  useEffect(() => {
    if (queue.length < 3 && !isFetching && batch.isSuccess && batch.data.length > 0 && !exhausted.current) {
      void refetch();
    }
  }, [queue.length, isFetching, refetch, batch.isSuccess, batch.data]);

  const onDecision = useCallback(
    (card: ProfileCard, decision: Decision) => {
      seen.current.add(card.userId);
      setQueue((q) => q.filter((c) => c.userId !== card.userId));
      swipe.mutate(
        { userId: card.userId, decision },
        {
          onError: (error) => {
            if (error instanceof ApiError && (error.code === "already_swiped" || error.status === 404)) return;
            // Put the card back so the action is not silently lost.
            seen.current.delete(card.userId);
            setQueue((q) => [card, ...q]);
            toast(errorMessage(error), "error");
          }
        }
      );
    },
    [swipe, toast]
  );

  const header = (
    <Header
      title="Découvrir"
      large
      right={
        <>
          <IconButton icon="options-outline" label="Préférences de recherche" onPress={() => navigation.navigate("Preferences")} />
          <IconButton
            icon="notifications-outline"
            label="Notifications"
            badge={unread.data}
            onPress={() => navigation.navigate("Notifications")}
            testID="open-notifications"
          />
        </>
      }
    />
  );

  let body: React.ReactNode;
  if (batch.isLoading && queue.length === 0) {
    body = <LoadingState label="Recherche de profils…" />;
  } else if (batch.error && queue.length === 0) {
    body = <ErrorState message={errorMessage(batch.error)} onRetry={() => void refetch()} />;
  } else if (queue.length === 0) {
    body = (
      <EmptyState
        icon="sparkles-outline"
        title="Vous avez fait le tour"
        message="Revenez un peu plus tard ou élargissez vos critères (distance, âge) pour voir plus de profils."
        actionLabel="Actualiser"
        onAction={() => {
          exhausted.current = false;
          void refetch();
        }}
      />
    );
  } else {
    body = (
      <>
        <View style={styles.deck}>
          <SwipeDeck
            ref={deck}
            cards={queue}
            onDecision={onDecision}
            onOpen={(card) => navigation.navigate("ProfileDetail", { userId: card.userId, canSwipe: true })}
          />
        </View>
        <View style={styles.actions}>
          <IconButton
            icon="close"
            label="Passer"
            tone="default"
            size={64}
            onPress={() => deck.current?.swipe("pass")}
            testID="swipe-pass"
          />
          <IconButton
            icon="heart"
            label="J'aime"
            tone="primary"
            size={72}
            onPress={() => deck.current?.swipe("like")}
            testID="swipe-like"
          />
        </View>
      </>
    );
  }

  return (
    <Screen>
      {header}
      {profile && !profile.hasLocation ? (
        <View style={[styles.banner, { backgroundColor: colors.accentSoft }]}>
          <Icon name="location-outline" size={18} color={colors.accent} />
          <Text variant="caption" style={styles.bannerText}>
            Activez la localisation dans « Préférences » pour voir des profils proches de vous.
          </Text>
        </View>
      ) : null}
      {body}
    </Screen>
  );
}

const styles = StyleSheet.create({
  deck: { flex: 1, marginTop: spacing.xs },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl, paddingVertical: spacing.md },
  banner: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    padding: spacing.sm,
    borderRadius: radii.md,
    marginBottom: spacing.xs
  },
  bannerText: { flex: 1 }
});
