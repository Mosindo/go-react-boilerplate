import React, { useRef, useState } from "react";
import {
  Animated,
  PanResponder,
  Pressable,
  StyleSheet,
  Text,
  View,
  useWindowDimensions
} from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { formatDistance } from "../lib/format";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { AuthImage } from "./AuthImage";

const SWIPE_THRESHOLD = 110;

type Props = {
  profile: PublicProfile;
  /** Only the top card of the deck reacts to gestures. */
  active: boolean;
  onSwipe: (action: SwipeAction) => void;
  onOpen: () => void;
};

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

export function SwipeCardInfo({ profile }: { profile: PublicProfile }) {
  const distance = formatDistance(profile.distanceKm);
  return (
    <View>
      <Text style={styles.name}>
        {profile.firstName}, {profile.age}
      </Text>
      {distance || profile.city ? (
        <Text style={styles.sub}>{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
      ) : null}
      {profile.interests.length > 0 ? (
        <View style={styles.tags}>
          {profile.interests.slice(0, 3).map((interest) => (
            <View key={interest.id} style={styles.tag}>
              <Text style={styles.tagText}>{interest.label}</Text>
            </View>
          ))}
        </View>
      ) : null}
    </View>
  );
}

export const SwipeCard = React.forwardRef<SwipeCardHandle, Props>(function SwipeCard(
  { profile, active, onSwipe, onOpen },
  ref
) {
  const theme = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const photos = profile.photos;
  const callbacks = useRef({ onSwipe, active });
  callbacks.current = { onSwipe, active };

  const flyOut = (action: SwipeAction) => {
    Animated.timing(position, {
      toValue: { x: action === "like" ? width * 1.4 : -width * 1.4, y: 40 },
      duration: 220,
      useNativeDriver: true
    }).start(() => callbacks.current.onSwipe(action));
  };

  React.useImperativeHandle(ref, () => ({ swipe: flyOut }));

  const panResponder = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (_, g) =>
        callbacks.current.active && (Math.abs(g.dx) > 8 || Math.abs(g.dy) > 8),
      onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
        useNativeDriver: false
      }),
      onPanResponderRelease: (_, g) => {
        if (g.dx > SWIPE_THRESHOLD) flyOut("like");
        else if (g.dx < -SWIPE_THRESHOLD) flyOut("pass");
        else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
      }
    })
  ).current;

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

  const current = photos[Math.min(photoIndex, photos.length - 1)];

  const changePhoto = (delta: number) =>
    setPhotoIndex((index) => Math.max(0, Math.min(photos.length - 1, index + delta)));

  return (
    <Animated.View
      {...(active ? panResponder.panHandlers : {})}
      accessibilityLabel={`Profil de ${profile.firstName}, ${profile.age} ans`}
      style={[
        styles.card,
        { backgroundColor: theme.surfaceMuted },
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
    >
      {current ? <AuthImage style={StyleSheet.absoluteFill} url={current.url} /> : null}

      {photos.length > 1 ? (
        <View style={styles.bars}>
          {photos.map((photo, index) => (
            <View
              key={photo.id}
              style={[
                styles.bar,
                { backgroundColor: index === photoIndex ? "#FFFFFF" : "rgba(255,255,255,0.4)" }
              ]}
            />
          ))}
        </View>
      ) : null}

      <View style={styles.tapZones}>
        <Pressable
          accessibilityLabel="Photo précédente"
          onPress={() => changePhoto(-1)}
          style={styles.zone}
        />
        <Pressable
          accessibilityLabel="Photo suivante"
          onPress={() => changePhoto(1)}
          style={styles.zone}
        />
      </View>

      <Animated.View style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}>
        <Text style={[styles.stampText, { color: "#3DDC97" }]}>J&apos;AIME</Text>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}>
        <Text style={[styles.stampText, { color: "#FF8A80" }]}>PASSER</Text>
      </Animated.View>

      <Pressable
        accessibilityHint="Ouvre le profil complet"
        accessibilityRole="button"
        onPress={onOpen}
        style={styles.footer}
      >
        <SwipeCardInfo profile={profile} />
      </Pressable>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    overflow: "hidden"
  },
  bars: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.sm,
    right: spacing.sm,
    flexDirection: "row",
    gap: 4
  },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  zone: { flex: 1 },
  footer: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xl,
    backgroundColor: "rgba(20, 12, 22, 0.55)"
  },
  name: { color: "#FFFFFF", fontSize: 28, fontWeight: "700", letterSpacing: -0.5 },
  sub: { color: "rgba(255,255,255,0.9)", fontSize: fontSizes.md, marginTop: 2 },
  tags: { flexDirection: "row", flexWrap: "wrap", marginTop: spacing.sm, gap: spacing.xs },
  tag: {
    backgroundColor: "rgba(255,255,255,0.22)",
    borderRadius: radii.pill,
    paddingHorizontal: 10,
    paddingVertical: 4
  },
  tagText: { color: "#FFFFFF", fontSize: fontSizes.xs, fontWeight: "600" },
  stamp: {
    position: "absolute",
    top: 60,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: 12,
    paddingVertical: 4
  },
  stampLike: { left: 24, borderColor: "#3DDC97", transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, borderColor: "#FF8A80", transform: [{ rotate: "12deg" }] },
  stampText: { fontSize: 30, fontWeight: "800", letterSpacing: 2 }
});
