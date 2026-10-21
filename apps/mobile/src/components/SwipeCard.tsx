import { Ionicons } from "@expo/vector-icons";
import { Image } from "expo-image";
import React, { forwardRef, useImperativeHandle, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, useWindowDimensions, View } from "react-native";
import { absoluteUrl } from "../api/client";
import type { PublicProfile, SwipeAction } from "../api/types";
import { formatDistance } from "../lib/format";
import { Avatar, Chip, Text } from "../shared/ui";
import { radius, spacing } from "../theme";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profile: PublicProfile;
  onSwiped: (action: SwipeAction) => void;
  onOpenDetails: () => void;
};

const THRESHOLD = 110;

/**
 * Profile card with drag-to-decide (right = like, left = pass). Buttons drive it through the ref so
 * gestures are an enhancement, never the only way to act (accessibility).
 */
export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ profile, onSwiped, onOpenDetails }, ref) {
  const { width } = useWindowDimensions();
  const pan = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const photos = profile.photos;
  const done = useRef(false);

  const flyOut = (action: SwipeAction) => {
    if (done.current) return;
    done.current = true;
    Animated.timing(pan, {
      toValue: { x: action === "like" ? width * 1.4 : -width * 1.4, y: 0 },
      duration: 200,
      useNativeDriver: false
    }).start(() => onSwiped(action));
  };
  const flyOutRef = useRef(flyOut);
  flyOutRef.current = flyOut;

  useImperativeHandle(ref, () => ({ swipe: (action) => flyOutRef.current(action) }), []);

  const responder = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponderCapture: (_, g) => Math.abs(g.dx) > 10 && Math.abs(g.dx) > Math.abs(g.dy),
      onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], { useNativeDriver: false }),
      onPanResponderRelease: (_, g) => {
        if (g.dx > THRESHOLD) flyOutRef.current("like");
        else if (g.dx < -THRESHOLD) flyOutRef.current("pass");
        else Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
      }
    })
  ).current;

  const rotate = pan.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = pan.x.interpolate({ inputRange: [20, THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = pan.x.interpolate({ inputRange: [-THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });

  const current = photos[photoIndex];
  const distance = formatDistance(profile.distanceKm);

  return (
    <Animated.View
      {...responder.panHandlers}
      testID="swipe-card"
      style={[styles.card, { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }]}
    >
      {current ? (
        <Image
          accessibilityLabel={`Photo ${photoIndex + 1} sur ${photos.length} de ${profile.firstName}`}
          source={{ uri: absoluteUrl(current.url) }}
          style={StyleSheet.absoluteFill}
          contentFit="cover"
          transition={120}
          recyclingKey={current.id}
        />
      ) : (
        <View style={[StyleSheet.absoluteFill, styles.center]}>
          <Avatar name={profile.firstName} size={120} />
        </View>
      )}

      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, i === photoIndex && styles.dotActive]} />
          ))}
        </View>
      ) : null}

      {/* Tap zones: left / right third browse photos. */}
      <View style={styles.zones}>
        <Pressable style={styles.zone} accessibilityLabel="Photo précédente" onPress={() => setPhotoIndex((i) => Math.max(0, i - 1))} />
        <Pressable
          style={styles.zone}
          accessibilityLabel="Photo suivante"
          onPress={() => setPhotoIndex((i) => Math.min(photos.length - 1, i + 1))}
        />
      </View>

      <Animated.View style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]} pointerEvents="none">
        <Text variant="heading" style={{ color: "#6BC49C" }}>
          J'AIME
        </Text>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]} pointerEvents="none">
        <Text variant="heading" style={{ color: "#FFB4A8" }}>
          PASSE
        </Text>
      </Animated.View>

      <Pressable
        testID="swipe-card-details"
        accessibilityRole="button"
        accessibilityLabel={`Voir le profil de ${profile.firstName}`}
        onPress={onOpenDetails}
        style={styles.info}
      >
        <View style={styles.titleRow}>
          <Text variant="title" tone="inverse" style={styles.shadow} numberOfLines={1} testID="swipe-card-name">
            {profile.firstName}, {profile.age}
          </Text>
          <Ionicons name="information-circle-outline" size={26} color="#FFFFFF" />
        </View>
        {profile.city || distance ? (
          <Text tone="inverse" style={styles.shadow}>
            {[profile.city, distance].filter(Boolean).join(" · ")}
          </Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.tags}>
            {profile.interests.slice(0, 3).map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
      </Pressable>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: { flex: 1, width: "100%", borderRadius: radius.lg, overflow: "hidden", backgroundColor: "#2B2025" },
  center: { alignItems: "center", justifyContent: "center" },
  dots: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: "#FFFFFF55" },
  dotActive: { backgroundColor: "#FFFFFF" },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  zone: { flex: 1 },
  stamp: { position: "absolute", top: 48, borderWidth: 3, borderRadius: radius.md, paddingHorizontal: spacing.md, paddingVertical: 2 },
  stampLike: { left: 20, borderColor: "#6BC49C", transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 20, borderColor: "#FFB4A8", transform: [{ rotate: "12deg" }] },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    paddingTop: spacing.xxl,
    gap: spacing.xs,
    backgroundColor: "rgba(20, 10, 16, 0.55)"
  },
  titleRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.sm },
  tags: { flexDirection: "row", gap: spacing.sm, flexWrap: "wrap", marginTop: spacing.xs },
  shadow: { textShadowColor: "rgba(0,0,0,0.5)", textShadowRadius: 4, flexShrink: 1 }
});
