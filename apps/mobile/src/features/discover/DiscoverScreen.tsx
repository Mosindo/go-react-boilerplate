import React, { useCallback, useEffect, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { fetchDiscover, sendSwipe } from "../../api/social";
import type { MatchItem, Profile } from "../../api/types";
import { appendProfiles } from "../../lib/swipeQueue";
import type { MainStackParamList, MainTabParamList } from "../../navigation/types";
import { spacing, useTheme, radii } from "../../theme";
import { Avatar, Button, EmptyState, ErrorState, LoadingState, Screen, Text, useToast } from "../../ui";
import { useNotificationBadge } from "../notifications/hooks";
import { SwipeDeck, type SwipeAction } from "./SwipeDeck";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../api/keys";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Discover">,
  NativeStackScreenProps<MainStackParamList>
>;

const BATCH = 10;
const REFILL_AT = 3;

export function DiscoverScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const toast = useToast();
  const qc = useQueryClient();
  const badge = useNotificationBadge();
  const [queue, setQueue] = useState<Profile[]>([]);
  const [match, setMatch] = useState<MatchItem | null>(null);
  const handled = useRef(new Set<string>());
  const refilling = useRef(false);

  const initial = useQuery({
    queryKey: ["discover"],
    queryFn: () => fetchDiscover(BATCH),
    staleTime: 0,
    gcTime: 0
  });

  useEffect(() => {
    if (initial.data) setQueue((q) => appendProfiles(q, initial.data.profiles, handled.current));
  }, [initial.data]);

  const refill = useCallback(async () => {
    if (refilling.current) return;
    refilling.current = true;
    try {
      const { profiles } = await fetchDiscover(BATCH);
      setQueue((q) => appendProfiles(q, profiles, handled.current));
    } catch {
      // The deck keeps working with what it has; the next swipe retries.
    } finally {
      refilling.current = false;
    }
  }, []);

  const loadedOnce = initial.isSuccess;
  useEffect(() => {
    if (loadedOnce && queue.length <= REFILL_AT) void refill();
  }, [loadedOnce, queue.length, refill]);

  const onSwipe = useCallback(
    (profile: Profile, action: SwipeAction) => {
      handled.current.add(profile.userId);
      setQueue((q) => q.filter((p) => p.userId !== profile.userId));
      sendSwipe(profile.userId, action)
        .then((result) => {
          if (result.matched && result.match) {
            setMatch(result.match);
            void qc.invalidateQueries({ queryKey: queryKeys.matches });
            void qc.invalidateQueries({ queryKey: queryKeys.conversations });
          }
        })
        .catch((error: unknown) => {
          if (error instanceof ApiError && error.status === 409) return;
          toast.show(error instanceof Error ? error.message : "Could not save your choice.", "error");
        });
    },
    [qc, toast]
  );

  let body: React.ReactNode;
  if (initial.isLoading) {
    body = <LoadingState label="Finding people near you" />;
  } else if (initial.isError && queue.length === 0) {
    body = <ErrorState error={initial.error} onRetry={() => void initial.refetch()} />;
  } else if (queue.length === 0) {
    body = (
      <EmptyState
        title="You are all caught up"
        message="No new profiles right now. Widen your distance or age range, or check back later."
        actionLabel="Refresh"
        onAction={() => void initial.refetch()}
      />
    );
  } else {
    body = (
      <SwipeDeck
        profiles={queue}
        onSwipe={onSwipe}
        onOpen={(p) => navigation.navigate("ProfileDetail", { userId: p.userId, canAct: true })}
      />
    );
  }

  return (
    <Screen>
      <View style={styles.header}>
        <Text variant="title" tone="primary">
          Lumen
        </Text>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={badge > 0 ? `Notifications, ${badge} unread` : "Notifications"}
          onPress={() => navigation.navigate("Notifications")}
          style={styles.bell}
        >
          <Text variant="label">Alerts</Text>
          {badge > 0 ? (
            <View style={[styles.badge, { backgroundColor: colors.accent }]}>
              <Text variant="caption" tone="onPrimary">
                {badge > 9 ? "9+" : badge}
              </Text>
            </View>
          ) : null}
        </Pressable>
      </View>
      <View style={styles.body}>{body}</View>
      <MatchOverlay
        match={match}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Thread", {
            conversationId: m.conversationId,
            userId: m.user.userId,
            name: m.user.firstName
          });
        }}
      />
    </Screen>
  );
}

export function MatchOverlay({
  match,
  onClose,
  onMessage
}: {
  match: MatchItem | null;
  onClose: () => void;
  onMessage: (m: MatchItem) => void;
}) {
  const { colors } = useTheme();
  return (
    <Modal visible={match !== null} transparent animationType="fade" onRequestClose={onClose}>
      <View style={[styles.overlay, { backgroundColor: colors.overlay }]}>
        {match ? (
          <View style={[styles.sheet, { backgroundColor: colors.surface }]} accessibilityViewIsModal>
            <Avatar url={match.user.photoUrl} name={match.user.firstName} size={96} />
            <Text variant="title" style={styles.center}>
              It&apos;s a match!
            </Text>
            <Text tone="muted" style={styles.center}>
              You and {match.user.firstName} liked each other.
            </Text>
            <View style={{ height: spacing.lg }} />
            <Button
              label={`Say hi to ${match.user.firstName}`}
              onPress={() => onMessage(match)}
              testID="match-message-button"
            />
            <Button label="Keep browsing" variant="ghost" onPress={onClose} />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: spacing.md
  },
  bell: { flexDirection: "row", alignItems: "center", padding: spacing.sm },
  badge: {
    marginLeft: spacing.xs,
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 4
  },
  body: { flex: 1 },
  overlay: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  sheet: { width: "100%", borderRadius: radii.lg, padding: spacing.xl, alignItems: "stretch" },
  center: { textAlign: "center", marginTop: spacing.sm }
});
