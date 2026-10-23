import React, {
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
  forwardRef
} from "react";
import { Animated, PanResponder, View } from "react-native";
import type { Card, SwipeAction } from "../api/types";
import { t } from "../i18n/fr";
import { formatDistance } from "../lib/format";
import { decideSwipe, isTap, rotationDegrees } from "../lib/swipe";
import { createStyles } from "../theme/ThemeProvider";
import { Photo, Text } from "../ui";

export type SwipeDeckHandle = { fling: (action: SwipeAction) => void };

type Props = {
  cards: Card[];
  onDecision: (card: Card, action: SwipeAction) => void;
  onOpen: (card: Card) => void;
};

const useStyles = createStyles(({ colors, radius, spacing }) => ({
  area: { flex: 1, alignItems: "center", justifyContent: "center" },
  card: {
    position: "absolute",
    top: 0,
    bottom: 0,
    left: 0,
    right: 0,
    borderRadius: radius.lg,
    overflow: "hidden",
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  behind: { transform: [{ scale: 0.95 }], opacity: 0.9 },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    backgroundColor: "rgba(20, 8, 18, 0.55)",
    gap: 2
  },
  white: { color: "#FFFFFF" },
  stamp: {
    position: "absolute",
    top: 28,
    borderWidth: 3,
    borderRadius: radius.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    backgroundColor: "rgba(0,0,0,0.25)"
  },
  stampLike: { left: 20, borderColor: "#FFD08A", transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 20, borderColor: "#FFFFFF", transform: [{ rotate: "12deg" }] }
}));

function CardFace({ card }: { card: Card }) {
  const styles = useStyles();
  const distance = formatDistance(card.distanceKm);
  const main = card.photos[0];
  return (
    <>
      <Photo
        label={t.a11y.photoOf(card.firstName)}
        name={card.firstName}
        path={main?.url ?? null}
        style={{ flex: 1 }}
      />
      <View style={styles.info}>
        <Text variant="title" style={styles.white}>
          {card.firstName}, {card.age}
        </Text>
        <Text style={styles.white}>
          {card.city}
          {distance ? ` · ${distance}` : ""}
        </Text>
      </View>
    </>
  );
}

/** Tinder-free swipe deck: PanResponder + Animated, with imperative fling for the buttons. */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck(
  { cards, onDecision, onOpen },
  ref
) {
  const styles = useStyles();
  const pan = useRef(new Animated.ValueXY()).current;
  const [width, setWidth] = useState(320);
  const [gone, setGone] = useState<ReadonlySet<string>>(new Set());
  const busy = useRef(false);

  const visible = useMemo(() => cards.filter((card) => !gone.has(card.userId)), [cards, gone]);
  const top = visible[0];
  const next = visible[1];

  const latest = useRef({ top, onDecision, onOpen, width });
  latest.current = { top, onDecision, onOpen, width };

  const commit = useCallback(
    (action: SwipeAction, velocity = 0) => {
      const current = latest.current.top;
      if (!current || busy.current) return;
      busy.current = true;
      const toX = (action === "like" ? 1 : -1) * latest.current.width * 1.4;
      Animated.timing(pan, {
        toValue: { x: toX, y: 0 },
        duration: Math.max(140, 240 - Math.abs(velocity) * 60),
        useNativeDriver: false
      }).start(() => {
        setGone((previous) => new Set(previous).add(current.userId));
        latest.current.onDecision(current, action);
        pan.setValue({ x: 0, y: 0 });
        busy.current = false;
      });
    },
    [pan]
  );

  useImperativeHandle(ref, () => ({ fling: (action) => commit(action) }), [commit]);

  // Allow a card to come back when the parent un-swipes it (failed request).
  useEffect(() => {
    setGone((previous) => {
      const ids = new Set(cards.map((card) => card.userId));
      const kept = [...previous].filter((id) => ids.has(id));
      return kept.length === previous.size ? previous : new Set(kept);
    });
  }, [cards]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onStartShouldSetPanResponder: () => !busy.current,
        onMoveShouldSetPanResponder: (_e, g) => !busy.current && Math.abs(g.dx) > 4,
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (_e, g) => {
          const gesture = { dx: g.dx, dy: g.dy, vx: g.vx };
          if (isTap(gesture)) {
            const current = latest.current.top;
            if (current) latest.current.onOpen(current);
            Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
            return;
          }
          const action = decideSwipe(gesture, latest.current.width);
          if (action) commit(action, g.vx);
          else
            Animated.spring(pan, {
              toValue: { x: 0, y: 0 },
              useNativeDriver: false,
              bounciness: 8
            }).start();
        },
        onPanResponderTerminate: () => {
          Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [commit, pan]
  );

  const rotate = pan.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: [
      `${-rotationDegrees(width, width)}deg`,
      "0deg",
      `${rotationDegrees(width, width)}deg`
    ]
  });
  const likeOpacity = pan.x.interpolate({
    inputRange: [0, Math.max(90, width * 0.28)],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = pan.x.interpolate({
    inputRange: [-Math.max(90, width * 0.28), 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });

  if (!top)
    return <View onLayout={(e) => setWidth(e.nativeEvent.layout.width)} style={styles.area} />;

  return (
    <View onLayout={(e) => setWidth(e.nativeEvent.layout.width)} style={styles.area}>
      {next ? (
        <View key={next.userId} pointerEvents="none" style={[styles.card, styles.behind]}>
          <CardFace card={next} />
        </View>
      ) : null}
      <Animated.View
        key={top.userId}
        {...responder.panHandlers}
        accessible
        accessibilityActions={[
          { name: "activate", label: t.discover.openProfile(top.firstName) },
          { name: "like", label: t.discover.likeLabel(top.firstName) },
          { name: "pass", label: t.discover.passLabel(top.firstName) }
        ]}
        accessibilityLabel={`${top.firstName}, ${top.age}, ${top.city}`}
        accessibilityRole="button"
        onAccessibilityAction={(event) => {
          const name = event.nativeEvent.actionName;
          if (name === "like" || name === "pass") commit(name);
          else onOpen(top);
        }}
        style={[
          styles.card,
          { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }
        ]}
        testID="discover-card"
      >
        <CardFace card={top} />
        <Animated.View
          pointerEvents="none"
          style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}
        >
          <Text variant="heading" style={{ color: "#FFD08A" }}>
            {t.discover.stamp.like}
          </Text>
        </Animated.View>
        <Animated.View
          pointerEvents="none"
          style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}
        >
          <Text variant="heading" style={styles.white}>
            {t.discover.stamp.pass}
          </Text>
        </Animated.View>
      </Animated.View>
    </View>
  );
});
