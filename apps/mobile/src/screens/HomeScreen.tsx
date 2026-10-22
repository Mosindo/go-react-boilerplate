import React, { useCallback, useRef } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { swipe } from "../api/discovery";
import type { PublicProfile, SwipeAction } from "../api/types";
import { SwipeDeck, type SwipeDeckHandle } from "../components/SwipeDeck";
import { useDiscoverQueue } from "../hooks/useDiscoverQueue";
import { useRealtime } from "../hooks/useRealtime";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, showGlobalError } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Loader, Text, colors, radii, shadows, spacing } from "../shared/ui";

/** Discover tab: one profile at a time, swipe or tap to like / pass. */
export default function HomeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { queue, status, error, dequeue, requeue, refresh } = useDiscoverQueue();
  const { announceMatch } = useRealtime();
  const deck = useRef<SwipeDeckHandle>(null);

  const handleSwipe = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      dequeue(profile.userId);
      try {
        const result = await swipe(profile.userId, action);
        if (result.matched && !result.alreadySwiped && result.match) {
          announceMatch(result.match);
        }
      } catch (err) {
        requeue(profile);
        showGlobalError(errorMessage(err, "Impossible d'enregistrer votre choix."));
      }
    },
    [announceMatch, dequeue, requeue]
  );

  const hasCard = queue.length > 0;

  return (
    <ScreenContainer contentMaxWidth={560} edges={["top", "left", "right"]} testID="discover-screen">
      <View style={styles.header}>
        <Text accessibilityRole="header" variant="heading" weight="bold">
          Découvrir
        </Text>
        <Pressable accessibilityRole="button" onPress={() => navigation.navigate("Preferences")} testID="open-preferences">
          <Text tone="primary" variant="label" weight="bold">
            Filtres
          </Text>
        </Pressable>
      </View>

      <View style={styles.stage}>
        {hasCard ? (
          <SwipeDeck
            onOpen={(profile) => navigation.navigate("ProfileDetail", { userId: profile.userId })}
            onSwipe={handleSwipe}
            profiles={queue}
            ref={deck}
          />
        ) : status === "loading" ? (
          <Loader fullScreen label="Recherche de profils…" />
        ) : status === "error" ? (
          <ErrorView
            actionLabel="Réessayer"
            message={error ?? "Chargement impossible."}
            onAction={refresh}
            title="Impossible de charger les profils"
          />
        ) : (
          <EmptyView
            actionLabel="Actualiser"
            message="Élargissez vos filtres d'âge ou de distance, ou revenez un peu plus tard : de nouvelles personnes arrivent chaque jour."
            onAction={refresh}
            testID="discover-empty"
            title="Vous avez fait le tour pour le moment"
          />
        )}
      </View>

      {hasCard ? (
        <View style={styles.actions}>
          <Pressable
            accessibilityLabel="Passer"
            accessibilityRole="button"
            onPress={() => deck.current?.swipe("pass")}
            style={({ pressed }) => [styles.round, pressed ? styles.pressed : null]}
            testID="pass-button"
          >
            <Text style={[styles.glyph, { color: colors.pass }]} weight="bold">
              ✕
            </Text>
          </Pressable>
          <Pressable
            accessibilityLabel="J'aime"
            accessibilityRole="button"
            onPress={() => deck.current?.swipe("like")}
            style={({ pressed }) => [styles.round, styles.roundLike, pressed ? styles.pressed : null]}
            testID="like-button"
          >
            <Text style={[styles.glyph, { color: colors.inverse }]} weight="bold">
              ♥
            </Text>
          </Pressable>
        </View>
      ) : null}
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.md },
  stage: { flex: 1, justifyContent: "center" },
  actions: { flexDirection: "row", justifyContent: "center", gap: spacing.xl, paddingTop: spacing.lg },
  round: {
    width: 64,
    height: 64,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.backgroundElevated,
    borderWidth: 1,
    borderColor: colors.border,
    ...shadows.card
  },
  roundLike: { backgroundColor: colors.primary, borderColor: colors.primary },
  pressed: { transform: [{ scale: 0.94 }] },
  glyph: { fontSize: 26, lineHeight: 30 }
});
