import { useEffect, useImperativeHandle, useLayoutEffect, useMemo, useRef, useState } from 'react';
import type { Ref } from 'react';
import { Animated, PanResponder, View } from 'react-native';
import type { LayoutChangeEvent } from 'react-native';
import type { PublicProfile, SwipeAction } from '../../api';
import { fr } from '../../i18n';
import { useStyles } from '../../theme';
import { Text } from '../../ui';
import { ProfileCard } from './ProfileCard';
import { SWIPE_DISTANCE_RATIO, decideSwipe, rotationDeg } from './swipeLogic';

export interface DeckHandle {
  swipe: (action: SwipeAction) => void;
}

interface SwipeDeckProps {
  /** queue[0] is on top; at most the first two are rendered. */
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpen: (profile: PublicProfile) => void;
  handle: Ref<DeckHandle>;
  /** Reports whether a swipe-out animation is running (to disable buttons). */
  onBusyChange?: (busy: boolean) => void;
}

const EXIT_DURATION = 220;

export function SwipeDeck({ profiles, onSwipe, onOpen, handle, onBusyChange }: SwipeDeckProps) {
  const [width, setWidth] = useState(0);
  const position = useRef(new Animated.ValueXY()).current;
  const top = profiles[0];
  const behind = profiles[1];
  const styles = useStyles((t) => ({
    stage: { flex: 1 },
    layer: { position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 },
    stamp: {
      position: 'absolute',
      top: 28,
      borderWidth: 4,
      borderRadius: t.radius.md,
      paddingHorizontal: t.spacing.md,
      paddingVertical: t.spacing.xs,
    },
    stampLike: { left: 24, borderColor: t.colors.success, transform: [{ rotate: '-12deg' }] },
    stampPass: { right: 24, borderColor: t.colors.danger, transform: [{ rotate: '12deg' }] },
  }));

  // A new top card must start centred before it paints.
  useLayoutEffect(() => {
    position.setValue({ x: 0, y: 0 });
  }, [top?.userId, position]);

  const onLayout = (e: LayoutChangeEvent) => setWidth(e.nativeEvent.layout.width);

  const state = useRef({ width, top, onSwipe, onBusyChange, busy: false });
  state.current.width = width;
  state.current.top = top;
  state.current.onSwipe = onSwipe;
  state.current.onBusyChange = onBusyChange;

  const flyOut = useRef((action: SwipeAction, dy = 0) => {
    const s = state.current;
    if (s.busy || !s.top) return;
    const profile = s.top;
    s.busy = true;
    s.onBusyChange?.(true);
    Animated.timing(position, {
      toValue: { x: (action === 'like' ? 1 : -1) * s.width * 1.4, y: dy },
      duration: EXIT_DURATION,
      useNativeDriver: false,
    }).start(() => {
      s.busy = false;
      s.onBusyChange?.(false);
      s.onSwipe(profile, action);
    });
  }).current;

  useImperativeHandle(handle, () => ({ swipe: (action) => flyOut(action) }), [flyOut]);

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        // Capture so a horizontal drag wins over the photo tap zones underneath.
        onMoveShouldSetPanResponderCapture: (_, g) =>
          !state.current.busy && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.4 }),
        onPanResponderRelease: (_, g) => {
          const decision = decideSwipe(g.dx, g.vx, state.current.width);
          if (decision) flyOut(decision, g.dy * 0.4);
          else
            Animated.spring(position, {
              toValue: { x: 0, y: 0 },
              friction: 6,
              useNativeDriver: false,
            }).start();
        },
        onPanResponderTerminate: () =>
          Animated.spring(position, {
            toValue: { x: 0, y: 0 },
            friction: 6,
            useNativeDriver: false,
          }).start(),
      }),
    [position, flyOut],
  );

  useEffect(() => () => position.stopAnimation(), [position]);

  const w = width || 1;
  const rotate = position.x.interpolate({
    inputRange: [-w, 0, w],
    outputRange: [`${-rotationDeg(w, w)}deg`, '0deg', `${rotationDeg(w, w)}deg`],
    extrapolate: 'clamp',
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, w * SWIPE_DISTANCE_RATIO],
    outputRange: [0, 1],
    extrapolate: 'clamp',
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-w * SWIPE_DISTANCE_RATIO, 0],
    outputRange: [1, 0],
    extrapolate: 'clamp',
  });
  const behindScale = position.x.interpolate({
    inputRange: [-w, 0, w],
    outputRange: [1, 0.94, 1],
    extrapolate: 'clamp',
  });

  return (
    <View style={styles.stage} onLayout={onLayout}>
      {behind ? (
        <Animated.View
          style={[styles.layer, { transform: [{ scale: behindScale }] }]}
          pointerEvents="none"
          accessible={false}
        >
          <ProfileCard key={behind.userId} profile={behind} onOpen={() => undefined} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          style={[
            styles.layer,
            { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] },
          ]}
          {...panResponder.panHandlers}
        >
          <ProfileCard profile={top} onOpen={() => onOpen(top)} />
          <Animated.View
            style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}
            pointerEvents="none"
          >
            <Text variant="heading" color="success" weight="800">
              {fr.discover.stampLike}
            </Text>
          </Animated.View>
          <Animated.View
            style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}
            pointerEvents="none"
          >
            <Text variant="heading" color="danger" weight="800">
              {fr.discover.stampPass}
            </Text>
          </Animated.View>
        </Animated.View>
      ) : null}
    </View>
  );
}
