import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type MutableRefObject
} from "react";
import { Animated, Image, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile } from "../api/models";
import { STAMP_FULL_RATIO, decideSwipe, flyOffTarget, type SwipeAction } from "../lib/dating/swipe";
import { useTheme } from "../shared/ui/theme";
import { Txt, useReducedMotion } from "./kit";
import { photoUri } from "./photoUrl";
import { ProfileCard } from "./ProfileCard";

export type DeckHandle = { swipe: (action: SwipeAction) => void };

type DeckProps = {
  profiles: readonly PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenInfo: (profile: PublicProfile) => void;
  handleRef: MutableRefObject<DeckHandle | null>;
};

const VISIBLE_BEHIND = 2;

type TopCardProps = {
  profile: PublicProfile;
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenInfo: (profile: PublicProfile) => void;
  handleRef: MutableRefObject<DeckHandle | null>;
};

function TopCard({ profile, onSwipe, onOpenInfo, handleRef }: TopCardProps) {
  const { colors, radii, spacing } = useTheme();
  const { width } = useWindowDimensions();
  const reduced = useReducedMotion();
  const [pan] = useState(() => new Animated.ValueXY({ x: 0, y: 0 }));
  const finished = useRef(false);

  const complete = useCallback(
    (action: SwipeAction) => {
      if (finished.current) return;
      finished.current = true;
      if (reduced) {
        onSwipe(profile, action);
        return;
      }
      Animated.timing(pan, {
        toValue: { x: flyOffTarget(action, width), y: 0 },
        duration: 220,
        useNativeDriver: false
      }).start(() => onSwipe(profile, action));
    },
    [onSwipe, pan, profile, reduced, width]
  );

  useEffect(() => {
    handleRef.current = { swipe: complete };
    return () => {
      handleRef.current = null;
    };
  }, [complete, handleRef]);

  // The handlers only read `finished` when a gesture happens, never during render.
  const responder = useMemo(
    () =>
      // eslint-disable-next-line react-hooks/refs
      PanResponder.create({
        onMoveShouldSetPanResponderCapture: (_e, g) =>
          !finished.current && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy) * 1.2,
        onPanResponderTerminationRequest: () => false,
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (_e, g) => {
          const action = decideSwipe(g.dx, g.vx, width);
          if (action) {
            complete(action);
            return;
          }
          if (reduced) {
            pan.setValue({ x: 0, y: 0 });
            return;
          }
          Animated.spring(pan, {
            toValue: { x: 0, y: 0 },
            friction: 6,
            tension: 80,
            useNativeDriver: false
          }).start();
        },
        onPanResponderTerminate: () => {
          Animated.spring(pan, {
            toValue: { x: 0, y: 0 },
            useNativeDriver: false
          }).start();
        }
      }),
    [complete, pan, reduced, width]
  );

  const rotate = pan.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: ["-14deg", "0deg", "14deg"],
    extrapolate: "clamp"
  });
  const full = width * STAMP_FULL_RATIO;
  const likeOpacity = pan.x.interpolate({
    inputRange: [0, full],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = pan.x.interpolate({
    inputRange: [-full, 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });

  return (
    <Animated.View
      {...responder.panHandlers}
      style={[
        styles.fill,
        {
          transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }]
        }
      ]}
    >
      <ProfileCard profile={profile} onOpenInfo={onOpenInfo} testID="discover-card" />
      <Animated.View
        pointerEvents="none"
        style={[
          styles.stamp,
          styles.stampLike,
          {
            opacity: likeOpacity,
            borderColor: colors.success,
            borderRadius: radii.md,
            padding: spacing.sm
          }
        ]}
      >
        <Txt variant="title" style={{ color: colors.success }}>
          LIKE
        </Txt>
      </Animated.View>
      <Animated.View
        pointerEvents="none"
        style={[
          styles.stamp,
          styles.stampPass,
          {
            opacity: passOpacity,
            borderColor: colors.danger,
            borderRadius: radii.md,
            padding: spacing.sm
          }
        ]}
      >
        <Txt variant="title" style={{ color: colors.danger }}>
          PASS
        </Txt>
      </Animated.View>
    </Animated.View>
  );
}

function SwipeDeckBase({ profiles, onSwipe, onOpenInfo, handleRef }: DeckProps) {
  const top = profiles[0];
  const behind = profiles.slice(1, 1 + VISIBLE_BEHIND);

  // Warm the image cache for the next card.
  useEffect(() => {
    const next = profiles[1];
    const uri = photoUri(next?.photos[0]);
    if (uri) void Image.prefetch(uri).catch(() => undefined);
  }, [profiles]);

  if (!top) return null;

  return (
    <View style={styles.deck}>
      {behind
        .map((profile, i) => ({ profile, depth: i + 1 }))
        .reverse()
        .map(({ profile, depth }) => (
          <View
            key={profile.userId}
            pointerEvents="none"
            accessibilityElementsHidden
            importantForAccessibility="no-hide-descendants"
            style={[
              styles.fill,
              {
                transform: [{ scale: 1 - depth * 0.04 }, { translateY: depth * 10 }]
              }
            ]}
          >
            <ProfileCard profile={profile} interactive={false} />
          </View>
        ))}
      <TopCard
        key={top.userId}
        profile={top}
        onSwipe={onSwipe}
        onOpenInfo={onOpenInfo}
        handleRef={handleRef}
      />
    </View>
  );
}

export const SwipeDeck = React.memo(SwipeDeckBase);

const styles = StyleSheet.create({
  deck: { flex: 1 },
  fill: { ...StyleSheet.absoluteFillObject },
  stamp: { position: "absolute", top: 48, borderWidth: 4 },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] }
});
