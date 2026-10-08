import React, { useCallback } from "react";
import { StyleSheet, View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { describeError } from "../api/client";
import type { PublicProfile, SwipeAction } from "../api/types";
import { SwipeDeck } from "../components/SwipeDeck";
import { useMe, useNotificationsUnread, useSavePreferences, useSwipe } from "../hooks/data";
import { useDeck } from "../hooks/useDeck";
import { useRealtime } from "../hooks/useRealtime";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Badge, Button, IconButton, Notice, spacing } from "../shared/ui";
import { markSwiped, unmarkSwiped } from "../state/swiped";

export default function DiscoverScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { data: me } = useMe();
  const unread = useNotificationsUnread();
  const swipe = useSwipe();
  const savePreferences = useSavePreferences();
  const { announceMatch } = useRealtime();
  const deck = useDeck(true);

  const paused = me ? !me.preferences.isVisible : false;

  // Coming back from Preferences or a profile: top up an empty deck so new filters apply.
  useFocusEffect(
    useCallback(() => {
      if (deck.cards.length === 0) void deck.reload();
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [deck.cards.length === 0])
  );

  const handleSwipe = async (card: PublicProfile, action: SwipeAction) => {
    deck.remove(card.id);
    markSwiped(card.id);
    try {
      const result = await swipe.mutateAsync({ userId: card.id, action });
      if (result.matched && !result.alreadyProcessed && result.matchId && result.conversationId) {
        announceMatch({ matchId: result.matchId, conversationId: result.conversationId, userId: card.id });
      }
    } catch (error) {
      unmarkSwiped(card.id);
      deck.restore(card);
      showToast(describeError(error), "error");
    }
  };

  const resume = async () => {
    if (!me) return;
    try {
      await savePreferences.mutateAsync({ ...me.preferences, isVisible: true });
      showToast("Votre profil est de nouveau visible.", "success");
    } catch (error) {
      showToast(describeError(error), "error");
    }
  };

  return (
    <SafeAreaLayout edges={["top"]}>
      <Header
        right={
          <View style={styles.headerActions}>
            <View>
              <IconButton icon="notifications-outline" label="Notifications" onPress={() => navigation.navigate("Notifications")} />
              {(unread.data ?? 0) > 0 ? (
                <View pointerEvents="none" style={styles.badge}>
                  <Badge count={unread.data ?? 0} />
                </View>
              ) : null}
            </View>
            <IconButton icon="options-outline" label="Préférences de rencontre" onPress={() => navigation.navigate("Preferences")} />
          </View>
        }
        title="Découvrir"
      />
      <View style={styles.body}>
        {paused ? (
          <View style={styles.banner}>
            <Notice message="Votre profil est en pause : personne ne peut vous voir dans la découverte." />
            <Button loading={savePreferences.isPending} onPress={resume} size="sm" variant="secondary">
              Réactiver mon profil
            </Button>
          </View>
        ) : null}

        {deck.status === "loading" ? (
          <LoadingView label="Recherche de profils…" />
        ) : deck.status === "error" ? (
          <ErrorView message={deck.error ?? "Impossible de charger les profils."} onAction={() => void deck.reload()} />
        ) : deck.cards.length === 0 ? (
          <EmptyView
            actionLabel="Élargir mes préférences"
            icon="compass-outline"
            message="Vous avez vu tous les profils qui correspondent à vos critères pour l'instant. Revenez bientôt ou élargissez votre recherche."
            onAction={() => navigation.navigate("Preferences")}
            title="C'est tout pour le moment"
          />
        ) : (
          <SwipeDeck cards={deck.cards} onOpen={(card) => navigation.navigate("ProfileDetail", { userId: card.id, canSwipe: true })} onSwipe={handleSwipe} />
        )}
        {deck.status === "ready" && deck.cards.length === 0 ? (
          <Button onPress={() => void deck.reload()} style={styles.refresh} variant="ghost">
            Actualiser
          </Button>
        ) : null}
      </View>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  body: { flex: 1, paddingHorizontal: spacing.lg, paddingBottom: spacing.sm, width: "100%", maxWidth: 560, alignSelf: "center" },
  headerActions: { flexDirection: "row", alignItems: "center" },
  badge: { position: "absolute", top: 2, right: 0 },
  banner: { gap: spacing.sm, marginBottom: spacing.md, alignItems: "flex-start" },
  refresh: { alignSelf: "center" }
});
