import React, { useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { Candidate, SwipeAction } from "../api/types";
import { formatDistance } from "../domain/distance";
import { Text } from "../shared/ui";
import { radius, spacing } from "../theme";
import { PhotoCarousel } from "./PhotoCarousel";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

export type SwipeCardProps = {
  candidate: Candidate;
  onSwiped: (action: SwipeAction) => void;
  onOpenProfile: () => void;
  ref?: React.Ref<SwipeCardHandle>;
};

const SWIPE_THRESHOLD_RATIO = 0.28;

export function SwipeCard({ candidate, onSwiped, onOpenProfile, ref }: SwipeCardProps) {
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const settled = useRef(false);
  const [photoIndex, setPhotoIndex] = useState(0);
  const onSwipedRef = useRef(onSwiped);
  useEffect(() => {
    onSwipedRef.current = onSwiped;
  }, [onSwiped]);

  const flyOut = useMemo(
    () =>
      (action: SwipeAction, velocityX = 0) => {
        if (settled.current) {
          return;
        }
        settled.current = true;
        Animated.timing(position, {
          toValue: { x: action === "like" ? width * 1.4 : -width * 1.4, y: 0 },
          duration: velocityX > 1 ? 160 : 240,
          useNativeDriver: false
        }).start(() => onSwipedRef.current(action));
      },
    [position, width]
  );

  useImperativeHandle(ref, () => ({ swipe: (action) => flyOut(action) }), [flyOut]);

  const panResponder = useMemo(() => {
    const springBack = () =>
      Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
    return PanResponder.create({
      onMoveShouldSetPanResponder: (_event, gesture) =>
        !settled.current && Math.abs(gesture.dx) > 8 && Math.abs(gesture.dx) > Math.abs(gesture.dy),
      onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
      onPanResponderRelease: (_event, gesture) => {
        const threshold = width * SWIPE_THRESHOLD_RATIO;
        if (gesture.dx > threshold || gesture.vx > 0.9) {
          flyOut("like", gesture.vx);
        } else if (gesture.dx < -threshold || gesture.vx < -0.9) {
          flyOut("pass", Math.abs(gesture.vx));
        } else {
          springBack();
        }
      },
      onPanResponderTerminate: springBack
    });
  }, [flyOut, position, width]);

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, width * 0.3],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-width * 0.3, 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const distance = formatDistance(candidate.distanceKm);

  return (
    <Animated.View
      {...panResponder.panHandlers}
      accessibilityLabel={`${candidate.firstName}, ${candidate.age}. Use the Like and Pass buttons below to respond.`}
      style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
    >
      <PhotoCarousel
        index={photoIndex}
        name={candidate.firstName}
        onCenterPress={onOpenProfile}
        onIndexChange={setPhotoIndex}
        photos={candidate.photos}
      />
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity }]}>
        <Text style={styles.stampText} variant="heading">
          LIKE
        </Text>
      </Animated.View>
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.passStamp, { opacity: passOpacity }]}>
        <Text style={styles.stampText} variant="heading">
          PASS
        </Text>
      </Animated.View>
      <View pointerEvents="none" style={styles.info}>
        <Text style={styles.infoName} variant="title">
          {candidate.firstName}, {candidate.age}
        </Text>
        <Text style={styles.infoLine}>{[candidate.locationLabel, distance].filter(Boolean).join(" · ")}</Text>
        {candidate.sharedInterestCount > 0 ? (
          <Text style={styles.infoLine} variant="label">
            {candidate.sharedInterestCount} {candidate.sharedInterestCount === 1 ? "interest" : "interests"} in common
          </Text>
        ) : null}
      </View>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radius.lg },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    gap: 2,
    backgroundColor: "rgba(20, 10, 24, 0.62)",
    borderBottomLeftRadius: radius.lg,
    borderBottomRightRadius: radius.lg
  },
  infoName: { color: "#FFFFFF" },
  infoLine: { color: "#FFFFFF" },
  stamp: {
    position: "absolute",
    top: spacing.xl + spacing.lg,
    borderWidth: 3,
    borderRadius: radius.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: 2
  },
  likeStamp: { left: spacing.lg, borderColor: "#7CE0AE", transform: [{ rotate: "-12deg" }] },
  passStamp: { right: spacing.lg, borderColor: "#FF9C8F", transform: [{ rotate: "12deg" }] },
  stampText: { color: "#FFFFFF" }
});
