import React, { useRef } from "react";
import { StyleSheet, View, useAnimatedValue } from "react-native";
import type { PublicProfile, SwipeAction } from "../../api";
import { IconButton } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";
import { SwipeCard, type SwipeCardHandle } from "./SwipeCard";

type Props = {
  queue: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenProfile: (profile: PublicProfile) => void;
};

export function SwipeDeck({ queue, onSwipe, onOpenProfile }: Props) {
  const { colors } = useTheme();
  const dragProgress = useAnimatedValue(0);
  const topCardRef = useRef<SwipeCardHandle>(null);
  const [top, next] = queue;

  if (!top) {
    return null;
  }

  return (
    <View style={styles.wrapper}>
      <View style={styles.stack}>
        {next ? (
          <SwipeCard
            dragProgress={dragProgress}
            interactive={false}
            key={next.userId}
            onOpenProfile={() => undefined}
            onSwiped={() => undefined}
            profile={next}
          />
        ) : null}
        <SwipeCard
          dragProgress={dragProgress}
          interactive
          key={top.userId}
          onOpenProfile={() => onOpenProfile(top)}
          onSwiped={(action) => {
            dragProgress.setValue(0);
            onSwipe(top, action);
          }}
          profile={top}
          ref={topCardRef}
        />
      </View>
      <View style={styles.actions}>
        <IconButton
          color="pass"
          icon="close"
          label="Passer"
          onPress={() => topCardRef.current?.swipe("pass")}
          size={30}
          style={[styles.action, { backgroundColor: colors.surface, borderColor: colors.border }]}
          testID="deck-pass-button"
        />
        <IconButton
          color="onPrimary"
          icon="heart"
          label="J'aime"
          onPress={() => topCardRef.current?.swipe("like")}
          size={30}
          style={[styles.action, { backgroundColor: colors.primary, borderColor: colors.primary }]}
          testID="deck-like-button"
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: { flex: 1 },
  stack: { flex: 1, marginHorizontal: spacing.lg },
  actions: { flexDirection: "row", justifyContent: "center", gap: spacing.xl, paddingVertical: spacing.lg },
  action: { width: 64, height: 64, borderRadius: radius.pill, borderWidth: 1 }
});
