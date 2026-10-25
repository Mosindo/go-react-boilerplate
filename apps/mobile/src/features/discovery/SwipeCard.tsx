import React, { useEffect, useImperativeHandle, useState } from "react";
import { Animated, StyleSheet, View, useWindowDimensions, type AccessibilityActionEvent } from "react-native";
import type { PublicProfile, SwipeAction } from "../../api";
import { formatDistance } from "../../shared/format";
import { AuthImage, IconButton, Text } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";
import { SwipeController, type SwipeProps } from "./swipeController";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profile: PublicProfile;
  /** Only the top card reacts to touches. */
  interactive: boolean;
  /** 0 to 1 while the top card is dragged; lets the card behind scale up. */
  dragProgress: Animated.Value;
  onSwiped: (action: SwipeAction) => void;
  onOpenProfile: () => void;
  ref?: React.Ref<SwipeCardHandle>;
};

export const SWIPE_ACCESSIBILITY_ACTIONS = [
  { name: "activate", label: "Voir le profil complet" },
  { name: "like", label: "J'aime" },
  { name: "pass", label: "Passer" }
] as const;

export function SwipeCard({ profile, interactive, dragProgress, onSwiped, onOpenProfile, ref }: Props) {
  const { colors } = useTheme();
  const { width: windowWidth } = useWindowDimensions();
  const cardWidth = windowWidth - spacing.lg * 2;
  const [photoIndex, setPhotoIndex] = useState(0);
  const photoCount = profile.photos.length;

  const swipeProps: SwipeProps = {
    interactive,
    cardWidth,
    photoCount,
    onSwiped,
    onOpenProfile,
    onPhotoStep: (delta) => setPhotoIndex((index) => Math.min(Math.max(index + delta, 0), photoCount - 1))
  };
  const [controller] = useState(() => new SwipeController(dragProgress, swipeProps));
  useEffect(() => {
    controller.update(swipeProps);
  });
  useImperativeHandle(ref, () => ({ swipe: (action) => controller.swipe(action) }), [controller]);
  const position = controller.position;

  const rotate = position.x.interpolate({
    inputRange: [-cardWidth, 0, cardWidth],
    outputRange: ["-12deg", "0deg", "12deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, cardWidth * 0.3],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-cardWidth * 0.3, 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const behindScale = dragProgress.interpolate({ inputRange: [0, 1], outputRange: [0.94, 1] });

  const photo = profile.photos[Math.min(photoIndex, photoCount - 1)];
  const distance = formatDistance(profile.distanceKm);
  const place = [profile.city, distance].filter(Boolean).join(" · ");

  const handleAccessibilityAction = (event: AccessibilityActionEvent) => {
    const name = event.nativeEvent.actionName;
    if (name === "like" || name === "pass") controller.swipe(name);
    else if (name === "activate") onOpenProfile();
  };

  return (
    <Animated.View
      accessibilityActions={interactive ? [...SWIPE_ACCESSIBILITY_ACTIONS] : undefined}
      accessibilityLabel={`${profile.firstName}, ${profile.age} ans${place ? `, ${place}` : ""}`}
      accessible={interactive}
      onAccessibilityAction={handleAccessibilityAction}
      style={[
        styles.card,
        { width: cardWidth, backgroundColor: colors.surfaceAlt },
        interactive
          ? { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
          : { transform: [{ scale: behindScale }] }
      ]}
      testID={interactive ? "deck-top-card" : undefined}
      {...(interactive ? controller.responder.panHandlers : {})}
    >
      <View pointerEvents="none" style={StyleSheet.absoluteFill}>
        <AuthImage path={photo?.url} style={StyleSheet.absoluteFill} />
      </View>

      {photoCount > 1 ? (
        <View pointerEvents="none" style={styles.dots}>
          {profile.photos.map((item, index) => (
            <View
              key={item.id}
              style={[styles.dot, { backgroundColor: index === photoIndex ? colors.onScrim : "rgba(255,255,255,0.4)" }]}
            />
          ))}
        </View>
      ) : null}

      <Animated.View
        pointerEvents="none"
        style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.like }]}
      >
        <Text style={{ color: colors.like }} variant="heading">
          J&apos;AIME
        </Text>
      </Animated.View>
      <Animated.View
        pointerEvents="none"
        style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: colors.pass }]}
      >
        <Text style={{ color: colors.pass }} variant="heading">
          PASSER
        </Text>
      </Animated.View>

      <View pointerEvents="box-none" style={[styles.info, { backgroundColor: colors.scrim }]}>
        <View pointerEvents="none" style={styles.infoText}>
          <Text style={{ color: colors.onScrim }} variant="heading">
            {`${profile.firstName}, ${profile.age}`}
          </Text>
          {place ? (
            <Text style={{ color: colors.onScrim }} variant="caption">
              {place}
            </Text>
          ) : null}
          {profile.sharedInterests.length > 0 ? (
            <Text style={{ color: colors.onScrim }} variant="caption">
              {`${profile.sharedInterests.length} centre${profile.sharedInterests.length > 1 ? "s" : ""} d'intérêt en commun`}
            </Text>
          ) : null}
        </View>
        {interactive ? (
          <IconButton
            color="onScrim"
            icon="information-circle-outline"
            label={`Voir le profil de ${profile.firstName}`}
            onPress={onOpenProfile}
            testID="deck-open-profile"
          />
        ) : null}
      </View>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  card: { position: "absolute", top: 0, bottom: 0, alignSelf: "center", borderRadius: radius.lg, overflow: "hidden" },
  dots: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  stamp: {
    position: "absolute",
    top: spacing.xxl,
    borderWidth: 3,
    borderRadius: radius.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  stampLike: { left: spacing.xl, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: spacing.xl, transform: [{ rotate: "12deg" }] },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    alignItems: "center",
    padding: spacing.lg
  },
  infoText: { flex: 1, gap: 2 }
});
