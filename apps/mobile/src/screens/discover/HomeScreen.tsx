import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useRef } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { ApiError } from "../../api/client";
import { fetchDiscover, resetPasses } from "../../api/discovery";
import type { PublicProfile, SwipeAction } from "../../api/types";
import { BellButton } from "../../components/BellButton";
import { SwipeCard, type SwipeCardHandle } from "../../components/SwipeCard";
import { keys } from "../../hooks/queries";
import { isSwipeInFlight, useSwipe } from "../../hooks/useSwipe";
import type { RootStackParamList } from "../../navigation/types";
import { EmptyView, ErrorView, LoadingView, showError } from "../../shared/feedback";
import { Header, ScreenContainer } from "../../shared/layout";
import { Button } from "../../shared/ui";
import { MIN_TOUCH, spacing, useTheme } from "../../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

/** Discovery deck: one profile at a time, with pass / like buttons and swipe gestures. */
export function HomeScreen() {
  const theme = useTheme();
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const swipe = useSwipe();
  const cardRef = useRef<SwipeCardHandle>(null);

  const deckQuery = useQuery({
    queryKey: keys.discover,
    queryFn: () => fetchDiscover(10),
    staleTime: 60_000,
    select: (items) => items.filter((p) => !isSwipeInFlight(p.userId))
  });
  const deck = deckQuery.data ?? [];
  const top = deck[0];

  const decide = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      const result = await swipe(profile, action);
      if (result?.matched && result.match) navigation.navigate("Match", { match: result.match });
    },
    [swipe, navigation]
  );

  const reviewPasses = async () => {
    try {
      await resetPasses();
      await queryClient.invalidateQueries({ queryKey: keys.discover });
    } catch (err) {
      showError(err instanceof Error ? err.message : "Action impossible.");
    }
  };

  let body: React.ReactNode;
  if (deckQuery.isPending) {
    body = <LoadingView label="Recherche de profils…" />;
  } else if (deckQuery.isError && deck.length === 0) {
    const needsProfile = deckQuery.error instanceof ApiError && deckQuery.error.code === "profile_incomplete";
    body = (
      <ErrorView
        title={needsProfile ? "Profil à compléter" : undefined}
        message={deckQuery.error instanceof Error ? deckQuery.error.message : undefined}
        onRetry={() => void deckQuery.refetch()}
      />
    );
  } else if (!top) {
    body = (
      <EmptyView
        testID="discover-empty"
        icon="compass-outline"
        title="Vous avez fait le tour pour l'instant"
        message="Revenez plus tard, élargissez vos préférences, ou revoyez les profils que vous avez passés."
        actionLabel="Élargir mes préférences"
        onAction={() => navigation.navigate("Preferences")}
      />
    );
  } else {
    body = (
      <>
        <View style={styles.deck}>
          <SwipeCard
            key={top.userId}
            ref={cardRef}
            profile={top}
            onSwiped={(action) => void decide(top, action)}
            onOpenDetails={() => navigation.navigate("ProfileDetail", { userId: top.userId, canSwipe: true })}
          />
        </View>
        <View style={styles.buttons}>
          <RoundButton
            testID="swipe-pass"
            label={`Passer ${top.firstName}`}
            icon="close"
            color={theme.textMuted}
            onPress={() => cardRef.current?.swipe("pass")}
          />
          <RoundButton
            testID="swipe-like"
            label={`Liker ${top.firstName}`}
            icon="heart"
            color={theme.primary}
            big
            onPress={() => cardRef.current?.swipe("like")}
          />
        </View>
      </>
    );
  }

  return (
    <ScreenContainer edges={["top", "left", "right"]} contentMaxWidth={520} testID="discover-screen">
      <Header title="Découvrir" right={<BellButton />} />
      <View style={styles.body}>{body}</View>
      {!deckQuery.isPending && !top ? (
        <Button testID="review-passes" label="Revoir les profils que j'ai passés" variant="ghost" onPress={() => void reviewPasses()} />
      ) : null}
    </ScreenContainer>
  );
}

function RoundButton({
  icon,
  color,
  label,
  onPress,
  big,
  testID
}: {
  icon: "close" | "heart";
  color: string;
  label: string;
  onPress: () => void;
  big?: boolean;
  testID: string;
}) {
  const theme = useTheme();
  const size = big ? 72 : 60;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: theme.surface,
          borderColor: color,
          opacity: pressed ? 0.8 : 1
        }
      ]}
    >
      <Ionicons name={icon} size={big ? 34 : 28} color={color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  body: { flex: 1, gap: spacing.lg },
  deck: { flex: 1, minHeight: 320 },
  buttons: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl, paddingBottom: spacing.sm },
  round: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    shadowColor: "#000",
    shadowOpacity: 0.12,
    shadowRadius: 8,
    elevation: 3,
    minWidth: MIN_TOUCH
  }
});
