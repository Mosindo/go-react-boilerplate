import React, { useCallback, useEffect, useRef, useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { useFocusEffect, type CompositeScreenProps } from '@react-navigation/native';
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useQueryClient } from '@tanstack/react-query';
import { discoverApi } from '../../api/discover';
import { errorMessage, isApiError } from '../../api/errors';
import type { PublicProfile, SwipeAction } from '../../api/types';
import { performSwipe, swipeBus, toSwipeTarget } from '../../features/swipe';
import { fr } from '../../i18n/fr';
import type { MainStackParamList, TabParamList } from '../../navigation/types';
import {
  EmptyState,
  ErrorState,
  LoadingView,
  RoundButton,
  Screen,
  Skeleton,
} from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { bumpDiscoverEpoch, discoverEpoch } from '../../store/appState';
import { spacing, useTheme } from '../../theme';
import { SwipeCard, type SwipeCardHandle } from './SwipeCard';

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, 'Discover'>,
  NativeStackScreenProps<MainStackParamList>
>;

const BATCH = 10;
const PREFETCH_BELOW = 3;

export function DiscoverScreen({ navigation }: Props) {
  const qc = useQueryClient();
  const { colors } = useTheme();
  const epoch = discoverEpoch.use();
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [loadedOnce, setLoadedOnce] = useState(false);
  const [exhausted, setExhausted] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);
  const generation = useRef(0);
  const cardRef = useRef<SwipeCardHandle>(null);

  const fetchMore = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    const gen = generation.current;
    try {
      const batch = await discoverApi.discover(BATCH);
      if (gen !== generation.current) return;
      const fresh = batch.filter((p) => !seen.current.has(p.id));
      fresh.forEach((p) => seen.current.add(p.id));
      setQueue((q) => [...q, ...fresh]);
      setExhausted(fresh.length === 0);
      setError(null);
    } catch (err) {
      if (gen === generation.current) setError(err);
    } finally {
      fetching.current = false;
      if (gen === generation.current) setLoadedOnce(true);
    }
  }, []);

  const reset = useCallback(() => {
    generation.current += 1;
    fetching.current = false;
    seen.current = new Set();
    setQueue([]);
    setExhausted(false);
    setError(null);
    setLoadedOnce(false);
  }, []);

  // Rebuild the deck when criteria change (epoch) - also runs on mount.
  useEffect(() => {
    reset();
  }, [epoch, reset]);

  // Prefetch the next batch when fewer than 3 cards remain.
  useEffect(() => {
    if (!exhausted && !error && queue.length < PREFETCH_BELOW) void fetchMore();
  }, [queue.length, exhausted, error, fetchMore, loadedOnce, epoch]);

  // The profile screen may already have swiped the current card.
  const bus = swipeBus.use();
  useEffect(() => {
    if (bus) setQueue((q) => q.filter((p) => p.id !== bus.userId));
  }, [bus]);

  const top = queue[0];

  // Coming back to an empty deck (new people may have joined meanwhile): look again.
  const emptyRef = useRef(false);
  emptyRef.current = !top && exhausted;
  useFocusEffect(
    useCallback(() => {
      if (emptyRef.current) bumpDiscoverEpoch();
    }, []),
  );

  const handleSwiped = useCallback(
    (profile: PublicProfile, action: SwipeAction) => {
      setQueue((q) => q.filter((p) => p.id !== profile.id));
      performSwipe(qc, toSwipeTarget(profile), action).catch((err) => {
        if (isApiError(err, 'not_found')) {
          toast.error(fr.discover.profileUnavailable);
          return;
        }
        toast.error(`${fr.discover.swipeFailed} ${errorMessage(err)}`);
        setQueue((q) => (q.some((p) => p.id === profile.id) ? q : [profile, ...q]));
      });
    },
    [qc],
  );

  const open = useCallback(
    (profile: PublicProfile) =>
      navigation.navigate('ProfileDetail', { userId: profile.id, mode: 'swipe' }),
    [navigation],
  );

  let body: React.ReactNode;
  if (top) {
    const second = queue[1];
    body = (
      <>
        {second ? (
          <SwipeCard
            key={second.id}
            profile={second}
            interactive={false}
            onSwiped={noop}
            onOpenProfile={noop}
          />
        ) : null}
        <SwipeCard
          key={top.id}
          ref={cardRef}
          profile={top}
          interactive
          onSwiped={(action) => handleSwiped(top, action)}
          onOpenProfile={() => open(top)}
        />
      </>
    );
  } else if (error) {
    body = (
      <ErrorState
        message={errorMessage(error)}
        onRetry={() => {
          setError(null);
          void fetchMore();
        }}
      />
    );
  } else if (!loadedOnce || !exhausted) {
    body = (
      <View style={styles.skeleton} testID="discover-loading">
        <Skeleton height={400} rounded={20} style={styles.skeletonCard} />
        <LoadingView />
      </View>
    );
  } else {
    body = (
      <EmptyState
        testID="discover-empty"
        icon="compass-outline"
        title={fr.discover.emptyTitle}
        body={fr.discover.emptyBody}
        actionLabel={fr.common.refresh}
        onAction={bumpDiscoverEpoch}
      />
    );
  }

  return (
    <Screen padded={false} contentStyle={styles.screen} testID="discover-screen">
      <View style={styles.deck}>{body}</View>
      <View style={styles.actions}>
        <RoundButton
          testID="discover-pass"
          icon="close"
          label={fr.discover.pass}
          color={colors.textMuted}
          disabled={!top}
          onPress={() => cardRef.current?.swipe('pass')}
        />
        <RoundButton
          testID="discover-like"
          icon="heart"
          label={fr.discover.like}
          color={colors.primary}
          filled
          disabled={!top}
          onPress={() => cardRef.current?.swipe('like')}
        />
      </View>
    </Screen>
  );
}

function noop() {}

const styles = StyleSheet.create({
  screen: { paddingHorizontal: spacing.lg, paddingTop: spacing.sm },
  deck: { flex: 1, marginBottom: spacing.md, maxHeight: 720 },
  skeleton: { flex: 1 },
  skeletonCard: { ...StyleSheet.absoluteFillObject, height: undefined },
  actions: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: spacing.xxl,
    paddingBottom: spacing.lg,
  },
});
