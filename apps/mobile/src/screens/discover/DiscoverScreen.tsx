import React, { useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useNavigation } from "@react-navigation/native";
import { errorMessage } from "../../api/client";
import type { Card, MatchPayload, SwipeDirection } from "../../api/types";
import { useDiscover, useSwipe } from "../../hooks/queries";
import { useRealtime } from "../../hooks/useRealtime";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { EmptyState, ErrorState, IconButton, LoadingState, Text, useFeedback } from "../../ui";
import { MatchModal } from "./MatchModal";
import { SwipeDeck, type SwipeDeckHandle } from "./SwipeDeck";

export function DiscoverScreen() {
  const { colors, spacing } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { toast } = useFeedback();
  const feed = useDiscover(true);
  const swipe = useSwipe();
  const deck = useRef<SwipeDeckHandle>(null);
  const [match, setMatch] = useState<MatchPayload | null>(null);
  const realtime = useRealtime();
  const shownMatch = match ?? realtime.incomingMatch;

  const onSwipe = (card: Card, direction: SwipeDirection) => {
    swipe.mutate(
      { card, direction },
      {
        onSuccess: (res) => {
          if (res.matched && res.match) {
            setMatch(res.match);
          }
        },
        onError: (e) => toast(errorMessage(e), "error")
      }
    );
  };

  const closeMatch = () => {
    setMatch(null);
    realtime.dismissIncomingMatch();
  };

  const cards = feed.data ?? [];

  let body: React.ReactNode;
  if (feed.isLoading) {
    body = <LoadingState label="On cherche des profils pour toi…" />;
  } else if (feed.isError) {
    body = <ErrorState message={errorMessage(feed.error)} onRetry={() => void feed.refetch()} />;
  } else if (cards.length === 0) {
    body = (
      <EmptyState
        testID="discover-empty"
        icon="sparkles-outline"
        title="Tu as fait le tour pour l'instant"
        message="Reviens un peu plus tard, ou élargis l'âge et la distance dans tes préférences."
        actionLabel="Modifier mes préférences"
        onAction={() => navigation.navigate("EditPreferences")}
      />
    );
  } else {
    body = (
      <>
        <SwipeDeck
          ref={deck}
          cards={cards}
          onSwipe={onSwipe}
          onOpen={(card) => navigation.navigate("ProfileDetail", { userId: card.userId, card })}
        />
        <View style={[styles.actions, { gap: spacing.xl }]}>
          <IconButton
            testID="discover-pass"
            icon="close"
            label="Passer"
            size={64}
            filled
            tone="default"
            onPress={() => deck.current?.swipe("pass")}
          />
          <IconButton
            testID="discover-like"
            icon="heart"
            label="J'aime"
            size={72}
            filled
            tone="primary"
            onPress={() => deck.current?.swipe("like")}
          />
        </View>
      </>
    );
  }

  return (
    <SafeAreaView
      edges={["top"]}
      style={[styles.flex, { backgroundColor: colors.background }]}
      testID="discover-screen"
    >
      <View style={[styles.header, { paddingHorizontal: spacing.lg }]}>
        <Text variant="title">Découvrir</Text>
      </View>
      <View style={[styles.flex, styles.content, { padding: spacing.md, paddingTop: 0 }]}>{body}</View>
      <MatchModal
        match={shownMatch}
        onClose={closeMatch}
        onMessage={(m) => {
          closeMatch();
          navigation.navigate("Chat", { matchId: m.matchId, user: m.user });
        }}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { width: "100%", maxWidth: 560, alignSelf: "center" },
  header: { paddingVertical: 12 },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", paddingVertical: 16 }
});
