import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { discoverApi, notificationApi } from "../../api/endpoints";
import type { Card } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import { EmptyState, ErrorState, Icon, LoadingState, Screen, Text } from "../../ui";
import { onHandled } from "./deckBus";
import { ProfileCardView } from "./ProfileCardView";
import { SwipeCard, type SwipeCardHandle, type SwipeDirection } from "./SwipeCard";
import { useSwipe } from "./useSwipe";

const REFILL_AT = 3;

export function DiscoverScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const swipe = useSwipe();
  const deck = useRef<SwipeCardHandle>(null);

  const [queue, setQueue] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [exhausted, setExhausted] = useState(false);
  const handled = useRef(new Set<string>());
  const inFlight = useRef(false);

  const { data: unread = 0 } = useQuery({
    queryKey: queryKeys.notificationsUnread,
    queryFn: notificationApi.unread,
  });

  const fill = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    try {
      const feed = await discoverApi.feed(10);
      setError(null);
      setQueue((current) => {
        const known = new Set(current.map((c) => c.userId));
        const fresh = feed.filter((c) => !known.has(c.userId) && !handled.current.has(c.userId));
        if (fresh.length === 0) setExhausted(true);
        return [...current, ...fresh];
      });
    } catch (e) {
      setError(e);
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fill();
  }, [fill]);

  useEffect(() => {
    if (!loading && !exhausted && !error && queue.length <= REFILL_AT) void fill();
  }, [queue.length, loading, exhausted, error, fill]);

  useEffect(
    () =>
      onHandled((userId) => {
        handled.current.add(userId);
        setQueue((q) => q.filter((c) => c.userId !== userId));
      }),
    [],
  );

  const top = queue[0];
  const next = queue[1];

  const onSwiped = (direction: SwipeDirection) => {
    if (!top) return;
    const target = top;
    handled.current.add(target.userId);
    setQueue((q) => q.filter((c) => c.userId !== target.userId)); // optimistic: the next card is instant
    void swipe(target.userId, direction === "right" ? "like" : "pass").then((ok) => {
      if (!ok) {
        handled.current.delete(target.userId);
        setQueue((q) => [target, ...q]);
      }
    });
  };

  const refresh = () => {
    setExhausted(false);
    setLoading(true);
    void fill();
  };

  let body: React.ReactNode;
  if (loading && queue.length === 0) body = <LoadingState label="Recherche de profils…" />;
  else if (error && queue.length === 0) body = <ErrorState error={error} onRetry={refresh} />;
  else if (!top)
    body = (
      <EmptyState
        testID="discover-empty"
        icon="sparkles-outline"
        title="Vous avez fait le tour pour l'instant"
        message="Revenez plus tard ou élargissez vos préférences (âge, distance) pour voir plus de monde."
        actionLabel="Actualiser"
        actionTestID="discover-refresh"
        onAction={refresh}
      />
    );
  else
    body = (
      <>
        <View style={styles.deck}>
          {next ? (
            <View style={[StyleSheet.absoluteFill, styles.behind]} pointerEvents="none">
              <ProfileCardView key={next.userId} card={next} onOpenProfile={() => undefined} />
            </View>
          ) : null}
          <SwipeCard key={top.userId} ref={deck} onSwiped={onSwiped}>
            <ProfileCardView
              card={top}
              onOpenProfile={() =>
                navigation.navigate("ProfileDetail", { userId: top.userId, canSwipe: true })
              }
            />
          </SwipeCard>
        </View>
        <View style={styles.actions}>
          <RoundButton
            testID="discover-pass"
            label="Passer"
            icon="close"
            color={colors.danger}
            onPress={() => deck.current?.swipe("left")}
          />
          <RoundButton
            testID="discover-like"
            label="J'aime"
            icon="heart"
            color={colors.success}
            onPress={() => deck.current?.swipe("right")}
          />
        </View>
      </>
    );

  return (
    <Screen testID="discover-screen" edges={["top"]} padded={false}>
      <View style={styles.header}>
        <Text variant="title">Alba</Text>
        <Pressable
          testID="open-notifications"
          accessibilityRole="button"
          accessibilityLabel={unread > 0 ? `Notifications, ${unread} non lues` : "Notifications"}
          onPress={() => navigation.navigate("Notifications")}
          style={styles.bell}
        >
          <Icon name="notifications-outline" size={26} />
          {unread > 0 ? (
            <View style={[styles.badge, { backgroundColor: colors.danger }]}>
              <Text variant="caption" color="#FFFFFF" testID="notifications-badge">
                {unread > 9 ? "9+" : unread}
              </Text>
            </View>
          ) : null}
        </Pressable>
      </View>
      <View style={styles.body}>{body}</View>
    </Screen>
  );
}

function RoundButton({
  icon,
  color,
  label,
  onPress,
  testID,
}: {
  icon: "close" | "heart";
  color: string;
  label: string;
  onPress: () => void;
  testID: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        { backgroundColor: colors.surface, borderColor: color, opacity: pressed ? 0.8 : 1 },
      ]}
    >
      <Icon name={icon} size={32} color={color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
  },
  bell: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  badge: {
    position: "absolute",
    top: 4,
    right: 2,
    minWidth: 18,
    height: 18,
    borderRadius: 9,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 4,
  },
  body: { flex: 1 },
  deck: { flex: 1, marginHorizontal: spacing.lg },
  behind: { transform: [{ scale: 0.95 }], opacity: 0.9 },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xxl,
    paddingVertical: spacing.lg,
  },
  round: {
    width: 68,
    height: 68,
    borderRadius: radius.pill,
    borderWidth: 2,
    alignItems: "center",
    justifyContent: "center",
  },
});
