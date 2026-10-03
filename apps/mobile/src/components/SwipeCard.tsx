import React, { forwardRef, useCallback, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { formatDistance } from "../lib/format";
import { decideSwipe } from "../lib/swipe";
import { Text, colors, radii, spacing } from "../shared/ui";
import { PhotoImage } from "./PhotoImage";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profile: PublicProfile;
  interactive: boolean;
  onSwiped: (action: SwipeAction) => void;
  onOpen: () => void;
};

const EXIT_MS = 220;

export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ interactive, onOpen, onSwiped, profile }, ref) {
  const { width: screenWidth } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const leaving = useRef(false);
  const onSwipedRef = useRef(onSwiped);
  onSwipedRef.current = onSwiped;

  const flyOut = useCallback(
    (action: SwipeAction) => {
      if (leaving.current) {
        return;
      }
      leaving.current = true;
      Animated.timing(position, {
        toValue: { x: action === "like" ? screenWidth * 1.4 : -screenWidth * 1.4, y: 0 },
        duration: EXIT_MS,
        useNativeDriver: true
      }).start(() => onSwipedRef.current(action));
    },
    [position, screenWidth]
  );

  useImperativeHandle(ref, () => ({ swipe: flyOut }), [flyOut]);

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => interactive && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          const decision = decideSwipe(g.dx, g.vx, screenWidth);
          if (decision) {
            flyOut(decision);
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [flyOut, interactive, position, screenWidth]
  );

  const rotate = position.x.interpolate({ inputRange: [-screenWidth, 0, screenWidth], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, screenWidth * 0.3], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-screenWidth * 0.3, 0], outputRange: [1, 0], extrapolate: "clamp" });

  const count = profile.photos.length;
  const photo = profile.photos[Math.min(photoIndex, Math.max(count - 1, 0))];
  const cycle = (delta: number) => count > 1 && setPhotoIndex((i) => (i + delta + count) % count);
  const distance = formatDistance(profile.distanceKm);

  return (
    <Animated.View
      {...(interactive ? pan.panHandlers : {})}
      accessibilityLabel={`${profile.firstName}, ${profile.age}`}
      style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
      testID={interactive ? "swipe-card" : undefined}
    >
      <PhotoImage fallbackLabel={profile.firstName} path={photo?.url} style={StyleSheet.absoluteFill} />
      {count > 1 ? (
        <View pointerEvents="none" style={styles.dots}>
          {profile.photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, i === photoIndex ? styles.dotActive : null]} />
          ))}
        </View>
      ) : null}
      {interactive ? (
        <>
          <Pressable accessibilityLabel="Previous photo" onPress={() => cycle(-1)} style={[styles.zone, styles.zoneLeft]} />
          <Pressable accessibilityLabel="Next photo" onPress={() => cycle(1)} style={[styles.zone, styles.zoneRight]} />
        </>
      ) : null}
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}>
        <Text style={styles.stampText} tone="success" variant="heading" weight="bold">
          LIKE
        </Text>
      </Animated.View>
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}>
        <Text style={styles.stampText} tone="danger" variant="heading" weight="bold">
          PASS
        </Text>
      </Animated.View>
      <Pressable accessibilityHint="Opens the full profile" accessibilityRole="button" onPress={onOpen} style={styles.info} testID="swipe-card-info">
        <Text tone="inverse" variant="title" weight="bold">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.city || distance ? (
          <Text style={styles.infoSub} tone="inverse">
            {[profile.city, distance].filter(Boolean).join(" · ")}
          </Text>
        ) : null}
        {profile.bio ? (
          <Text numberOfLines={2} style={styles.infoSub} tone="inverse">
            {profile.bio}
          </Text>
        ) : null}
      </Pressable>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radii.xl, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  dots: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: spacing.xs },
  dot: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.45)" },
  dotActive: { backgroundColor: "#ffffff" },
  zone: { position: "absolute", top: 0, bottom: 140, width: "40%" },
  zoneLeft: { left: 0 },
  zoneRight: { right: 0 },
  stamp: { position: "absolute", top: spacing.xxl, paddingHorizontal: spacing.md, paddingVertical: spacing.xs, borderWidth: 3, borderRadius: radii.sm, backgroundColor: "rgba(255,255,255,0.85)" },
  stampLike: { left: spacing.xl, borderColor: colors.success, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: spacing.xl, borderColor: colors.danger, transform: [{ rotate: "12deg" }] },
  stampText: { letterSpacing: 2 },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, paddingTop: spacing.xxl, gap: spacing.xs, backgroundColor: "rgba(30,16,22,0.55)" },
  infoSub: { opacity: 0.92 }
});
