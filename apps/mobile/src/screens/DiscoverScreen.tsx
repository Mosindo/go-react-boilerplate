import React, { useCallback, useEffect, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { matchingApi } from "../api/endpoints";
import type { PublicProfile, SwipeAction } from "../api/types";
import { SwipeCard, type SwipeCardHandle } from "../features/SwipeCard";
import { keys } from "../features/keys";
import { useMyProfile, useUnreadNotifications } from "../features/hooks";
import { describeError } from "../lib/validation";
import { useNav } from "../nav/Navigation";
import { radii, useTheme } from "../theme/theme";
import { Button, EmptyState, ErrorState, LoadingView, PhotoImage, Screen, Text, useToast } from "../ui";

const REFILL_AT = 3;

type MatchInfo = { conversationId: string; profile: PublicProfile };

export default function DiscoverScreen() {
  const { colors } = useTheme();
  const nav = useNav();
  const toast = useToast();
  const queryClient = useQueryClient();
  const me = useMyProfile();
  const unread = useUnreadNotifications();

  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [exhausted, setExhausted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [match, setMatch] = useState<MatchInfo | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);
  const cardRef = useRef<SwipeCardHandle>(null);

  const load = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    try {
      const batch = await matchingApi.discover(10);
      const fresh = batch.filter((p) => !seen.current.has(p.userId));
      fresh.forEach((p) => seen.current.add(p.userId));
      setQueue((q) => [...q, ...fresh]);
      setExhausted(fresh.length === 0);
      setError(null);
    } catch (e) {
      setError(describeError(e));
    } finally {
      fetching.current = false;
      setLoading(false);
    }
  }, []);

  const reload = useCallback(() => {
    seen.current.clear();
    setQueue([]);
    setExhausted(false);
    setLoading(true);
    void load();
  }, [load]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!loading && !exhausted && !error && queue.length <= REFILL_AT) void load();
  }, [queue.length, loading, exhausted, error, load]);

  const swipe = useMutation({
    mutationFn: ({ profile, action }: { profile: PublicProfile; action: SwipeAction }) => matchingApi.swipe(profile.userId, action),
    onSuccess: (result, { profile }) => {
      if (result.matched && result.conversationId && !result.alreadySwiped) {
        setMatch({ conversationId: result.conversationId, profile });
        void queryClient.invalidateQueries({ queryKey: keys.conversations });
        void queryClient.invalidateQueries({ queryKey: keys.notifications });
      }
    },
    onError: (e) => toast.show(describeError(e), "error")
  });

  const onSwiped = (profile: PublicProfile, action: SwipeAction) => {
    setQueue((q) => q.filter((p) => p.userId !== profile.userId));
    swipe.mutate({ profile, action });
  };

  const resetPasses = useMutation({
    mutationFn: matchingApi.resetPasses,
    onSuccess: reload,
    onError: (e) => toast.show(describeError(e), "error")
  });

  const top = queue[0];
  const next = queue[1];
  const bell = (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`Notifications${unread.data ? `, ${unread.data} non lues` : ""}`}
      onPress={() => nav.push({ name: "notifications" })}
      hitSlop={10}
    >
      <Text variant="heading">🔔{unread.data ? ` ${unread.data}` : ""}</Text>
    </Pressable>
  );

  let body: React.ReactNode;
  if (me.data && !me.data.onboardingComplete) {
    body = <EmptyState title="Complétez votre profil" message="Ajoutez une photo et vos préférences pour découvrir des profils." />;
  } else if (loading) {
    body = <LoadingView label="Recherche de profils…" />;
  } else if (error && queue.length === 0) {
    body = <ErrorState message={error} onRetry={reload} />;
  } else if (!top) {
    body = (
      <EmptyState
        title="Vous avez tout vu pour le moment"
        message="Revenez plus tard, élargissez vos préférences, ou redonnez une chance aux profils passés."
        actionLabel="Revoir les profils passés"
        onAction={() => resetPasses.mutate()}
      />
    );
  } else {
    body = (
      <View style={styles.deck}>
        <View style={styles.cardArea}>
          {next ? (
            <View style={[styles.behind]} pointerEvents="none">
              <PhotoImage url={next.photos[0]?.url} name={next.firstName} style={StyleSheet.absoluteFill} />
            </View>
          ) : null}
          <SwipeCard
            key={top.userId}
            ref={cardRef}
            profile={top}
            interactive
            onSwiped={(action) => onSwiped(top, action)}
            onOpenProfile={() => nav.push({ name: "profile", userId: top.userId })}
          />
        </View>
        <View style={styles.actions}>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Passer"
            onPress={() => cardRef.current?.swipe("pass")}
            style={[styles.round, { backgroundColor: colors.surface, borderColor: colors.border }]}
            testID="swipe-pass"
          >
            <Text variant="title" color={colors.pass}>
              ✕
            </Text>
          </Pressable>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="J'aime"
            onPress={() => cardRef.current?.swipe("like")}
            style={[styles.round, styles.likeBtn, { backgroundColor: colors.primary }]}
            testID="swipe-like"
          >
            <Text variant="title" color={colors.onPrimary}>
              ♥
            </Text>
          </Pressable>
        </View>
      </View>
    );
  }

  return (
    <Screen title="Découvrir" right={bell} noBottomInset>
      {body}
      <Modal visible={!!match} transparent animationType="fade" onRequestClose={() => setMatch(null)}>
        {match ? (
          <View style={[styles.matchBackdrop, { backgroundColor: colors.overlay }]}>
            <View style={[styles.matchCard, { backgroundColor: colors.surface }]}>
              <Text variant="title" color={colors.primary}>
                C'est un match !
              </Text>
              <PhotoImage url={match.profile.photos[0]?.url} name={match.profile.firstName} circle style={styles.matchPhoto} />
              <Text muted style={styles.center}>
                Vous et {match.profile.firstName} vous plaisez mutuellement.
              </Text>
              <Button
                title="Envoyer un message"
                onPress={() => {
                  const info = match;
                  setMatch(null);
                  nav.push({
                    name: "chat",
                    conversationId: info.conversationId,
                    peer: { userId: info.profile.userId, firstName: info.profile.firstName, photoUrl: info.profile.photos[0]?.url }
                  });
                }}
                testID="match-send-message"
              />
              <Button title="Continuer à découvrir" variant="ghost" onPress={() => setMatch(null)} />
            </View>
          </View>
        ) : null}
      </Modal>
    </Screen>
  );
}

const styles = StyleSheet.create({
  deck: { flex: 1, padding: 16, gap: 16 },
  cardArea: { flex: 1 },
  behind: { ...StyleSheet.absoluteFillObject, borderRadius: radii.lg, overflow: "hidden", transform: [{ scale: 0.95 }], opacity: 0.6 },
  actions: { flexDirection: "row", justifyContent: "center", gap: 28, paddingBottom: 8 },
  round: { width: 68, height: 68, borderRadius: 34, alignItems: "center", justifyContent: "center", borderWidth: 1, elevation: 2 },
  likeBtn: { borderWidth: 0 },
  matchBackdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24 },
  matchCard: { width: "100%", borderRadius: radii.lg, padding: 24, alignItems: "center", gap: 16 },
  matchPhoto: { width: 140, height: 140 },
  center: { textAlign: "center" }
});
