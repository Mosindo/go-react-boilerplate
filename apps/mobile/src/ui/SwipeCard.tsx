import React, { forwardRef, useImperativeHandle, useMemo, useRef, useState } from "react";
import {
  Animated,
  PanResponder,
  Pressable,
  StyleSheet,
  View,
  useWindowDimensions,
} from "react-native";

import type { Card } from "../api/types";
import { formatDistance } from "../lib/format";
import { radii, spacing, useTheme } from "../theme";
import { PhotoImage } from "./PhotoImage";
import { Text } from "./Text";

export type SwipeDirection = "like" | "pass";

export interface SwipeCardHandle {
  swipe: (direction: SwipeDirection) => void;
}

export interface SwipeCardProps {
  card: Card;
  /** Only the top card of the deck reacts to touch. */
  interactive: boolean;
  onSwiped: (direction: SwipeDirection) => void;
  onOpen: () => void;
}

const THRESHOLD = 110;

export const SwipeCard = forwardRef<SwipeCardHandle, SwipeCardProps>(function SwipeCard(
  { card, interactive, onSwiped, onOpen },
  ref,
) {
  const { colors } = useTheme();
  const { width: screenWidth } = useWindowDimensions();
  const pan = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const [cardWidth, setCardWidth] = useState(0);
  const callbacks = useRef({ onSwiped, onOpen, photos: card.photos.length, cardWidth });
  callbacks.current = { onSwiped, onOpen, photos: card.photos.length, cardWidth };

  const fling = (direction: SwipeDirection) => {
    Animated.timing(pan, {
      toValue: { x: (direction === "like" ? 1 : -1) * screenWidth * 1.3, y: 40 },
      duration: 220,
      useNativeDriver: false,
    }).start(() => callbacks.current.onSwiped(direction));
  };
  const flingRef = useRef(fling);
  flingRef.current = fling;

  useImperativeHandle(ref, () => ({ swipe: (d) => flingRef.current(d) }), []);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onStartShouldSetPanResponder: () => false,
        onMoveShouldSetPanResponder: (_, g) =>
          Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], {
          useNativeDriver: false,
        }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > THRESHOLD) flingRef.current("like");
          else if (g.dx < -THRESHOLD) flingRef.current("pass");
          else
            Animated.spring(pan, {
              toValue: { x: 0, y: 0 },
              friction: 6,
              useNativeDriver: false,
            }).start();
        },
      }),
    [pan],
  );

  const rotate = pan.x.interpolate({
    inputRange: [-250, 0, 250],
    outputRange: ["-10deg", "0deg", "10deg"],
  });
  const likeOpacity = pan.x.interpolate({
    inputRange: [20, THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp",
  });
  const passOpacity = pan.x.interpolate({
    inputRange: [-THRESHOLD, -20],
    outputRange: [1, 0],
    extrapolate: "clamp",
  });

  const tapPhoto = (x: number) => {
    const { photos, cardWidth: w } = callbacks.current;
    if (photos <= 1 || w === 0) return;
    setPhotoIndex((i) => (x < w / 2 ? Math.max(0, i - 1) : Math.min(photos - 1, i + 1)));
  };

  const distance = formatDistance(card.distanceKm);

  return (
    <Animated.View
      {...(interactive ? responder.panHandlers : {})}
      onLayout={(e) => setCardWidth(e.nativeEvent.layout.width)}
      style={[
        styles.card,
        { backgroundColor: colors.surface },
        interactive && { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] },
      ]}
    >
      <Pressable
        style={StyleSheet.absoluteFill}
        accessibilityLabel={`Photo ${photoIndex + 1} sur ${Math.max(1, card.photos.length)} de ${card.firstName}`}
        onPress={(e) => tapPhoto(e.nativeEvent.locationX)}
      >
        <PhotoImage
          photo={card.photos[photoIndex]}
          name={card.firstName}
          style={StyleSheet.absoluteFill}
        />
      </Pressable>

      {card.photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {card.photos.map((p, i) => (
            <View
              key={p.id}
              style={[
                styles.dot,
                { backgroundColor: i === photoIndex ? "#fff" : "rgba(255,255,255,0.45)" },
              ]}
            />
          ))}
        </View>
      ) : null}

      <Pressable
        style={styles.info}
        onPress={onOpen}
        accessibilityRole="button"
        accessibilityLabel={`Voir le profil de ${card.firstName}`}
      >
        <View style={styles.nameRow}>
          <Text variant="title" style={styles.white}>
            {card.firstName}
          </Text>
          <Text variant="title" style={[styles.white, { fontWeight: "400" }]}>
            {card.age}
          </Text>
        </View>
        {card.city || distance ? (
          <Text style={styles.white}>{[card.city, distance].filter(Boolean).join(" · ")}</Text>
        ) : null}
        {card.interests.length > 0 ? (
          <Text variant="caption" style={styles.white} numberOfLines={1}>
            {card.interests
              .slice(0, 4)
              .map((i) => i.label)
              .join(" · ")}
          </Text>
        ) : null}
      </Pressable>

      {interactive ? (
        <>
          <Animated.View
            pointerEvents="none"
            style={[
              styles.stamp,
              styles.stampLike,
              { opacity: likeOpacity, borderColor: colors.like },
            ]}
          >
            <Text variant="heading" style={{ color: colors.like }}>
              J&apos;AIME
            </Text>
          </Animated.View>
          <Animated.View
            pointerEvents="none"
            style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#fff" }]}
          >
            <Text variant="heading" style={styles.white}>
              PASSER
            </Text>
          </Animated.View>
        </>
      ) : null}
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radii.lg, overflow: "hidden" },
  dots: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 4, borderRadius: 2 },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxl,
    backgroundColor: "rgba(20,12,22,0.55)",
    gap: 2,
  },
  nameRow: { flexDirection: "row", alignItems: "baseline", gap: spacing.sm },
  white: { color: "#fff" },
  stamp: {
    position: "absolute",
    top: 48,
    borderWidth: 3,
    borderRadius: radii.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
  },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] },
});
