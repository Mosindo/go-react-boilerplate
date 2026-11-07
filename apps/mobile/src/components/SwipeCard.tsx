import React, { useEffect, useMemo, useRef } from "react";
import { Animated, PanResponder, Platform, View } from "react-native";
import { PhotoCarousel } from "./ProfileView";
import { AppText } from "./ui";
import { radius, spacing, useTheme } from "../theme/theme";
import type { Card } from "../api/types";

const THRESHOLD = 110;

/** The top card of the deck. Drag right to like, left to pass; buttons trigger the same callbacks. */
export default function SwipeCard({
  card,
  onDecide,
  onOpen,
  decideRef,
}: {
  card: Card;
  onDecide: (action: "like" | "pass") => void;
  onOpen: () => void;
  /** Lets the parent's buttons animate the card away. */
  decideRef: React.MutableRefObject<((action: "like" | "pass") => void) | null>;
}) {
  const { colors } = useTheme();
  const position = useRef(new Animated.ValueXY()).current;
  const native = Platform.OS !== "web";

  const fling = (action: "like" | "pass") => {
    Animated.timing(position, {
      toValue: { x: action === "like" ? 700 : -700, y: 0 },
      duration: 180,
      useNativeDriver: native,
    }).start(() => {
      position.setValue({ x: 0, y: 0 });
      onDecide(action);
    });
  };
  const flingRef = useRef(fling);
  flingRef.current = fling;

  useEffect(() => {
    decideRef.current = (a) => flingRef.current(a);
    return () => {
      decideRef.current = null;
    };
  }, [decideRef]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
          useNativeDriver: false,
        }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > THRESHOLD) flingRef.current("like");
          else if (g.dx < -THRESHOLD) flingRef.current("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        },
      }),
    [position],
  );

  const rotate = position.x.interpolate({
    inputRange: [-300, 0, 300],
    outputRange: ["-8deg", "0deg", "8deg"],
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp",
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-THRESHOLD, 0],
    outputRange: [1, 0],
    extrapolate: "clamp",
  });

  return (
    <Animated.View
      {...responder.panHandlers}
      style={{ transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }}
    >
      <PhotoCarousel card={card} />
      <Animated.View
        pointerEvents="none"
        style={{
          position: "absolute",
          top: spacing.xl + 12,
          left: spacing.lg,
          opacity: likeOpacity,
          borderWidth: 3,
          borderColor: colors.success,
          borderRadius: radius.sm,
          paddingHorizontal: spacing.sm,
          transform: [{ rotate: "-12deg" }],
        }}
      >
        <AppText variant="heading" style={{ color: colors.success }}>
          LIKE
        </AppText>
      </Animated.View>
      <Animated.View
        pointerEvents="none"
        style={{
          position: "absolute",
          top: spacing.xl + 12,
          right: spacing.lg,
          opacity: passOpacity,
          borderWidth: 3,
          borderColor: "#fff",
          borderRadius: radius.sm,
          paddingHorizontal: spacing.sm,
          transform: [{ rotate: "12deg" }],
        }}
      >
        <AppText variant="heading" style={{ color: "#fff" }}>
          PASSER
        </AppText>
      </Animated.View>
      <View style={{ position: "absolute", right: spacing.md, bottom: spacing.md }}>
        <AppText
          testID="open-profile"
          accessibilityRole="button"
          accessibilityLabel="Voir le profil complet"
          onPress={onOpen}
          style={{
            color: "#fff",
            fontWeight: "700",
            fontSize: 14,
            backgroundColor: "rgba(0,0,0,0.5)",
            paddingHorizontal: spacing.md,
            paddingVertical: spacing.sm,
            borderRadius: radius.pill,
            overflow: "hidden",
          }}
        >
          Profil ⓘ
        </AppText>
      </View>
    </Animated.View>
  );
}
