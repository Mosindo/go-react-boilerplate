import React, { useCallback, useRef, useState } from "react";
import { RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { MatchSummary, SwipeAction } from "../api/types";
import { IconButton, PhotoImage } from "../shared/ui";
import { ProfileSheet } from "../components/ProfileSheet";
import { SwipeCard, type SwipeCardHandle } from "../components/SwipeCard";
import { useMe } from "../hooks/useAuth";
import { useDeck } from "../hooks/useDeck";
import type { MainStackParamList, TabParamList } from "../navigation/types";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { EmptyView, ErrorView, Skeleton } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { BRAND, radius, spacing, useTheme } from "../theme";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, "Discover">,
  NativeStackScreenProps<MainStackParamList>
>;

function PeekCard({ photoId, url, position }: { photoId: string; url: string; position: number }) {
  return (
    <View pointerEvents="none" style={styles.peek}>
      <PhotoImage photo={{ id: photoId, url, position }} style={StyleSheet.absoluteFill} />
    </View>
  );
}

export default function DiscoverScreen({ navigation }: Props) {
  const theme = useTheme();
  const me = useMe().data;
  const topRef = useRef<SwipeCardHandle>(null);
  const [profileOpen, setProfileOpen] = useState(false);

  const onMatch = useCallback(
    (match: MatchSummary) => navigation.navigate("MatchCelebration", { match }),
    [navigation]
  );
  const { state, decide, refresh, discardTop } = useDeck(true, onMatch);
  const top = state.cards[0];
  const second = state.cards[1];
  const myInterestIds = me?.profile?.interests.map((interest) => interest.id) ?? [];

  const swipeTop = (action: SwipeAction) => topRef.current?.swipe(action);

  if (!top) {
    const refreshControl = <RefreshControl onRefresh={refresh} refreshing={state.loading} tintColor={theme.primary} />;
    return (
      <SafeAreaLayout edges={["top"]}>
        <ScrollView contentContainerStyle={styles.emptyScroll} refreshControl={refreshControl}>
          <View style={styles.headerPad}>
            <Header title={BRAND.name} />
          </View>
          {state.loading || (!state.exhausted && !state.loadError) ? (
            <View style={styles.skeletonWrap}>
              <Skeleton height={420} rounded={radius.lg} />
            </View>
          ) : state.loadError ? (
            <ErrorView message="We could not load new people. Check your connection." onRetry={refresh} />
          ) : (
            <EmptyView
              actionLabel="Adjust preferences"
              message="You have seen everyone nearby for now. Widen your age range or distance, or check back soon. New people join every day."
              onAction={() => navigation.navigate("Preferences")}
              title="You're all caught up"
            />
          )}
        </ScrollView>
      </SafeAreaLayout>
    );
  }

  return (
    <SafeAreaLayout edges={["top"]}>
      <View style={styles.headerPad}>
        <Header title={BRAND.name} />
      </View>
      <View style={styles.deck}>
        {second?.photos[0] ? (
          <PeekCard photoId={second.photos[0].id} position={second.photos[0].position} url={second.photos[0].url} />
        ) : null}
        <SwipeCard
          candidate={top}
          key={top.userId}
          onOpenProfile={() => setProfileOpen(true)}
          onSwiped={(action) => void decide(action)}
          ref={topRef}
        />
      </View>
      <View style={styles.actions}>
        <IconButton glyph="✕" label={`Pass on ${top.firstName}`} onPress={() => swipeTop("pass")} size={64} />
        <IconButton glyph="i" label={`View ${top.firstName}'s profile`} onPress={() => setProfileOpen(true)} />
        <IconButton filled glyph="♥" label={`Like ${top.firstName}`} onPress={() => swipeTop("like")} size={64} />
      </View>
      {profileOpen ? (
        <ProfileSheet
          candidate={top}
          key={top.userId}
          myInterestIds={myInterestIds}
          onBlocked={discardTop}
          onClose={() => setProfileOpen(false)}
          onDecide={swipeTop}
        />
      ) : null}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  headerPad: { paddingHorizontal: spacing.lg, paddingTop: spacing.sm },
  deck: { flex: 1, marginHorizontal: spacing.lg, marginVertical: spacing.md },
  peek: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radius.lg,
    overflow: "hidden",
    transform: [{ scale: 0.95 }],
    opacity: 0.8
  },
  actions: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.xl,
    paddingBottom: spacing.lg
  },
  emptyScroll: { flexGrow: 1 },
  skeletonWrap: { padding: spacing.lg }
});
