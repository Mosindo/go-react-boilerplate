import { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { discoverApi } from "../api";
import type { Card } from "../api/types";
import { useToast } from "../components/feedback";
import { SwipeDeck, type SwipeAction } from "../components/SwipeDeck";
import { Badge, Button, EmptyState, ErrorState, Loading, Screen, Text } from "../components/ui";
import { ApiError, errorMessage } from "../lib/errors";
import { keys, useNotifications } from "../state/queries";
import { spacing, useColors } from "../theme";

const REFILL_THRESHOLD = 3;

export function DiscoverScreen() {
  const navigation = useNavigation();
  const colors = useColors();
  const toast = useToast();
  const queryClient = useQueryClient();
  const notifications = useNotifications();

  const [cards, setCards] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const cardsRef = useRef<Card[]>([]);
  const fetching = useRef(false);
  const [swiping, setSwiping] = useState(false);

  const update = useCallback((next: Card[]) => {
    cardsRef.current = next;
    setCards(next);
  }, []);

  const load = useCallback(async () => {
    if (fetching.current) {
      return;
    }
    fetching.current = true;
    try {
      const items = await discoverApi.feed(10);
      const known = new Set(cardsRef.current.map((c) => c.userId));
      const fresh = items.filter((item) => !known.has(item.userId));
      setExhausted(fresh.length === 0);
      setError(null);
      update([...cardsRef.current, ...fresh]);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      fetching.current = false;
      setLoading(false);
    }
  }, [update]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleSwipe = async (card: Card, action: SwipeAction) => {
    setSwiping(true);
    // Mise a jour optimiste: la carte disparait tout de suite.
    update(cardsRef.current.filter((c) => c.userId !== card.userId));
    if (cardsRef.current.length <= REFILL_THRESHOLD && !exhausted) {
      void load();
    }
    try {
      const result = action === "like" ? await discoverApi.like(card.userId) : await discoverApi.pass(card.userId);
      if (result.matched && result.matchId && result.conversationId) {
        void queryClient.invalidateQueries({ queryKey: keys.matches });
        void queryClient.invalidateQueries({ queryKey: keys.notifications });
        navigation.navigate("MatchModal", {
          matchId: result.matchId,
          conversationId: result.conversationId,
          userId: card.userId,
          firstName: card.firstName,
          photoId: card.photoIds[0],
        });
      }
    } catch (e) {
      // 409 = deja traite, 404 = n'est plus eligible: la carte reste retiree.
      if (!(e instanceof ApiError) || (e.status !== 409 && e.status !== 404)) {
        update([card, ...cardsRef.current]);
        toast.show(errorMessage(e), "error");
      }
    } finally {
      setSwiping(false);
    }
  };

  const unread = notifications.data?.unreadCount ?? 0;

  const refresh = () => {
    setLoading(true);
    setExhausted(false);
    update([]);
    void load();
  };

  let body;
  if (loading && cards.length === 0) {
    body = <Loading label="Recherche de profils…" />;
  } else if (error && cards.length === 0) {
    body = <ErrorState message={error} onRetry={refresh} />;
  } else if (cards.length === 0) {
    body = (
      <EmptyState
        action={<Button onPress={refresh} title="Actualiser" variant="secondary" />}
        icon="compass-outline"
        message="Vous avez vu tous les profils pour l'instant. Élargissez vos préférences ou revenez plus tard."
        title="Plus personne à découvrir"
      />
    );
  } else {
    body = (
      <SwipeDeck
        cards={cards}
        disabled={swiping}
        onOpen={(card) => navigation.navigate("ProfileDetail", { userId: card.userId, canSwipe: true })}
        onSwipe={(card, action) => void handleSwipe(card, action)}
      />
    );
  }

  return (
    <Screen edges={["top"]} padded={false}>
      <View
        style={{
          flexDirection: "row",
          alignItems: "center",
          justifyContent: "space-between",
          paddingHorizontal: spacing.lg,
          paddingTop: spacing.sm,
        }}
      >
        <Text style={{ color: colors.primary }} variant="title">
          Alma
        </Text>
        <Pressable
          accessibilityLabel={`Notifications${unread ? `, ${unread} non lues` : ""}`}
          accessibilityRole="button"
          onPress={() => navigation.navigate("Notifications")}
        >
          <Ionicons color={colors.text} name="notifications-outline" size={26} />
          <View style={{ position: "absolute", top: -6, right: -8 }}>
            <Badge count={unread} />
          </View>
        </Pressable>
      </View>
      {body}
    </Screen>
  );
}
