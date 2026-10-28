import React, { useCallback, useEffect, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { keys } from "../../api/keys";
import { fetchDiscover, resetPasses, swipe } from "../../api/social";
import type { Conversation, PublicProfile, SwipeAction } from "../../api/types";
import { AuthImage, Button, EmptyState, ErrorState, LoadingState, Screen, SwipeDeck, Text, useToast, type SwipeDeckHandle } from "../../components";
import { errorMessage } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { MIN_TOUCH, radii, spacing, useTheme } from "../../theme";

const PAGE = 10;
const REFILL_AT = 3;

export function DiscoverScreen() {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const deck = useRef<SwipeDeckHandle>(null);
  const seen = useRef(new Set<string>());
  const refilling = useRef(false);
  const [match, setMatch] = useState<Conversation | null>(null);
  const [resetting, setResetting] = useState(false);

  const query = useQuery<PublicProfile[], Error>({
    queryKey: keys.discover,
    queryFn: () => fetchDiscover(PAGE),
    staleTime: Infinity
  });
  const profiles = query.data ?? [];

  const refill = useCallback(async () => {
    if (refilling.current) return;
    refilling.current = true;
    try {
      const more = await fetchDiscover(PAGE);
      queryClient.setQueryData<PublicProfile[]>(keys.discover, (current = []) => {
        const have = new Set(current.map((p) => p.userId));
        return [...current, ...more.filter((p) => !have.has(p.userId) && !seen.current.has(p.userId))];
      });
    } catch {
      // The next swipe retries; the deck keeps working with what it has.
    } finally {
      refilling.current = false;
    }
  }, [queryClient]);

  useEffect(() => {
    if (query.isSuccess && profiles.length > 0 && profiles.length <= REFILL_AT) {
      void refill();
    }
  }, [profiles.length, query.isSuccess, refill]);

  const handleSwipe = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      seen.current.add(profile.userId);
      queryClient.setQueryData<PublicProfile[]>(keys.discover, (current = []) => current.filter((p) => p.userId !== profile.userId));
      try {
        const result = await swipe(profile.userId, action);
        if (result.matched && result.conversation) {
          setMatch(result.conversation);
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
        }
      } catch (error) {
        seen.current.delete(profile.userId);
        queryClient.setQueryData<PublicProfile[]>(keys.discover, (current = []) => [profile, ...current]);
        toast.show(errorMessage(error), "error");
      }
    },
    [queryClient, toast]
  );

  const doReset = async (): Promise<void> => {
    setResetting(true);
    try {
      await resetPasses();
      seen.current.clear();
      await query.refetch();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setResetting(false);
    }
  };

  const openProfile = (profile: PublicProfile): void => navigation.navigate("ProfileDetail", { userId: profile.userId });

  let body: React.ReactNode;
  if (query.isPending) {
    body = <LoadingState label="Finding people near you" />;
  } else if (query.isError) {
    body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  } else if (profiles.length === 0) {
    body = (
      <View style={{ flex: 1 }}>
        <EmptyState
          testID="discover-empty"
          title="You're all caught up"
          message="No one new matches your preferences right now. Widen your filters, bring back profiles you passed, or check back soon."
          actionLabel={resetting ? "Working…" : "Show passed profiles again"}
          onAction={() => void doReset()}
          secondaryLabel="Adjust my preferences"
          onSecondary={() => navigation.navigate("Preferences")}
        />
        <Button title="Refresh" variant="ghost" onPress={() => void query.refetch()} />
      </View>
    );
  } else {
    body = (
      <>
        <View style={styles.deckWrap}>
          <SwipeDeck ref={deck} profiles={profiles} onSwipe={(p, a) => void handleSwipe(p, a)} onOpenProfile={openProfile} />
        </View>
        <View style={styles.actions}>
          <RoundButton label="Pass" glyph="✕" color={colors.pass} onPress={() => deck.current?.swipe("pass")} testID="discover-pass" />
          <RoundButton label="Like" glyph="♥" color={colors.like} big onPress={() => deck.current?.swipe("like")} testID="discover-like" />
        </View>
      </>
    );
  }

  return (
    <Screen padded={false} edges={[]} testID="discover-screen">
      <View style={styles.flex}>{body}</View>
      <Modal visible={match !== null} transparent animationType="fade" onRequestClose={() => setMatch(null)}>
        <View style={[styles.modalBackdrop, { backgroundColor: colors.overlay }]} testID="match-modal">
          {match ? (
            <View style={[styles.modalCard, { backgroundColor: colors.surface }]}>
              <AuthImage url={match.user.photoUrl} style={styles.matchPhoto} fallbackInitial={match.user.firstName} label={match.user.firstName} />
              <Text variant="title" style={{ textAlign: "center" }}>
                It&apos;s a match!
              </Text>
              <Text tone="muted" style={{ textAlign: "center" }}>
                You and {match.user.firstName} like each other.
              </Text>
              <Button
                title="Send a message"
                testID="match-send-message"
                onPress={() => {
                  const target = match;
                  setMatch(null);
                  navigation.navigate("Chat", { conversationId: target.id, name: target.user.firstName, userId: target.user.userId });
                }}
              />
              <Button title="Keep swiping" variant="ghost" testID="match-keep-swiping" onPress={() => setMatch(null)} />
            </View>
          ) : null}
        </View>
      </Modal>
    </Screen>
  );
}

function RoundButton({ label, glyph, color, onPress, big, testID }: { label: string; glyph: string; color: string; onPress: () => void; big?: boolean; testID: string }) {
  const { colors } = useTheme();
  const size = big ? 72 : 60;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.round,
        { width: size, height: size, borderRadius: size / 2, borderColor: color, backgroundColor: colors.surface, transform: [{ scale: pressed ? 0.94 : 1 }] }
      ]}
    >
      <Text style={{ color, fontSize: big ? 32 : 26, fontWeight: "700" }}>{glyph}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  deckWrap: { flex: 1, marginHorizontal: spacing.lg, marginTop: spacing.sm },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl, paddingVertical: spacing.lg, minHeight: MIN_TOUCH },
  round: { borderWidth: 2, alignItems: "center", justifyContent: "center", elevation: 3, shadowColor: "#000", shadowOpacity: 0.12, shadowRadius: 6, shadowOffset: { width: 0, height: 2 } },
  modalBackdrop: { flex: 1, justifyContent: "center", padding: spacing.xl },
  modalCard: { borderRadius: radii.lg, padding: spacing.xl, gap: spacing.md, alignItems: "stretch" },
  matchPhoto: { width: 140, height: 140, borderRadius: 70, alignSelf: "center" }
});
