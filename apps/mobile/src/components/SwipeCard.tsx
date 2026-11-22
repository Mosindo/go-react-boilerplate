import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { ProfileCard, SwipeAction } from "../api/types";
import { formatDistance } from "../lib/format";
import { Chip, Text, colors, radii, shadows, spacing } from "../ui";
import { PhotoCarousel } from "./PhotoCarousel";

export type SwipeCardHandle = {
  swipe: (action: SwipeAction) => void;
};

type Props = {
  profile: ProfileCard;
  sharedInterests: Set<string>;
  onSwiped: (action: SwipeAction) => void;
  onOpen: () => void;
};

const THRESHOLD = 110;

export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ profile, sharedInterests, onSwiped, onOpen }, ref) {
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const done = useRef(false);

  const fling = (action: SwipeAction) => {
    if (done.current) {
      return;
    }
    done.current = true;
    // The decision must never depend on the animation finishing: animation
    // frames can be throttled (background tab, low-power mode) or interrupted.
    let reported = false;
    const report = () => {
      if (!reported) {
        reported = true;
        onSwiped(action);
      }
    };
    const fallback = setTimeout(report, 400);
    Animated.timing(position, {
      toValue: { x: (action === "like" ? 1 : -1) * width * 1.4, y: 40 },
      duration: 240,
      useNativeDriver: false
    }).start(() => {
      clearTimeout(fallback);
      report();
    });
  };

  useImperativeHandle(ref, () => ({ swipe: fling }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > THRESHOLD) {
            fling("like");
          } else if (g.dx < -THRESHOLD) {
            fling("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        }
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [position, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const distance = formatDistance(profile.distanceKm);

  return (
    <Animated.View
      {...responder.panHandlers}
      accessibilityHint="Faites glisser à droite pour liker, à gauche pour passer, ou utilisez les boutons."
      accessibilityLabel={`${profile.firstName}, ${profile.age} ans`}
      style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
      testID={`swipe-card-${profile.userId}`}
    >
      <PhotoCarousel name={profile.firstName} photos={profile.photos} rounded={radii.xl} />
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}>
        <Text style={{ color: colors.like }} variant="heading">
          J’AIME
        </Text>
      </Animated.View>
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}>
        <Text style={{ color: colors.pass }} variant="heading">
          PASSER
        </Text>
      </Animated.View>
      <View pointerEvents="box-none" style={styles.info}>
        <View pointerEvents="none" style={styles.scrim} />
        <Text accessibilityRole="button" onPress={onOpen} style={styles.name} testID="swipe-card-name" variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        <Text tone="muted" variant="caption">
          {[profile.jobTitle, profile.city, distance].filter(Boolean).join(" · ")}
        </Text>
        {profile.isDemo ? (
          <View style={styles.demo}>
            <Text tone="onAccent" variant="eyebrow">
              Profil de démo
            </Text>
          </View>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.slice(0, 4).map((i) => (
              <Chip highlighted={sharedInterests.has(i.slug)} key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
      </View>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    backgroundColor: colors.surfaceHigh,
    ...shadows.card
  },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    gap: spacing.xs,
    borderBottomLeftRadius: radii.xl,
    borderBottomRightRadius: radii.xl,
    overflow: "hidden"
  },
  scrim: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: colors.scrimBottom
  },
  name: {
    alignSelf: "flex-start"
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs,
    marginTop: spacing.xxs
  },
  demo: {
    alignSelf: "flex-start",
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: radii.sm,
    backgroundColor: colors.warning
  },
  stamp: {
    position: "absolute",
    top: 36,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xxs,
    borderRadius: radii.sm,
    borderWidth: 3
  },
  stampLike: {
    left: 24,
    borderColor: colors.like,
    transform: [{ rotate: "-14deg" }]
  },
  stampPass: {
    right: 24,
    borderColor: colors.pass,
    transform: [{ rotate: "14deg" }]
  }
});
