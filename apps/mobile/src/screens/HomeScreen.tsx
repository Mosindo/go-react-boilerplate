import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { PublicProfile } from "../api/platform";
import { MatchModal } from "../components/MatchModal";
import { Screen } from "../components/Screen";
import { SwipeCard, type SwipeCardHandle } from "../components/SwipeCard";
import { useDiscoveryQueue } from "../hooks/useDiscoveryQueue";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Text, colors, radii, shadows, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<RootStackParamList>;

function ActionButton({ accent, glyph, label, onPress, testID }: { accent: string; glyph: string; label: string; onPress: () => void; testID: string }) {
  return (
    <Pressable accessibilityLabel={label} accessibilityRole="button" onPress={onPress} style={styles.action} testID={testID}>
      <Text style={{ color: accent, fontSize: 26, lineHeight: 30 }} weight="bold">
        {glyph}
      </Text>
    </Pressable>
  );
}

/** Discover tab: a stack of cards with like / pass by gesture or by button. */
export default function HomeScreen() {
  const navigation = useNavigation<Nav>();
  const { decide, dismissError, error, exhausted, incomplete, load, loading, queue } = useDiscoveryQueue();
  const [match, setMatch] = useState<{ profile: PublicProfile; matchId: string } | null>(null);
  const top = queue[0];
  const next = queue[1];
  const cardRef = useRef<SwipeCardHandle>(null);

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    // Refresh after a pause (e.g. coming back from preferences) once the queue ran dry.
    return navigation.addListener("focus", () => {
      if (exhausted) {
        void load(true);
      }
    });
  }, [exhausted, load, navigation]);

  const onSwiped = useCallback(
    async (profile: PublicProfile, action: "like" | "pass") => {
      const result = await decide(profile, action);
      if (result?.matched && result.matchId) {
        setMatch({ profile: result.profile ?? profile, matchId: result.matchId });
      }
    },
    [decide]
  );

  useEffect(() => {
    if (error && queue.length > 0) {
      showToast(error, { tone: "error" });
      dismissError();
    }
  }, [dismissError, error, queue.length]);

  let body: React.ReactNode;
  if (!top && loading) {
    body = <LoadingView label="Finding people for you…" />;
  } else if (!top && incomplete) {
    body = <EmptyView message="Add at least one photo to your profile to start meeting people." title="Almost there" />;
  } else if (!top && error) {
    body = <ErrorView message={error} onAction={() => void load(true)} />;
  } else if (!top) {
    body = (
      <EmptyView
        actionLabel="Look again"
        message="You have seen everyone who matches your preferences nearby. Widen your distance or age range, or check back later."
        onAction={() => void load(true)}
        testID="discover-empty"
        title="That's everyone for now"
      />
    );
  } else {
    body = (
      <>
        <View style={styles.deck}>
          {next ? <SwipeCard interactive={false} key={next.id} onOpen={() => undefined} onSwiped={() => undefined} profile={next} /> : null}
          <SwipeCard
            interactive
            key={top.id}
            onOpen={() => navigation.navigate("ProfileDetail", { userId: top.id, name: top.firstName })}
            onSwiped={(action) => void onSwiped(top, action)}
            profile={top}
            ref={cardRef}
          />
        </View>
        <View style={styles.actions}>
          <ActionButton accent={colors.danger} glyph="✕" label={`Pass on ${top.firstName}`} onPress={() => cardRef.current?.swipe("pass")} testID="discover-pass" />
          <ActionButton accent={colors.success} glyph="♥" label={`Like ${top.firstName}`} onPress={() => cardRef.current?.swipe("like")} testID="discover-like" />
        </View>
      </>
    );
  }

  return (
    <Screen edges={["top", "right", "left"]} scroll={false} testID="discover-screen">
      <Text tone="primary" variant="heading" weight="bold">
        amora
      </Text>
      <View style={styles.stage}>{body}</View>
      <MatchModal
        onClose={() => setMatch(null)}
        onMessage={() => {
          const current = match;
          setMatch(null);
          if (current) {
            navigation.navigate("Chat", { matchId: current.matchId, userId: current.profile.id, name: current.profile.firstName });
          }
        }}
        profile={match?.profile ?? null}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  stage: { flex: 1, gap: spacing.lg, justifyContent: "center" },
  deck: { flex: 1, marginBottom: spacing.xs },
  actions: { flexDirection: "row", justifyContent: "center", gap: spacing.xxl, paddingBottom: spacing.sm },
  action: { width: 64, height: 64, borderRadius: radii.pill, backgroundColor: colors.backgroundElevated, alignItems: "center", justifyContent: "center", ...shadows.card }
});
