import React, {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  Animated,
  PanResponder,
  Pressable,
  StyleSheet,
  View,
  type LayoutChangeEvent,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import type { PublicProfile, SwipeAction } from '../../api/types';
import { fr } from '../../i18n/fr';
import { AuthImage, Chip, Text } from '../../shared/ui';
import { radius, spacing, useTheme } from '../../theme';

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profile: PublicProfile;
  interactive: boolean;
  onSwiped: (action: SwipeAction) => void;
  onOpenProfile: () => void;
};

const SWIPE_THRESHOLD_RATIO = 0.25;

function SwipeCardImpl(
  { profile, interactive, onSwiped, onOpenProfile }: Props,
  ref: React.Ref<SwipeCardHandle>,
) {
  const { colors } = useTheme();
  const [photoIndex, setPhotoIndex] = useState(0);
  const pan = useRef(new Animated.ValueXY()).current;
  const width = useRef(360);
  const done = useRef(false);
  const latest = useRef({ onSwiped, interactive });
  latest.current = { onSwiped, interactive };

  const photos = profile.photos;
  const current = photos[Math.min(photoIndex, Math.max(photos.length - 1, 0))];

  const fly = useCallback(
    (action: SwipeAction) => {
      if (done.current) return;
      done.current = true;
      Animated.timing(pan, {
        toValue: { x: (action === 'like' ? 1 : -1) * width.current * 1.4, y: 0 },
        duration: 220,
        useNativeDriver: false,
      }).start(() => latest.current.onSwiped(action));
    },
    [pan],
  );

  const reset = useCallback(
    () =>
      Animated.spring(pan, {
        toValue: { x: 0, y: 0 },
        friction: 7,
        useNativeDriver: false,
      }).start(),
    [pan],
  );

  useImperativeHandle(ref, () => ({ swipe: fly }), [fly]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onStartShouldSetPanResponder: () => false,
        onMoveShouldSetPanResponder: (_e, g) =>
          latest.current.interactive &&
          !done.current &&
          Math.abs(g.dx) > 8 &&
          Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderGrant: () => pan.extractOffset(),
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], {
          useNativeDriver: false,
        }),
        onPanResponderRelease: (_e, g) => {
          pan.flattenOffset();
          const limit = width.current * SWIPE_THRESHOLD_RATIO;
          if (g.dx > limit || g.vx > 0.9) fly('like');
          else if (g.dx < -limit || g.vx < -0.9) fly('pass');
          else reset();
        },
        onPanResponderTerminate: () => {
          pan.flattenOffset();
          reset();
        },
      }),
    [pan, fly, reset],
  );

  const rotate = pan.x.interpolate({
    inputRange: [-300, 0, 300],
    outputRange: ['-8deg', '0deg', '8deg'],
  });
  const likeOpacity = pan.x.interpolate({
    inputRange: [0, 100],
    outputRange: [0, 1],
    extrapolate: 'clamp',
  });
  const passOpacity = pan.x.interpolate({
    inputRange: [-100, 0],
    outputRange: [1, 0],
    extrapolate: 'clamp',
  });

  const onLayout = (e: LayoutChangeEvent) => {
    width.current = e.nativeEvent.layout.width;
  };

  const next = () => setPhotoIndex((i) => Math.min(i + 1, photos.length - 1));
  const prev = () => setPhotoIndex((i) => Math.max(i - 1, 0));
  const place = [
    profile.city,
    profile.distanceKm !== null ? fr.common.kmAway(profile.distanceKm) : null,
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <Animated.View
      testID={interactive ? 'swipe-card' : undefined}
      accessibilityElementsHidden={!interactive}
      importantForAccessibility={interactive ? 'auto' : 'no-hide-descendants'}
      onLayout={onLayout}
      {...(interactive ? responder.panHandlers : {})}
      style={[
        styles.card,
        { backgroundColor: colors.surfaceAlt },
        interactive
          ? { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }
          : styles.behind,
      ]}
    >
      <AuthImage
        path={current?.url}
        style={StyleSheet.absoluteFill}
        resizeMode="cover"
        accessibilityLabel={fr.photos.photoOf(
          profile.firstName,
          Math.min(photoIndex + 1, photos.length),
          photos.length,
        )}
        testID={interactive ? 'card-photo' : undefined}
      />

      {photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {photos.map((p, i) => (
            <View
              key={p.id}
              style={[
                styles.dot,
                { backgroundColor: i === photoIndex ? '#FFFFFF' : 'rgba(255,255,255,0.4)' },
              ]}
            />
          ))}
        </View>
      ) : null}

      {interactive ? (
        <View style={styles.zones}>
          <Pressable
            testID="card-prev"
            accessibilityRole="button"
            accessibilityLabel={fr.discover.previousPhoto}
            onPress={prev}
            disabled={!interactive}
            style={styles.zoneSide}
          />
          <Pressable
            testID="card-open"
            accessibilityRole="button"
            accessibilityLabel={fr.discover.openProfile}
            onPress={onOpenProfile}
            disabled={!interactive}
            style={styles.zoneCenter}
          />
          <Pressable
            testID="card-next"
            accessibilityRole="button"
            accessibilityLabel={fr.discover.nextPhoto}
            onPress={next}
            disabled={!interactive}
            style={styles.zoneSide}
          />
        </View>
      ) : null}

      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${fr.discover.card(profile.firstName, profile.age)}. ${fr.discover.openProfile}`}
        onPress={onOpenProfile}
        disabled={!interactive}
        style={styles.info}
      >
        <View style={styles.nameRow}>
          <Text
            variant="title"
            style={styles.white}
            numberOfLines={1}
            testID={interactive ? 'card-name' : undefined}
          >
            {profile.firstName}, {profile.age}
          </Text>
          <Ionicons name="information-circle-outline" size={24} color="#FFFFFF" />
        </View>
        {place ? (
          <Text variant="caption" style={styles.whiteSoft} numberOfLines={1}>
            {place}
          </Text>
        ) : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.slice(0, 3).map((i) => (
              <Chip key={i.slug} label={i.label} tone="overlay" />
            ))}
          </View>
        ) : null}
      </Pressable>

      <Animated.View
        pointerEvents="none"
        style={[
          styles.stamp,
          styles.stampLike,
          { opacity: likeOpacity, borderColor: colors.success },
        ]}
      >
        <Text variant="title" style={{ color: '#FFFFFF' }}>
          {fr.discover.like.toUpperCase()}
        </Text>
      </Animated.View>
      <Animated.View
        pointerEvents="none"
        style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}
      >
        <Text variant="title" style={{ color: '#FFFFFF' }}>
          {fr.discover.pass.toUpperCase()}
        </Text>
      </Animated.View>
    </Animated.View>
  );
}

export const SwipeCard = React.memo(forwardRef(SwipeCardImpl));

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radius.lg,
    overflow: 'hidden',
  },
  behind: { transform: [{ scale: 0.96 }], opacity: 0.9 },
  dots: {
    position: 'absolute',
    top: spacing.sm,
    left: spacing.md,
    right: spacing.md,
    flexDirection: 'row',
    gap: 4,
  },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  zones: { ...StyleSheet.absoluteFillObject, flexDirection: 'row', bottom: 130 },
  zoneSide: { flex: 3 },
  zoneCenter: { flex: 4 },
  info: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    backgroundColor: 'rgba(20,8,26,0.55)',
  },
  nameRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  white: { color: '#FFFFFF', flex: 1 },
  whiteSoft: { color: 'rgba(255,255,255,0.85)', marginTop: 2 },
  chips: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    marginTop: spacing.sm,
    marginBottom: -spacing.sm,
  },
  stamp: {
    position: 'absolute',
    top: spacing.xl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderWidth: 3,
    borderRadius: radius.sm,
  },
  stampLike: {
    left: spacing.xl,
    transform: [{ rotate: '-12deg' }],
    backgroundColor: 'rgba(46,125,79,0.55)',
  },
  stampPass: {
    right: spacing.xl,
    transform: [{ rotate: '12deg' }],
    borderColor: '#FFFFFF',
    backgroundColor: 'rgba(20,8,26,0.55)',
  },
});
