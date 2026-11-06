import React, { useCallback, useEffect, useRef, useState } from "react";
import { Modal, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/errors";
import { discoverApi } from "../api/services";
import type { PublicProfile, SwipeAction, SwipeResult } from "../api/types";
import { Avatar } from "../components/Avatar";
import { DeckActions, SwipeDeck, type SwipeDeckHandle } from "../components/SwipeDeck";
import { useToast } from "../components/Toast";
import { Button, EmptyState, ErrorState, IconButton, Loader, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { onProfileDecided } from "../lib/deckBus";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";

const BATCH = 10;
const REFILL_AT = 3;

export function DiscoverScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const toast = useToast();
  const deck = useRef<SwipeDeckHandle>(null);

  const [cards, setCards] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [match, setMatch] = useState<{ profile: PublicProfile; result: SwipeResult } | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);

  const load = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    try {
      const batch = await discoverApi.next(BATCH);
      setCards((cur) => {
        const known = new Set(cur.map((c) => c.id));
        const fresh = batch.filter((p) => !known.has(p.id) && !seen.current.has(p.id));
        return [...cur, ...fresh];
      });
      setStatus("ready");
    } catch {
      setStatus((s) => (s === "loading" ? "error" : s));
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(
    () =>
      onProfileDecided((id) => {
        seen.current.add(id);
        setCards((cur) => cur.filter((c) => c.id !== id));
      }),
    []
  );

  useEffect(() => {
    if (status === "ready" && cards.length <= REFILL_AT) void load();
  }, [cards.length, status, load]);

  const onSwipe = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      seen.current.add(profile.id);
      setCards((cur) => cur.filter((c) => c.id !== profile.id));
      try {
        const result = await discoverApi.swipe(profile.id, action);
        if (result.matched && !result.duplicate) {
          setMatch({ profile, result });
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
        }
      } catch (error) {
        // The decision was not stored: put the card back on top so nothing is lost.
        seen.current.delete(profile.id);
        setCards((cur) => [profile, ...cur]);
        toast.show(errorMessage(error, fr.discover.swipeFailed), "error");
      }
    },
    [queryClient, toast]
  );

  const restore = async () => {
    try {
      const { restored } = await discoverApi.restorePasses();
      toast.show(fr.discover.restored(restored), restored > 0 ? "success" : "info");
      seen.current.clear();
      setStatus("loading");
      await load();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const top = cards[0];
  return (
    <Screen>
      <View style={{ paddingHorizontal: spacing.lg, paddingTop: spacing.sm }}>
        <Text variant="title" color={colors.primary}>
          {fr.appName}
        </Text>
      </View>
      {status === "loading" ? <Loader /> : null}
      {status === "error" ? (
        <ErrorState
          title={fr.discover.errorTitle}
          onRetry={() => {
            setStatus("loading");
            void load();
          }}
        />
      ) : null}
      {status === "ready" && !top ? (
        <EmptyState
          glyph="🌙"
          title={fr.discover.emptyTitle}
          body={fr.discover.emptyBody}
          action={
            <Button
              testID="discover-restore"
              variant="secondary"
              title={fr.discover.restore}
              onPress={() => void restore()}
            />
          }
        />
      ) : null}
      {top ? (
        <View style={{ flex: 1, paddingTop: spacing.md }}>
          <SwipeDeck
            ref={deck}
            profiles={cards}
            onSwipe={(p, a) => void onSwipe(p, a)}
            onOpen={(p) => navigation.navigate("ProfileDetail", { userId: p.id, canSwipe: true })}
          />
          <DeckActions>
            <IconButton
              testID="swipe-pass"
              label={fr.discover.pass}
              glyph="✕"
              onPress={() => deck.current?.swipe("pass")}
            />
            <IconButton
              testID="swipe-info"
              label={fr.discover.details}
              glyph="i"
              size={44}
              onPress={() =>
                navigation.navigate("ProfileDetail", { userId: top.id, canSwipe: true })
              }
            />
            <IconButton
              testID="swipe-like"
              label={fr.discover.like}
              glyph="♥"
              tone="primary"
              onPress={() => deck.current?.swipe("like")}
            />
          </DeckActions>
        </View>
      ) : null}

      <Modal
        visible={!!match}
        transparent
        animationType="fade"
        onRequestClose={() => setMatch(null)}
      >
        <View
          style={{
            flex: 1,
            backgroundColor: colors.overlay,
            alignItems: "center",
            justifyContent: "center",
            padding: spacing.xl
          }}
        >
          {match ? (
            <View
              testID="match-modal"
              accessibilityViewIsModal
              style={{
                backgroundColor: colors.surface,
                borderRadius: 22,
                padding: spacing.xl,
                alignItems: "center",
                gap: spacing.md,
                width: "100%",
                maxWidth: 380
              }}
            >
              <Text variant="display" color={colors.primary} style={{ textAlign: "center" }}>
                {fr.discover.matchTitle}
              </Text>
              <Avatar
                name={match.profile.firstName}
                photoId={match.profile.photos[0]?.id}
                size={112}
              />
              <Text style={{ textAlign: "center" }}>
                {fr.discover.matchBody(match.profile.firstName)}
              </Text>
              <Button
                testID="match-message"
                title={fr.discover.sendMessage}
                style={{ alignSelf: "stretch" }}
                onPress={() => {
                  const target = match;
                  setMatch(null);
                  if (target.result.conversationId) {
                    navigation.navigate("Chat", {
                      conversationId: target.result.conversationId,
                      name: target.profile.firstName
                    });
                  }
                }}
              />
              <Button
                testID="match-continue"
                variant="ghost"
                title={fr.discover.keepSwiping}
                onPress={() => setMatch(null)}
              />
            </View>
          ) : null}
        </View>
      </Modal>
    </Screen>
  );
}
