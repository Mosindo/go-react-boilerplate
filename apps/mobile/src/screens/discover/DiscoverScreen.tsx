import React, { useCallback, useEffect, useRef } from "react";
import { StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { SwipeResult } from "../../api/types";
import { useCelebrateMatch } from "../../components/MatchCelebration";
import { useUnreadCounts } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { useDeck } from "../../hooks/useDeck";
import type { AppStackParamList, TabParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Badge, EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, Text } from "../../ui";
import { SwipeCard, type SwipeCardHandle } from "./SwipeCard";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Discover">, NativeStackScreenProps<AppStackParamList>>;

export function NotificationsBell({ onPress }: { onPress: () => void }) {
  const { notifications } = useUnreadCounts();
  return (
    <View>
      <IconButton icon="notifications-outline" label="Notifications" onPress={onPress} testID="open-notifications" />
      {notifications > 0 ? (
        <View pointerEvents="none" style={styles.bellBadge}>
          <Badge count={notifications} />
        </View>
      ) : null}
    </View>
  );
}

export function DiscoverScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const celebrate = useCelebrateMatch();
  const onMatch = useCallback((result: SwipeResult) => result.match && celebrate(result.match), [celebrate]);
  const deck = useDeck(onMatch);
  const topCard = useRef<SwipeCardHandle>(null);
  const [top, second] = deck.queue;

  // New search criteria or location: start a fresh deck.
  const { profile } = useAuth();
  const criteria = JSON.stringify([profile?.preferences, profile?.hasLocation]);
  const lastCriteria = useRef(criteria);
  const { reset } = deck;
  useEffect(() => {
    if (lastCriteria.current !== criteria) {
      lastCriteria.current = criteria;
      reset();
    }
  }, [criteria, reset]);

  let content: React.ReactNode;
  if (deck.loading) {
    content = <LoadingState label="Recherche de profils…" />;
  } else if (!top && deck.error) {
    content = <ErrorState message={deck.error} onRetry={deck.reset} />;
  } else if (!top) {
    content =
      deck.reason === "no_more_profiles_location_off" ? (
        <EmptyState
          actionLabel="Ouvrir les réglages"
          icon="location-outline"
          message="Activez la localisation approximative pour découvrir des personnes près de chez vous."
          onAction={() => navigation.navigate("Settings")}
          testID="deck-empty"
          title="Plus personne pour l'instant"
        />
      ) : (
        <EmptyState
          actionLabel="Élargir mes préférences"
          icon="sparkles-outline"
          message="Vous avez vu tous les profils correspondant à vos critères. Revenez plus tard ou élargissez l'âge et la distance."
          onAction={() => navigation.navigate("Preferences")}
          testID="deck-empty"
          title="Vous avez fait le tour"
        />
      );
  } else {
    content = (
      <>
        <View style={styles.deck}>
          {second ? (
            <View pointerEvents="none" style={[StyleSheet.absoluteFill, styles.behind]}>
              <SwipeCard active={false} onOpen={() => undefined} onSwiped={() => undefined} profile={second} />
            </View>
          ) : null}
          <SwipeCard
            active
            key={top.userId}
            onOpen={() => navigation.navigate("ProfileDetail", { userId: top.userId, source: "discover" })}
            onSwiped={(action) => deck.decide(top, action)}
            profile={top}
            ref={topCard}
          />
        </View>
        <View style={styles.actions}>
          <IconButton
            color={colors.textMuted}
            icon="close"
            label={`Passer ${top.firstName}`}
            onPress={() => topCard.current?.swipe("pass")}
            size={64}
            testID="deck-pass"
            variant="surface"
          />
          <IconButton
            icon="heart"
            label={`Liker ${top.firstName}`}
            onPress={() => topCard.current?.swipe("like")}
            size={76}
            testID="deck-like"
            variant="primary"
          />
          <IconButton
            color={colors.accent}
            icon="person-outline"
            label={`Voir le profil de ${top.firstName}`}
            onPress={() => navigation.navigate("ProfileDetail", { userId: top.userId, source: "discover" })}
            size={64}
            testID="deck-open"
            variant="surface"
          />
        </View>
      </>
    );
  }

  return (
    <Screen testID="discover-screen">
      <Header
        large
        right={
          <>
            <IconButton icon="options-outline" label="Préférences de recherche" onPress={() => navigation.navigate("Preferences")} />
            <NotificationsBell onPress={() => navigation.navigate("Notifications")} />
          </>
        }
      >
        <Text accessibilityRole="header" variant="title">
          Découvrir
        </Text>
      </Header>
      {content}
    </Screen>
  );
}

const styles = StyleSheet.create({
  deck: { flex: 1, marginTop: spacing.xs },
  behind: { transform: [{ scale: 0.95 }, { translateY: 14 }], opacity: 0.7 },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.xl, paddingVertical: spacing.lg },
  bellBadge: { position: "absolute", top: 2, right: 0 }
});
