import React, {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useMemo,
  useRef,
  useState
} from "react";
import {
  Animated,
  PanResponder,
  Pressable,
  StyleSheet,
  View,
  useWindowDimensions
} from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { fr } from "../i18n/fr";
import { radius, spacing, useTheme } from "../theme";
import { AuthImage } from "./AuthImage";
import { Chip, Text } from "./ui";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpen: (profile: PublicProfile) => void;
};

const THRESHOLD = 110;

/** Stack of profile cards. Drag right to like, left to pass; tap the photo edges to browse photos. */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck(
  { profiles, onSwipe, onOpen },
  ref
) {
  const top = profiles[0];
  const next = profiles[1];
  const { width } = useWindowDimensions();
  const cardWidth = Math.min(width - spacing.lg * 2, 420);
  const pan = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const flying = useRef(false);

  // Keep the latest props reachable from the (stable) PanResponder callbacks.
  const latest = useRef({ top, onSwipe, cardWidth, photoCount: top?.photos.length ?? 0 });
  latest.current = { top, onSwipe, cardWidth, photoCount: top?.photos.length ?? 0 };

  const fling = useCallback(
    (action: SwipeAction) => {
      const { top: current, onSwipe: notify, cardWidth: w } = latest.current;
      if (!current || flying.current) return;
      flying.current = true;
      Animated.timing(pan, {
        toValue: { x: (action === "like" ? 1 : -1) * w * 1.5, y: 40 },
        duration: 220,
        useNativeDriver: true
      }).start(() => {
        pan.setValue({ x: 0, y: 0 });
        setPhotoIndex(0);
        flying.current = false;
        notify(current, action);
      });
    },
    [pan]
  );

  useImperativeHandle(ref, () => ({ swipe: fling }), [fling]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onStartShouldSetPanResponder: () => true,
        onMoveShouldSetPanResponder: (_e, g) => Math.abs(g.dx) > 6 || Math.abs(g.dy) > 6,
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (e, g) => {
          if (Math.abs(g.dx) < 8 && Math.abs(g.dy) < 8) {
            // A tap: left third = previous photo, right third = next photo, centre = open profile.
            const { top: current, cardWidth: w, photoCount } = latest.current;
            const x = e.nativeEvent.locationX;
            Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
            if (x < w / 3) setPhotoIndex((i) => Math.max(0, i - 1));
            else if (x > (w * 2) / 3) setPhotoIndex((i) => Math.min(photoCount - 1, i + 1));
            else if (current) onOpenRef.current(current);
            return;
          }
          if (g.dx > THRESHOLD) fling("like");
          else if (g.dx < -THRESHOLD) fling("pass");
          else
            Animated.spring(pan, {
              toValue: { x: 0, y: 0 },
              friction: 6,
              useNativeDriver: false
            }).start();
        }
      }),
    [fling, pan]
  );
  const onOpenRef = useRef(onOpen);
  onOpenRef.current = onOpen;

  if (!top) return null;
  const rotate = pan.x.interpolate({
    inputRange: [-200, 0, 200],
    outputRange: ["-8deg", "0deg", "8deg"]
  });
  const likeOpacity = pan.x.interpolate({
    inputRange: [20, THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = pan.x.interpolate({
    inputRange: [-THRESHOLD, -20],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const behindScale = pan.x.interpolate({
    inputRange: [-200, 0, 200],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  return (
    <View style={[styles.stack, { width: cardWidth }]}>
      {next ? (
        <Animated.View
          style={[styles.card, StyleSheet.absoluteFill, { transform: [{ scale: behindScale }] }]}
          pointerEvents="none"
          accessibilityElementsHidden
          importantForAccessibility="no-hide-descendants"
        >
          <Card profile={next} index={0} />
        </Animated.View>
      ) : null}
      <Animated.View
        testID="deck-top-card"
        accessibilityLabel={`${top.firstName}, ${top.age} ans`}
        style={[
          styles.card,
          StyleSheet.absoluteFill,
          { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }
        ]}
        {...responder.panHandlers}
      >
        <Card profile={top} index={photoIndex} />
        <Animated.View
          pointerEvents="none"
          style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}
        >
          <Text variant="title" color="#2E7D5B">
            {fr.discover.like.toUpperCase()}
          </Text>
        </Animated.View>
        <Animated.View
          pointerEvents="none"
          style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}
        >
          <Text variant="title" color="#B3261E">
            {fr.discover.pass.toUpperCase()}
          </Text>
        </Animated.View>
      </Animated.View>
    </View>
  );
});

function Card({ profile, index }: { profile: PublicProfile; index: number }) {
  const { colors } = useTheme();
  const photo = profile.photos[Math.min(index, profile.photos.length - 1)];
  return (
    <View style={{ flex: 1, backgroundColor: colors.surfaceAlt }}>
      <AuthImage
        photoId={photo?.id}
        style={StyleSheet.absoluteFill as object}
        accessibilityLabel={`Photo de ${profile.firstName}`}
      />
      {profile.photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {profile.photos.map((p, i) => (
            <View
              key={p.id}
              style={[
                styles.dot,
                { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }
              ]}
            />
          ))}
        </View>
      ) : null}
      {/* Layered translucent bands emulate a gradient without an extra dependency. */}
      <View style={styles.scrimWrap} pointerEvents="none">
        {[0.05, 0.1, 0.16, 0.22, 0.3, 0.38, 0.46].map((alpha) => (
          <View key={alpha} style={{ height: 14, backgroundColor: `rgba(15,8,6,${alpha})` }} />
        ))}
        <View
          style={{
            padding: spacing.lg,
            paddingTop: spacing.md,
            gap: spacing.sm,
            backgroundColor: "rgba(15,8,6,0.55)"
          }}
        >
          <Text variant="title" color="#FFFFFF" testID="deck-name">
            {profile.firstName}, {profile.age}
          </Text>
          <Text variant="label" color="rgba(255,255,255,0.88)">
            {[
              profile.city,
              profile.distanceKm !== null ? fr.discover.kmAway(profile.distanceKm) : null
            ]
              .filter(Boolean)
              .join(" · ")}
          </Text>
          {profile.interests.length > 0 ? (
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 6 }}>
              {profile.interests.slice(0, 3).map((i) => (
                <Chip key={i.slug} label={i.label} selected />
              ))}
            </View>
          ) : null}
        </View>
      </View>
    </View>
  );
}

export function DeckActions({ children }: { children: React.ReactNode }) {
  return <View style={styles.actions}>{children}</View>;
}

export function PressableOverlay({ onPress, label }: { onPress: () => void; label: string }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={StyleSheet.absoluteFill}
    />
  );
}

const styles = StyleSheet.create({
  stack: { flex: 1, alignSelf: "center", maxHeight: 640 },
  card: { borderRadius: radius.lg, overflow: "hidden", backgroundColor: "#000" },
  scrimWrap: { position: "absolute", left: 0, right: 0, bottom: 0 },
  dots: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  stamp: {
    position: "absolute",
    top: 36,
    borderWidth: 3,
    borderRadius: radius.md,
    paddingHorizontal: 12,
    paddingVertical: 4,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  stampLike: { left: 20, borderColor: "#2E7D5B", transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 20, borderColor: "#B3261E", transform: [{ rotate: "12deg" }] },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xl,
    paddingVertical: spacing.lg
  }
});
