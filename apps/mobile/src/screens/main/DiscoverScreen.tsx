import React, { useEffect, useRef } from "react";
import { View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { Card } from "../../api/types";
import { useDiscoverState } from "../../features/discoverStore";
import { MatchModal } from "../../features/MatchModal";
import { SwipeDeck, type SwipeDeckHandle } from "../../features/SwipeDeck";
import { useSwipe } from "../../features/useSwipe";
import { t } from "../../i18n/fr";
import { remainingDeck, shouldPrefetch } from "../../lib/swipe";
import { useDiscover } from "../../queries";
import { createStyles, useTheme } from "../../theme/ThemeProvider";
import { EmptyState, ErrorState, ScalePressable, Screen, Skeleton, Text } from "../../ui";
import type { MainStackParams, TabParams } from "../navTypes";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Discover">,
  NativeStackScreenProps<MainStackParams>
>;

const useStyles = createStyles(({ colors, spacing, minTouch }) => ({
  deck: { flex: 1, paddingTop: spacing.md },
  buttons: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xl,
    paddingVertical: spacing.lg
  },
  round: {
    width: minTouch + 20,
    height: minTouch + 20,
    borderRadius: (minTouch + 20) / 2,
    backgroundColor: colors.surface,
    borderWidth: 2,
    alignItems: "center",
    justifyContent: "center"
  }
}));

export default function DiscoverScreen({ navigation }: Props) {
  const styles = useStyles();
  const discover = useDiscover();
  const { swiped } = useDiscoverState();
  const { decide } = useSwipe();
  const deckRef = useRef<SwipeDeckHandle>(null);
  const fetchedAtSwipeCount = useRef(0);

  const cards = discover.data ?? [];
  const remaining = remainingDeck(cards, swiped);

  // Prefetch the next page once the deck runs low (only after new swipes, to avoid refetch loops).
  useEffect(() => {
    if (!discover.data || discover.isFetching) return;
    if (
      discover.data.length > 0 &&
      shouldPrefetch(remaining.length) &&
      swiped.size > fetchedAtSwipeCount.current
    ) {
      fetchedAtSwipeCount.current = swiped.size;
      void discover.refetch();
    }
  }, [discover, remaining.length, swiped.size]);

  const open = (card: Card) =>
    navigation.navigate("ProfileDetail", { userId: card.userId, card, canSwipe: true });

  const top = remaining[0];

  let content: React.ReactNode;
  if (discover.isPending) {
    content = (
      <View style={styles.deck}>
        <Skeleton height={400} radius={24} />
      </View>
    );
  } else if (discover.isError && remaining.length === 0) {
    content = <ErrorState onRetry={() => discover.refetch()} title={t.discover.errorTitle} />;
  } else if (remaining.length === 0) {
    content = (
      <EmptyState
        actionLabel={t.common.refresh}
        glyph="☼"
        message={t.discover.emptyText}
        onAction={() => {
          fetchedAtSwipeCount.current = swiped.size;
          void discover.refetch();
        }}
        testID="discover-empty"
        title={t.discover.emptyTitle}
      />
    );
  } else {
    content = (
      <>
        <View style={styles.deck}>
          <SwipeDeck cards={remaining} onDecision={decide} onOpen={open} ref={deckRef} />
        </View>
        <View style={styles.buttons}>
          <RoundButton
            glyph="✕"
            label={t.discover.passLabel(top.firstName)}
            onPress={() => deckRef.current?.fling("pass")}
            testID="discover-pass-button"
            tone="pass"
          />
          <RoundButton
            glyph="♥"
            label={t.discover.likeLabel(top.firstName)}
            onPress={() => deckRef.current?.fling("like")}
            testID="discover-like-button"
            tone="like"
          />
        </View>
      </>
    );
  }

  return (
    <Screen padded testID="discover-screen">
      {content}
      <MatchModal
        onSendMessage={(matchId, userId, firstName, thumbUrl) =>
          navigation.navigate("Chat", { matchId, userId, firstName, thumbUrl })
        }
      />
    </Screen>
  );
}

export function RoundButton({
  glyph,
  label,
  onPress,
  tone,
  testID
}: {
  glyph: string;
  label: string;
  onPress: () => void;
  tone: "like" | "pass";
  testID: string;
}) {
  const styles = useStyles();
  const { colors } = useTheme();
  const color = tone === "like" ? "accent" : "muted";
  return (
    <ScalePressable
      accessibilityLabel={label}
      accessibilityRole="button"
      onPress={onPress}
      testID={testID}
    >
      <View style={[styles.round, { borderColor: tone === "like" ? colors.like : colors.pass }]}>
        <Text tone={color} variant="title">
          {glyph}
        </Text>
      </View>
    </ScalePressable>
  );
}
