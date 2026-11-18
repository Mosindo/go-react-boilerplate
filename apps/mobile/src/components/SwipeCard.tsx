import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, Platform, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import type { ProfileCard, SwipeDecision } from "../api/types";
import { radii, spacing, useTheme } from "../theme";
import { Icon, Text } from "../ui";
import { goalLabels } from "../utils/labels";
import { PhotoCarousel } from "./PhotoCarousel";

export type SwipeCardHandle = { swipe: (decision: SwipeDecision) => void };

type Props = {
  profile: ProfileCard;
  onDecision: (decision: SwipeDecision) => void;
  onOpen: () => void;
};

const SWIPE_THRESHOLD = 110;

/**
 * Swipeable card built on the core Animated + PanResponder APIs (no native
 * gesture dependency). Buttons drive the same animation through the handle,
 * so the deck stays fully usable without gestures (accessibility, web).
 */
export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ profile, onDecision, onOpen }, ref) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const decided = useRef(false);
  const useNativeDriver = Platform.OS !== "web";

  const fling = (decision: SwipeDecision) => {
    if (decided.current) {
      return;
    }
    decided.current = true;
    const x = (decision === "like" ? 1 : -1) * (width + 120);
    Animated.timing(position, { toValue: { x, y: 40 }, duration: 240, useNativeDriver }).start(() => onDecision(decision));
  };

  useImperativeHandle(ref, () => ({ swipe: fling }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 0.8) {
            fling("like");
          } else if (g.dx < -SWIPE_THRESHOLD || g.vx < -0.8) {
            fling("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver }).start();
        }
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [position, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });

  const details = [profile.jobTitle, profile.distanceKm != null ? `à ${profile.distanceKm} km` : profile.city].filter(Boolean).join(" · ");

  return (
    <Animated.View
      {...responder.panHandlers}
      accessibilityHint="Glissez à droite pour aimer, à gauche pour passer"
      style={[
        styles.card,
        { backgroundColor: colors.surface, shadowColor: colors.shadow },
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
      testID="swipe-card"
    >
      <PhotoCarousel name={profile.firstName} photos={profile.photos} style={StyleSheet.absoluteFill}>
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: colors.success }]}>
          <Text style={[styles.stampText, { color: colors.success }]}>J’AIME</Text>
        </Animated.View>
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: "#FFFFFF" }]}>
          <Text style={[styles.stampText, { color: "#FFFFFF" }]}>PASSER</Text>
        </Animated.View>
        <View pointerEvents="box-none" style={styles.footer}>
          <View pointerEvents="none" style={styles.gradient} />
          <Pressable
            accessibilityLabel={`Voir le profil complet de ${profile.firstName}`}
            accessibilityRole="button"
            onPress={onOpen}
            style={styles.info}
            testID="swipe-card-open"
          >
            <View style={styles.nameRow}>
              <Text style={styles.name} variant="title">
                {profile.firstName}, {profile.age}
              </Text>
              {profile.isDemo ? (
                <View style={styles.demo}>
                  <Text style={styles.demoText} variant="caption">
                    Démo
                  </Text>
                </View>
              ) : null}
              <View style={styles.flex} />
              <Icon color="textOnAccent" name="information-circle-outline" size={26} />
            </View>
            {details ? <Text style={styles.light}>{details}</Text> : null}
            <Text numberOfLines={1} style={styles.light} variant="caption">
              {profile.sharedInterests > 0
                ? `${profile.sharedInterests} centre${profile.sharedInterests > 1 ? "s" : ""} d’intérêt en commun`
                : goalLabels[profile.relationshipGoal]}
            </Text>
          </Pressable>
        </View>
      </PhotoCarousel>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    overflow: "hidden",
    shadowOpacity: 0.18,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 12 },
    elevation: 6
  },
  footer: { position: "absolute", left: 0, right: 0, bottom: 0, zIndex: 3 },
  gradient: { ...StyleSheet.absoluteFillObject, backgroundColor: "rgba(10, 6, 4, 0.48)" },
  info: { padding: spacing.lg, gap: 4 },
  nameRow: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  name: { color: "#FFFFFF" },
  light: { color: "rgba(255,255,255,0.88)" },
  flex: { flex: 1 },
  demo: { backgroundColor: "rgba(255,255,255,0.22)", paddingHorizontal: 8, paddingVertical: 2, borderRadius: 999 },
  demoText: { color: "#FFFFFF", fontWeight: "600" },
  stamp: { position: "absolute", top: 36, borderWidth: 3, borderRadius: 10, paddingHorizontal: 12, paddingVertical: 4, zIndex: 4 },
  likeStamp: { left: 24, transform: [{ rotate: "-14deg" }] },
  passStamp: { right: 24, transform: [{ rotate: "14deg" }] },
  stampText: { fontSize: 26, fontWeight: "800", letterSpacing: 2 }
});
