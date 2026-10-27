import React, { useCallback, useEffect, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { discoveryApi } from "../api/endpoints";
import type { PublicProfile, SwipeAction } from "../api/types";
import { SwipeDeck, type DeckCommand } from "../components/SwipeDeck";
import type { MainStackParamList } from "../navigation/types";
import { keys } from "../state/keys";
import { useMyProfile } from "../state/queries";
import { radii, usePalette } from "../theme";
import { Avatar } from "../ui/AuthImage";
import { Button } from "../ui/Button";
import { Screen } from "../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../ui/states";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";

const BATCH = 10;
const REFILL_AT = 3;

type MatchInfo = { profile: PublicProfile; conversationId: string };

export function DiscoverScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const queryClient = useQueryClient();
  const toast = useToast();
  const p = usePalette();
  const myProfile = useMyProfile();

  const [deck, setDeck] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [exhausted, setExhausted] = useState(false);
  const [command, setCommand] = useState<DeckCommand>(null);
  const [match, setMatch] = useState<MatchInfo | null>(null);
  const handled = useRef(new Set<string>());
  const fetching = useRef(false);
  const deckIds = useRef(new Set<string>());

  const load = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    try {
      const batch = await discoveryApi.feed(BATCH);
      const fresh = batch.filter(
        (x) => !handled.current.has(x.userId) && !deckIds.current.has(x.userId)
      );
      fresh.forEach((x) => deckIds.current.add(x.userId));
      setDeck((current) => [...current, ...fresh]);
      setExhausted(fresh.length === 0);
      setStatus("ready");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load profiles.");
      setStatus((s) => (s === "loading" ? "error" : s));
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (status === "ready" && !exhausted && deck.length <= REFILL_AT) void load();
  }, [deck.length, status, exhausted, load]);

  const handleSwipe = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      handled.current.add(profile.userId);
      deckIds.current.delete(profile.userId);
      setDeck((current) => current.filter((x) => x.userId !== profile.userId));
      setCommand(null);
      try {
        const result = await discoveryApi.swipe(profile.userId, action);
        if (result.matched && result.conversationId) {
          setMatch({ profile, conversationId: result.conversationId });
          void queryClient.invalidateQueries({ queryKey: keys.matches });
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
        }
      } catch (e) {
        // 409/404: already handled or no longer available, nothing to restore.
        if (e instanceof ApiError && (e.status === 409 || e.status === 404)) return;
        handled.current.delete(profile.userId);
        deckIds.current.add(profile.userId);
        setDeck((current) => [profile, ...current]);
        toast.show(e instanceof Error ? e.message : "Could not save your choice.", "error");
      }
    },
    [queryClient, toast]
  );

  const top = deck[0];
  const trigger = (action: SwipeAction) => top && setCommand({ id: top.userId, action });

  const openInfo = (profile: PublicProfile) =>
    navigation.navigate("UserProfile", { userId: profile.userId });

  const refresh = () => {
    setStatus("loading");
    setExhausted(false);
    void load();
  };

  let body: React.ReactNode;
  if (status === "loading") body = <Loading label="Finding people near you…" />;
  else if (status === "error" && deck.length === 0)
    body = <ErrorState message={error} onRetry={refresh} />;
  else if (!top)
    body = (
      <EmptyState
        title="You're all caught up"
        message="No new profiles right now. Widen your preferences or check back soon."
        actionLabel="Refresh"
        onAction={refresh}
      />
    );
  else
    body = (
      <>
        <View style={styles.deckWrap}>
          <SwipeDeck
            profiles={deck.slice(0, 2)}
            command={command}
            onSwipe={handleSwipe}
            onInfo={openInfo}
          />
        </View>
        <View style={styles.actions}>
          <RoundButton
            label="Pass"
            glyph="✕"
            color={p.textMuted}
            onPress={() => trigger("pass")}
            testID="pass-button"
          />
          <RoundButton
            label="Like"
            glyph="♥"
            color={p.primary}
            onPress={() => trigger("like")}
            testID="like-button"
            big
          />
        </View>
      </>
    );

  return (
    <Screen contentStyle={styles.screen}>
      <Text variant="heading" tone="primary" style={styles.brand} accessibilityRole="header">
        ember
      </Text>
      {body}
      <Modal
        visible={!!match}
        transparent
        animationType="fade"
        onRequestClose={() => setMatch(null)}
      >
        <View style={[styles.modal, { backgroundColor: p.overlay }]}>
          <View style={[styles.modalCard, { backgroundColor: p.surface }]} accessibilityViewIsModal>
            <Text variant="title" tone="primary" style={styles.center}>
              {"It's a match!"}
            </Text>
            <View style={styles.avatars}>
              <Avatar size={88} photoId={myProfile.data?.photos[0]?.id} label="Your photo" />
              <Avatar
                size={88}
                photoId={match?.profile.photos[0]?.id}
                label={`${match?.profile.firstName ?? ""}'s photo`}
              />
            </View>
            <Text tone="muted" style={styles.center}>
              You and {match?.profile.firstName} liked each other.
            </Text>
            <Button
              label="Send a message"
              testID="match-send-message"
              onPress={() => {
                if (!match) return;
                const target = match;
                setMatch(null);
                navigation.navigate("Chat", {
                  conversationId: target.conversationId,
                  peer: {
                    userId: target.profile.userId,
                    firstName: target.profile.firstName,
                    photoId: target.profile.photos[0]?.id
                  }
                });
              }}
            />
            <Button label="Keep swiping" variant="ghost" onPress={() => setMatch(null)} />
          </View>
        </View>
      </Modal>
    </Screen>
  );
}

function RoundButton({
  label,
  glyph,
  color,
  onPress,
  big,
  testID
}: {
  label: string;
  glyph: string;
  color: string;
  onPress: () => void;
  big?: boolean;
  testID: string;
}) {
  const p = usePalette();
  const size = big ? 72 : 60;
  return (
    <Pressable
      testID={testID}
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={label}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          borderColor: color,
          backgroundColor: p.surface,
          opacity: pressed ? 0.8 : 1
        }
      ]}
    >
      <Text style={{ color, fontSize: big ? 30 : 24 }}>{glyph}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  screen: { paddingHorizontal: 16, paddingBottom: 8 },
  brand: { marginBottom: 8, marginLeft: 4 },
  deckWrap: { flex: 1, marginBottom: 14 },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: 28,
    paddingBottom: 8
  },
  round: { borderWidth: 2, alignItems: "center", justifyContent: "center" },
  modal: { flex: 1, justifyContent: "center", padding: 24 },
  modalCard: { borderRadius: radii.xl, padding: 24, gap: 16 },
  avatars: { flexDirection: "row", justifyContent: "center", gap: 16 },
  center: { textAlign: "center" }
});
