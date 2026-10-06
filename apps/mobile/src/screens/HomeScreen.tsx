import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { discover, swipe } from "../api/dating";
import type { PublicProfile, SwipeAction } from "../api/types";
import { MatchModal } from "../components/MatchModal";
import { SwipeDeck, type SwipeDeckHandle } from "../components/SwipeDeck";
import { useAuth } from "../hooks/useAuth";
import { useNotifications } from "../hooks/queries";
import { describeError } from "../lib/errors";
import type { RootStackParamList, TabParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Text, radii, shadows, spacing, useTheme } from "../shared/ui";
import { onSwiped } from "../state/swipeBus";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Discover">, NativeStackScreenProps<RootStackParamList>>;

const REFILL_AT = 3;

type MatchInfo = { matchId: string; profile: PublicProfile };

export default function HomeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { self } = useAuth();
  const queryClient = useQueryClient();
  const notifications = useNotifications();
  const deck = useRef<SwipeDeckHandle>(null);

  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const [match, setMatch] = useState<MatchInfo | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);

  const fetchMore = useCallback(async () => {
    if (fetching.current) {
      return;
    }
    fetching.current = true;
    setLoading(true);
    try {
      const batch = await discover(10);
      const fresh = batch.filter((profile) => !seen.current.has(profile.userId));
      fresh.forEach((profile) => seen.current.add(profile.userId));
      setQueue((current) => [...current, ...fresh]);
      setExhausted(fresh.length === 0);
      setLoadError(null);
    } catch (err) {
      setLoadError(describeError(err));
    } finally {
      fetching.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchMore();
  }, [fetchMore]);

  useEffect(() => {
    if (queue.length < REFILL_AT && !exhausted && !loadError && !fetching.current) {
      void fetchMore();
    }
  }, [queue.length, exhausted, loadError, fetchMore]);

  // A profile opened from the deck can be liked/passed there: drop it here too.
  useEffect(() => onSwiped((userId) => setQueue((current) => current.filter((p) => p.userId !== userId))), []);

  const refresh = useCallback(() => {
    seen.current = new Set(); // server-side filtering already hides everyone processed
    setQueue([]);
    setExhausted(false);
    setLoadError(null);
    void fetchMore();
  }, [fetchMore]);

  const handleSwipe = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      setQueue((current) => current.filter((p) => p.userId !== profile.userId));
      try {
        const result = await swipe(profile.userId, action);
        if (result.matched && !result.alreadyProcessed && result.matchId) {
          setMatch({ matchId: result.matchId, profile });
          void queryClient.invalidateQueries({ queryKey: ["matches"] });
        }
        if (action === "like") {
          void queryClient.invalidateQueries({ queryKey: ["likes"] });
        }
      } catch (err) {
        const status = (err as { status?: number }).status;
        if (status === 404 || status === 400) {
          // The profile is gone (blocked, suspended, deleted): never put it back.
          showToast("Ce profil n'est plus disponible.");
          return;
        }
        // Transient failure: put the card back so nothing is silently lost.
        setQueue((current) => [profile, ...current]);
        showToast(describeError(err, "Action impossible. Réessayez."), "danger");
      }
    },
    [queryClient]
  );

  const unread = notifications.data?.unreadCount ?? 0;
  const top = queue[0];

  const myPhoto = self?.photos[0]?.url;

  return (
    <SafeAreaLayout edges={["top", "left", "right"]}>
      <View style={styles.header}>
        <Text style={{ color: colors.primary }} variant="heading">
          Lien
        </Text>
        <Pressable
          accessibilityLabel={unread > 0 ? `Notifications, ${unread} non lues` : "Notifications"}
          accessibilityRole="button"
          hitSlop={12}
          onPress={() => navigation.navigate("Notifications")}
          testID="open-notifications"
        >
          <Ionicons color={colors.text} name="notifications-outline" size={26} />
          {unread > 0 ? (
            <View style={[styles.dot, { backgroundColor: colors.primary }]}>
              <Text style={{ color: colors.primaryForeground, fontSize: 10, lineHeight: 12 }} weight="bold">
                {unread > 9 ? "9+" : unread}
              </Text>
            </View>
          ) : null}
        </Pressable>
      </View>

      <View style={styles.stage} testID="home-screen">
        {top ? (
          <SwipeDeck
            cards={queue}
            onOpenProfile={(profile) => navigation.navigate("ProfileDetail", { userId: profile.userId, context: "discover" })}
            onSwipe={handleSwipe}
            ref={deck}
          />
        ) : loading ? (
          <LoadingView fullScreen label="On cherche des personnes pour vous…" />
        ) : loadError ? (
          <ErrorView message={loadError} onAction={refresh} />
        ) : (
          <EmptyView
            actionLabel="Actualiser"
            icon="sparkles-outline"
            message="Revenez plus tard, ou élargissez votre distance et votre tranche d'âge pour voir plus de monde."
            onAction={refresh}
            testID="discover-empty"
            title="Vous avez tout vu pour l'instant"
          />
        )}
      </View>

      {top ? (
        <View style={styles.actions}>
          <Pressable
            accessibilityLabel="Passer"
            accessibilityRole="button"
            onPress={() => deck.current?.swipe("pass")}
            style={[styles.round, shadows.card, { backgroundColor: colors.surface, borderColor: colors.border }]}
            testID="pass-button"
          >
            <Ionicons color={colors.pass} name="close" size={32} />
          </Pressable>
          <Pressable
            accessibilityLabel="J'aime"
            accessibilityRole="button"
            onPress={() => deck.current?.swipe("like")}
            style={[styles.round, styles.roundLarge, shadows.card, { backgroundColor: colors.primary }]}
            testID="like-button"
          >
            <Ionicons color={colors.primaryForeground} name="heart" size={34} />
          </Pressable>
        </View>
      ) : null}

      <MatchModal
        myName={self?.profile?.firstName ?? ""}
        myPhotoPath={myPhoto}
        onClose={() => setMatch(null)}
        onMessage={() => {
          if (match) {
            const info = match;
            setMatch(null);
            navigation.navigate("Chat", { matchId: info.matchId, peerId: info.profile.userId, peerName: info.profile.firstName });
          }
        }}
        otherName={match?.profile.firstName ?? ""}
        otherPhotoPath={match?.profile.photos[0]?.url}
        visible={match !== null}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.xl,
    paddingVertical: spacing.sm
  },
  dot: {
    position: "absolute",
    top: -4,
    right: -6,
    minWidth: 16,
    height: 16,
    borderRadius: 8,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 3
  },
  stage: { flex: 1, marginHorizontal: spacing.lg },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xxl, paddingVertical: spacing.lg },
  round: { width: 64, height: 64, borderRadius: radii.pill, borderWidth: 1, alignItems: "center", justifyContent: "center" },
  roundLarge: { width: 76, height: 76, borderWidth: 0 }
});
