import React, { useEffect, useMemo, useRef } from "react";
import { Animated, Dimensions, PanResponder, StyleSheet, View } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { radii } from "../theme";
import { Text } from "../ui/Text";
import { ProfileCard } from "./ProfileCard";

const SCREEN_WIDTH = Dimensions.get("window").width;
const THRESHOLD = SCREEN_WIDTH * 0.28;
const FLY_MS = 220;

export type DeckCommand = { id: string; action: SwipeAction } | null;

type Props = {
  profiles: PublicProfile[];
  command: DeckCommand;
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onInfo: (profile: PublicProfile) => void;
};

/** Shows the top card (draggable) over the next one. Drag or use the buttons via `command`. */
export function SwipeDeck({ profiles, command, onSwipe, onInfo }: Props) {
  const [top, next] = profiles;
  if (!top) return null;
  return (
    <View style={styles.deck}>
      {next ? (
        <View style={[styles.layer, styles.under]} pointerEvents="none">
          <ProfileCard profile={next} />
        </View>
      ) : null}
      <TopCard
        key={top.userId}
        profile={top}
        command={command?.id === top.userId ? command.action : null}
        onSwipe={onSwipe}
        onInfo={onInfo}
      />
    </View>
  );
}

function TopCard({
  profile,
  command,
  onSwipe,
  onInfo
}: {
  profile: PublicProfile;
  command: SwipeAction | null;
  onSwipe: Props["onSwipe"];
  onInfo: Props["onInfo"];
}) {
  const position = useRef(new Animated.ValueXY()).current;
  const done = useRef(false);

  const fly = useRef((action: SwipeAction) => {
    if (done.current) return;
    done.current = true;
    Animated.timing(position, {
      toValue: { x: action === "like" ? SCREEN_WIDTH * 1.4 : -SCREEN_WIDTH * 1.4, y: 0 },
      duration: FLY_MS,
      useNativeDriver: true
    }).start(() => onSwipe(profile, action));
  }).current;

  useEffect(() => {
    if (command) fly(command);
  }, [command, fly]);

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) =>
          Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > THRESHOLD || g.vx > 0.9) fly("like");
          else if (g.dx < -THRESHOLD || g.vx < -0.9) fly("pass");
          else
            Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: true }).start();
        }
      }),
    [fly, position]
  );

  const rotate = position.x.interpolate({
    inputRange: [-SCREEN_WIDTH, 0, SCREEN_WIDTH],
    outputRange: ["-12deg", "0deg", "12deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [20, THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-THRESHOLD, -20],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });

  return (
    <Animated.View
      {...pan.panHandlers}
      style={[
        styles.layer,
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
    >
      <ProfileCard profile={profile} onInfo={() => onInfo(profile)} />
      <Animated.View
        style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity }]}
        pointerEvents="none"
      >
        <Text variant="heading" style={styles.likeText}>
          LIKE
        </Text>
      </Animated.View>
      <Animated.View
        style={[styles.stamp, styles.passStamp, { opacity: passOpacity }]}
        pointerEvents="none"
      >
        <Text variant="heading" style={styles.passText}>
          PASS
        </Text>
      </Animated.View>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  deck: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  under: { transform: [{ scale: 0.96 }], top: 8 },
  stamp: {
    position: "absolute",
    top: 36,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: 12,
    paddingVertical: 4
  },
  likeStamp: { left: 24, borderColor: "#7CE0B1", transform: [{ rotate: "-14deg" }] },
  passStamp: { right: 24, borderColor: "#FF9C8A", transform: [{ rotate: "14deg" }] },
  likeText: { color: "#7CE0B1" },
  passText: { color: "#FF9C8A" }
});
