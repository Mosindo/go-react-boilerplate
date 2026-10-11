import { FlatList, Pressable, View } from 'react-native';
import { RefreshControl } from 'react-native';
import { errorMessage, mediaUrl } from '../../api';
import type { PublicProfile } from '../../api';
import { fr } from '../../i18n';
import { formatDistance, initials } from '../../lib/format';
import type { TabScreenProps } from '../../navigation/types';
import { useStyles, useTheme } from '../../theme';
import {
  EmptyState,
  ErrorState,
  IconButton,
  RemoteImage,
  Screen,
  Skeleton,
  Text,
  icons,
} from '../../ui';
import { useLikes, useRespondToLike } from './hooks';

export function LikesScreen({ navigation }: TabScreenProps<'Likes'>) {
  const { colors } = useTheme();
  const likes = useLikes();
  const respond = useRespondToLike();
  const styles = useStyles((t) => ({
    header: { paddingVertical: t.spacing.md, gap: t.spacing.xs },
    list: { paddingBottom: t.spacing.xl, gap: t.spacing.md, flexGrow: 1 },
    row: { gap: t.spacing.md },
    cell: { flex: 1, maxWidth: '50%' },
    card: {
      borderRadius: t.radius.lg,
      overflow: 'hidden',
      backgroundColor: t.colors.surface,
      borderWidth: 1,
      borderColor: t.colors.border,
    },
    photo: { width: '100%', aspectRatio: 3 / 4 },
    info: { padding: t.spacing.md, gap: 2 },
    actions: { flexDirection: 'row', justifyContent: 'space-around', paddingBottom: t.spacing.md },
  }));

  const renderItem = ({ item }: { item: PublicProfile }) => {
    const distance = formatDistance(item.distanceKm);
    const sub = [item.city, distance].filter(Boolean).join(' · ');
    return (
      <View style={styles.cell}>
        <View style={styles.card}>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={fr.likes.openProfile(item.firstName, item.age)}
            onPress={() =>
              navigation.navigate('ProfileDetail', {
                userId: item.userId,
                profile: item,
                source: 'likes',
              })
            }
          >
            <RemoteImage
              uri={item.photos[0] ? mediaUrl(item.photos[0].url) : null}
              fallbackText={initials(item.firstName)}
              style={styles.photo}
            />
            <View style={styles.info}>
              <Text weight="700" numberOfLines={1}>
                {item.firstName}, {item.age}
              </Text>
              {sub ? (
                <Text variant="caption" color="textMuted" numberOfLines={1}>
                  {sub}
                </Text>
              ) : null}
            </View>
          </Pressable>
          <View style={styles.actions}>
            <IconButton
              glyph={icons.close}
              accessibilityLabel={fr.likes.pass(item.firstName)}
              tone="outline"
              color="danger"
              onPress={() => respond(item, 'pass')}
            />
            <IconButton
              glyph={icons.heart}
              accessibilityLabel={fr.likes.likeBack(item.firstName)}
              tone="primary"
              onPress={() => respond(item, 'like')}
              testID={`like-back-${item.userId}`}
            />
          </View>
        </View>
      </View>
    );
  };

  let body;
  if (likes.isPending) {
    body = (
      <View
        style={styles.row}
        accessibilityRole="progressbar"
        accessibilityLabel={fr.common.loading}
      >
        {[0, 1, 2].map((i) => (
          <View key={i} style={{ flexDirection: 'row', gap: 12 }}>
            <Skeleton height={260} radius={20} style={{ flex: 1 }} />
            <Skeleton height={260} radius={20} style={{ flex: 1 }} />
          </View>
        ))}
      </View>
    );
  } else if (likes.isError && !likes.data) {
    body = <ErrorState message={errorMessage(likes.error)} onRetry={() => void likes.refetch()} />;
  } else {
    body = (
      <FlatList
        data={likes.data}
        keyExtractor={(item) => item.userId}
        numColumns={2}
        columnWrapperStyle={styles.row}
        contentContainerStyle={styles.list}
        renderItem={renderItem}
        refreshControl={
          <RefreshControl
            refreshing={likes.isRefetching}
            onRefresh={() => void likes.refetch()}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        }
        ListEmptyComponent={
          <EmptyState
            glyph={icons.heart}
            title={fr.likes.emptyTitle}
            message={fr.likes.emptyMessage}
          />
        }
      />
    );
  }

  return (
    <Screen edges={['top']}>
      <View style={styles.header}>
        <Text variant="title">{fr.likes.title}</Text>
        <Text color="textMuted">{fr.likes.subtitle}</Text>
      </View>
      {body}
    </Screen>
  );
}
