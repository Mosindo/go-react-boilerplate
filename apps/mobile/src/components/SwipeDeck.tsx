import React, { forwardRef, useCallback, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions, type LayoutChangeEvent } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { Text, colors, radii, spacing } from "../shared/ui";
import { PhotoCarousel } from "./PhotoCarousel";
import { ProfileSummary } from "./ProfileSummary";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type SwipeDeckProps = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpen: (profile: PublicProfile) => void;
};

const SWIPE_THRESHOLD = 110;
const FLING_VELOCITY = 0.7;
const ANIMATION_MS = 220;

export const SwipeDeck = forwardRef<SwipeDeckHandle, SwipeDeckProps>(function SwipeDeck({ profiles, onSwipe, onOpen }, ref) {
  const { width: windowWidth } = useWindowDimensions();
  const pan = useRef(new Animated.ValueXY()).current;
  const [cardWidth, setCardWidth] = useState(0);
  const top = profiles[0];
  const next = profiles[1];
  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  const flyOut = useCallback(
    (action: SwipeAction) => {
      const target = latest.current.top;
      if (!target) {
        return;
      }
      Animated.timing(pan, {
        toValue: { x: (action === "like" ? 1 : -1) * windowWidth * 1.3, y: 40 },
        duration: ANIMATION_MS,
        useNativeDriver: false
      }).start(() => {
        latest.current.onSwipe(target, action);
        pan.setValue({ x: 0, y: 0 });
      });
    },
    [pan, windowWidth]
  );

  useImperativeHandle(ref, () => ({ swipe: flyOut }), [flyOut]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_event, gesture) => Math.abs(gesture.dx) > 8 || Math.abs(gesture.dy) > 8,
        onPanResponderMove: (_event, gesture) => pan.setValue({ x: gesture.dx, y: gesture.dy * 0.4 }),
        onPanResponderRelease: (_event, gesture) => {
          if (gesture.dx > SWIPE_THRESHOLD || gesture.vx > FLING_VELOCITY) {
            flyOut("like");
          } else if (gesture.dx < -SWIPE_THRESHOLD || gesture.vx < -FLING_VELOCITY) {
            flyOut("pass");
          } else {
            Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
          }
        },
        onPanResponderTerminate: () => Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
      }),
    [flyOut, pan]
  );

  const rotate = pan.x.interpolate({ inputRange: [-windowWidth, 0, windowWidth], outputRange: ["-14deg", "0deg", "14deg"] });
  const likeOpacity = pan.x.interpolate({ inputRange: [20, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = pan.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = pan.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0, SWIPE_THRESHOLD],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  if (!top) {
    return null;
  }

  return (
    <View onLayout={(event: LayoutChangeEvent) => setCardWidth(event.nativeEvent.layout.width)} style={styles.deck}>
      {next ? (
        <Animated.View pointerEvents="none" style={[styles.card, styles.behind, { transform: [{ scale: nextScale }] }]}>
          <PhotoCarousel name={next.firstName} photos={next.photos.slice(0, 1)} style={StyleSheet.absoluteFill} />
        </Animated.View>
      ) : null}
      <Animated.View
        {...responder.panHandlers}
        accessibilityLabel={`${top.firstName}, ${top.age} ans`}
        key={top.userId}
        style={[styles.card, { width: cardWidth || undefined, transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }]}
        testID="discover-card"
      >
        <PhotoCarousel name={top.firstName} onPressCenter={() => onOpen(top)} photos={top.photos} style={StyleSheet.absoluteFill} />
        <View pointerEvents="none" style={styles.scrim} />
        <View pointerEvents="none" style={styles.info}>
          <ProfileSummary
            age={top.age}
            bio={top.bio}
            city={top.city}
            compact
            distanceKm={top.distanceKm}
            firstName={top.firstName}
            interests={top.interests}
            inverse
          />
        </View>
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}>
          <Text style={styles.stampText} tone="inverse" variant="heading" weight="bold">
            J&apos;AIME
          </Text>
        </Animated.View>
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}>
          <Text style={styles.stampText} tone="inverse" variant="heading" weight="bold">
            PASSER
          </Text>
        </Animated.View>
      </Animated.View>
    </View>
  );
});

const styles = StyleSheet.create({
  deck: { flex: 1, width: "100%" },
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    overflow: "hidden",
    backgroundColor: colors.surfaceSubtle
  },
  behind: { opacity: 0.9 },
  scrim: { position: "absolute", left: 0, right: 0, bottom: 0, height: "46%", backgroundColor: "rgba(20, 10, 16, 0.55)" },
  info: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: spacing.lg },
  stamp: {
    position: "absolute",
    top: spacing.xl,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  stampLike: { left: spacing.xl, borderColor: colors.like, backgroundColor: "rgba(47, 158, 110, 0.85)", transform: [{ rotate: "-12deg" }] },
  stampPass: {
    right: spacing.xl,
    borderColor: colors.pass,
    backgroundColor: "rgba(122, 107, 116, 0.85)",
    transform: [{ rotate: "12deg" }]
  },
  stampText: { letterSpacing: 2 }
});
