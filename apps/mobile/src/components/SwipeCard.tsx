import React, { useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile } from "../api/types";
import { Chip, PhotoImage, Text } from "../ui";
import { radius, spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  profile: PublicProfile;
  onSwipe: (direction: "like" | "pass") => void;
  onOpenProfile: () => void;
};

const SWIPE_THRESHOLD = 110;

export function distanceLabel(km?: number): string | null {
  if (km === undefined) return null;
  return km <= 1 ? "À moins de 1 km" : `À ${km} km`;
}

/**
 * Draggable profile card. Tapping the left/right third of the photo cycles
 * through photos; dragging past the threshold (or using the buttons below,
 * which call `onSwipe` through the ref API of the parent) like/passes.
 */
export const SwipeCard = React.forwardRef<{ fling: (d: "like" | "pass") => void }, Props>(function SwipeCard(
  { profile, onSwipe, onOpenProfile },
  ref
) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);

  const fling = React.useCallback(
    (direction: "like" | "pass") => {
      Animated.timing(position, {
        toValue: { x: direction === "like" ? width * 1.4 : -width * 1.4, y: 0 },
        duration: 220,
        useNativeDriver: false
      }).start(() => onSwipe(direction));
    },
    [onSwipe, position, width]
  );

  React.useImperativeHandle(ref, () => ({ fling }), [fling]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD) fling("like");
          else if (g.dx < -SWIPE_THRESHOLD) fling("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [fling, position]
  );

  const rotate = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: ["-12deg", "0deg", "12deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, SWIPE_THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });

  const photos = profile.photos;
  const current = photos[photoIndex];
  const distance = distanceLabel(profile.distanceKm);

  return (
    <Animated.View
      {...responder.panHandlers}
      testID="swipe-card"
      style={[
        styles.card,
        {
          backgroundColor: colors.surface,
          transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }]
        }
      ]}
    >
      <PhotoImage photoId={current?.id} name={profile.firstName} style={StyleSheet.absoluteFill} />

      <View style={styles.tapRow} pointerEvents="box-none">
        <Pressable
          style={styles.tapZone}
          accessibilityLabel="Photo précédente"
          accessibilityRole="button"
          onPress={() => setPhotoIndex((i) => Math.max(0, i - 1))}
        />
        <Pressable
          style={styles.tapZone}
          accessibilityLabel="Photo suivante"
          accessibilityRole="button"
          onPress={() => setPhotoIndex((i) => Math.min(photos.length - 1, i + 1))}
        />
      </View>

      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View
              key={p.id}
              style={[styles.dot, { backgroundColor: i === photoIndex ? "#fff" : "rgba(255,255,255,0.4)" }]}
            />
          ))}
        </View>
      ) : null}

      <Animated.View
        style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: colors.like }]}
        pointerEvents="none"
      >
        <Text bold style={{ color: colors.like }}>
          J&apos;AIME
        </Text>
      </Animated.View>
      <Animated.View
        style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: colors.pass }]}
        pointerEvents="none"
      >
        <Text bold style={{ color: colors.pass }}>
          PASSER
        </Text>
      </Animated.View>

      <Pressable
        onPress={onOpenProfile}
        accessibilityRole="button"
        accessibilityLabel={`Voir le profil de ${profile.firstName}`}
        style={[styles.info, { backgroundColor: colors.scrim }]}
        testID="open-profile"
      >
        <Text variant="title" style={styles.white}>
          {profile.firstName}, {profile.age}
        </Text>
        {profile.city || distance ? (
          <Text style={styles.white}>{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.slice(0, 3).map((i) => (
              <Chip key={i.id} label={i.label} selected />
            ))}
          </View>
        ) : null}
      </Pressable>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: { flex: 1, borderRadius: radius.lg, overflow: "hidden" },
  tapRow: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  tapZone: { flex: 1 },
  dots: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    gap: 4
  },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, gap: spacing.xs },
  chips: { flexDirection: "row", gap: spacing.xs, flexWrap: "wrap", marginTop: spacing.xs },
  white: { color: "#fff" },
  stamp: {
    position: "absolute",
    top: spacing.xxl,
    borderWidth: 3,
    borderRadius: radius.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  likeStamp: { left: spacing.lg, transform: [{ rotate: "-12deg" }] },
  passStamp: { right: spacing.lg, transform: [{ rotate: "12deg" }] }
});
