import React, { useCallback, useMemo } from 'react';
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native';
import type { CompositeScreenProps } from '@react-navigation/native';
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Ionicons } from '@expo/vector-icons';
import { discoverApi } from '../../api/discover';
import { errorMessage } from '../../api/errors';
import type { PublicProfile, SwipeAction } from '../../api/types';
import { announceSwipe, performSwipe, toSwipeTarget } from '../../features/swipe';
import { fr } from '../../i18n/fr';
import type { MainStackParamList, TabParamList } from '../../navigation/types';
import { qk } from '../../query/keys';
import { AuthImage, EmptyState, ErrorState, Skeleton, Text } from '../../shared/ui';
import { radius, spacing, useTheme } from '../../theme';

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, 'Likes'>,
  NativeStackScreenProps<MainStackParamList>
>;

const PAGE = 20;

const LikeTile = React.memo(function LikeTile({
  profile,
  onOpen,
  onSwipe,
  busy,
}: {
  profile: PublicProfile;
  onOpen: (p: PublicProfile) => void;
  onSwipe: (p: PublicProfile, a: SwipeAction) => void;
  busy: boolean;
}) {
  const { colors } = useTheme();
  return (
    <View style={styles.cell} testID={`like-tile-${profile.id}`}>
      <View style={[styles.tile, { backgroundColor: colors.surface, borderColor: colors.border }]}>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${fr.discover.card(profile.firstName, profile.age)}. ${fr.discover.openProfile}`}
          onPress={() => onOpen(profile)}
          testID={`like-open-${profile.id}`}
        >
          <AuthImage path={profile.photos[0]?.url} style={styles.photo} resizeMode="cover" />
          <View style={styles.caption}>
            <Text variant="bodyStrong" numberOfLines={1}>
              {profile.firstName}, {profile.age}
            </Text>
            <Text variant="caption" tone="muted" numberOfLines={1}>
              {profile.city}
            </Text>
          </View>
        </Pressable>
        <View style={styles.buttons}>
          <Pressable
            testID={`like-pass-${profile.id}`}
            accessibilityRole="button"
            accessibilityLabel={`${fr.likes.pass} ${profile.firstName}`}
            disabled={busy}
            onPress={() => onSwipe(profile, 'pass')}
            style={[styles.btn, { borderColor: colors.border, opacity: busy ? 0.5 : 1 }]}
          >
            <Ionicons name="close" size={22} color={colors.textMuted} />
          </Pressable>
          <Pressable
            testID={`like-back-${profile.id}`}
            accessibilityRole="button"
            accessibilityLabel={`${fr.likes.likeBack} ${profile.firstName}`}
            disabled={busy}
            onPress={() => onSwipe(profile, 'like')}
            style={[
              styles.btn,
              {
                backgroundColor: colors.primary,
                borderColor: colors.primary,
                opacity: busy ? 0.5 : 1,
              },
            ]}
          >
            <Ionicons name="heart" size={22} color={colors.onPrimary} />
          </Pressable>
        </View>
      </View>
    </View>
  );
});

export function LikesScreen({ navigation }: Props) {
  const qc = useQueryClient();
  const query = useInfiniteQuery({
    queryKey: qk.likes,
    initialPageParam: 0,
    queryFn: ({ pageParam, signal }) => discoverApi.likesReceived(PAGE, pageParam, signal),
    getNextPageParam: (last, all) => (last.length < PAGE ? undefined : all.length * PAGE),
  });

  const swipe = useMutation({
    mutationFn: ({ profile, action }: { profile: PublicProfile; action: SwipeAction }) =>
      performSwipe(qc, toSwipeTarget(profile), action),
    onSuccess: (_r, { profile, action }) => announceSwipe(profile.id, action),
  });

  const profiles = useMemo(() => query.data?.pages.flat() ?? [], [query.data]);
  const open = useCallback(
    (p: PublicProfile) => navigation.navigate('ProfileDetail', { userId: p.id, mode: 'swipe' }),
    [navigation],
  );
  const onSwipe = useCallback(
    (profile: PublicProfile, action: SwipeAction) => swipe.mutate({ profile, action }),
    [swipe],
  );
  const renderItem = useCallback(
    ({ item }: { item: PublicProfile }) => (
      <LikeTile
        profile={item}
        onOpen={open}
        onSwipe={onSwipe}
        busy={swipe.isPending && swipe.variables?.profile.id === item.id}
      />
    ),
    [open, onSwipe, swipe.isPending, swipe.variables],
  );

  if (query.isPending) {
    return (
      <View style={styles.skeletons} testID="likes-loading">
        {[0, 1, 2, 3].map((i) => (
          <View key={i} style={styles.cell}>
            <Skeleton height={230} rounded={radius.md} />
          </View>
        ))}
      </View>
    );
  }
  if (query.isError) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }

  return (
    <FlatList
      testID="likes-list"
      data={profiles}
      numColumns={2}
      keyExtractor={(p) => p.id}
      renderItem={renderItem}
      contentContainerStyle={profiles.length === 0 ? styles.emptyContent : styles.list}
      ListHeaderComponent={
        profiles.length > 0 ? (
          <Text variant="label" tone="muted" style={styles.header} testID="likes-count">
            {fr.likes.count(profiles.length)}
          </Text>
        ) : null
      }
      ListEmptyComponent={
        <EmptyState
          icon="heart-outline"
          title={fr.likes.emptyTitle}
          body={fr.likes.emptyBody}
          testID="likes-empty"
        />
      }
      onEndReached={() =>
        query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()
      }
      onEndReachedThreshold={0.5}
      refreshControl={
        <RefreshControl
          refreshing={query.isRefetching && !query.isFetchingNextPage}
          onRefresh={() => void query.refetch()}
        />
      }
      initialNumToRender={6}
      windowSize={7}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.md },
  emptyContent: { flexGrow: 1 },
  header: { marginHorizontal: spacing.xs, marginBottom: spacing.sm },
  skeletons: { flexDirection: 'row', flexWrap: 'wrap', padding: spacing.md },
  cell: { width: '50%', padding: spacing.xs },
  tile: { borderRadius: radius.md, borderWidth: 1, overflow: 'hidden' },
  photo: { width: '100%', aspectRatio: 3 / 4 },
  caption: { padding: spacing.sm },
  buttons: { flexDirection: 'row', justifyContent: 'space-around', paddingBottom: spacing.sm },
  btn: {
    width: 44,
    height: 44,
    borderRadius: 22,
    borderWidth: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
