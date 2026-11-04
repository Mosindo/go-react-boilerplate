import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, Dimensions, PanResponder, Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile, SwipeAction } from "../../api/types";
import { Chip, Text, radius, spacing } from "../../shared/ui";
import { PhotoCarousel } from "./PhotoCarousel";
import { distanceLabel } from "./ProfileSummary";

const SCREEN_WIDTH = Dimensions.get("window").width;
const THRESHOLD = SCREEN_WIDTH * 0.28;

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = { profile: PublicProfile; onSwiped: (action: SwipeAction) => void; onOpen: () => void };

export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ profile, onSwiped, onOpen }, ref) {
  const position = useRef(new Animated.ValueXY()).current;
  const onSwipedRef = useRef(onSwiped);
  onSwipedRef.current = onSwiped;

  const flyOut = (action: SwipeAction) => {
    Animated.timing(position, {
      toValue: { x: (action === "like" ? 1 : -1) * SCREEN_WIDTH * 1.4, y: 40 },
      duration: 220,
      useNativeDriver: true,
    }).start(() => onSwipedRef.current(action));
  };
  const flyOutRef = useRef(flyOut);
  flyOutRef.current = flyOut;

  useImperativeHandle(ref, () => ({ swipe: (action) => flyOutRef.current(action) }), []);

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => Math.abs(g.dx) > 10 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          if (g.dx > THRESHOLD) flyOutRef.current("like");
          else if (g.dx < -THRESHOLD) flyOutRef.current("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
        },
      }),
    [position],
  );

  const rotate = position.x.interpolate({ inputRange: [-SCREEN_WIDTH, 0, SCREEN_WIDTH], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });

  return (
    <Animated.View
      testID="swipe-card"
      accessibilityLabel={`${profile.firstName}, ${profile.age} ans`}
      style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
      {...pan.panHandlers}
    >
      <PhotoCarousel photos={profile.photos} name={profile.firstName}>
        <Animated.View style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]} pointerEvents="none">
          <Text variant="heading" style={{ color: "#2E7D5B" }}>J'AIME</Text>
        </Animated.View>
        <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]} pointerEvents="none">
          <Text variant="heading" style={{ color: "#B3261E" }}>PASSER</Text>
        </Animated.View>
        <View style={styles.info} pointerEvents="box-none">
          <Pressable onPress={onOpen} accessibilityRole="button" accessibilityLabel={`Voir le profil complet de ${profile.firstName}`} testID="open-profile" style={{ gap: 4 }}>
            <Text variant="title" style={styles.white}>{profile.firstName}, {profile.age}</Text>
            {distanceLabel(profile) ? <Text style={styles.white}>{distanceLabel(profile)}</Text> : null}
            {profile.interests.length > 0 ? (
              <View style={styles.chips}>
                {profile.interests.slice(0, 3).map((i) => <Chip key={i.slug} label={i.label} />)}
              </View>
            ) : null}
            <Text variant="caption" style={styles.white}>Voir le profil complet ›</Text>
          </Pressable>
        </View>
      </PhotoCarousel>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radius.lg, shadowColor: "#000", shadowOpacity: 0.18, shadowRadius: 12, shadowOffset: { width: 0, height: 6 }, elevation: 6 },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, backgroundColor: "rgba(20,8,24,0.55)" },
  white: { color: "#FFFFFF" },
  chips: { flexDirection: "row", gap: spacing.sm, flexWrap: "wrap", marginTop: 4 },
  stamp: { position: "absolute", top: 40, borderWidth: 3, borderRadius: radius.md, paddingHorizontal: spacing.md, paddingVertical: spacing.xs, backgroundColor: "rgba(255,255,255,0.85)" },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }], borderColor: "#2E7D5B" },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }], borderColor: "#B3261E" },
});
