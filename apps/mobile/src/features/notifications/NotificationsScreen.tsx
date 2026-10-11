import { FlatList, Pressable, RefreshControl, View } from 'react-native';
import { useQueryClient } from '@tanstack/react-query';
import { errorMessage, queryKeys } from '../../api';
import type { AppNotification } from '../../api';
import { fr } from '../../i18n';
import { formatListTime } from '../../lib/date';
import type { RootScreenProps } from '../../navigation/types';
import type { ConversationsData } from '../../realtime/cache';
import { useStyles, useTheme } from '../../theme';
import { Button, EmptyState, ErrorState, Screen, Skeleton, Text, icons } from '../../ui';
import { useMarkAllNotificationsRead, useMarkNotificationRead, useNotifications } from './hooks';

export function NotificationsScreen({ navigation }: RootScreenProps<'Notifications'>) {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const { query, notifications, unreadCount } = useNotifications();
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();
  const styles = useStyles((t) => ({
    list: { paddingVertical: t.spacing.md, flexGrow: 1 },
    row: {
      flexDirection: 'row',
      gap: t.spacing.md,
      paddingVertical: t.spacing.md,
      minHeight: 64,
      alignItems: 'flex-start',
    },
    dot: { width: 10, height: 10, borderRadius: 5, marginTop: 7 },
    text: { flex: 1, gap: 2 },
    separator: { height: 1, backgroundColor: t.colors.border },
    top: { paddingBottom: t.spacing.sm },
  }));

  const open = (n: AppNotification) => {
    if (!n.isRead) markRead.mutate(n.id);
    const conversations = qc.getQueryData<ConversationsData>(queryKeys.conversations);
    const conversation = n.data.conversationId
      ? conversations?.pages
          .flatMap((p) => p.conversations)
          .find((c) => c.id === n.data.conversationId)
      : undefined;
    if (n.type === 'new_match' && conversation) {
      navigation.navigate('Chat', { conversation });
    } else if (n.data.userId) {
      navigation.navigate('ProfileDetail', { userId: n.data.userId, source: 'notification' });
    } else {
      navigation.navigate('Tabs', { screen: n.type === 'new_match' ? 'Messages' : 'Likes' });
    }
  };

  const renderItem = ({ item }: { item: AppNotification }) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={fr.notifications.itemLabel(item.title, item.body, item.isRead)}
      onPress={() => open(item)}
      style={styles.row}
    >
      <View
        style={[styles.dot, { backgroundColor: item.isRead ? 'transparent' : colors.primary }]}
        accessible={false}
      />
      <View style={styles.text}>
        <Text weight={item.isRead ? '500' : '800'}>{item.title}</Text>
        <Text variant="small" color="textMuted">
          {item.body}
        </Text>
      </View>
      <Text variant="caption" color="textMuted">
        {formatListTime(item.createdAt)}
      </Text>
    </Pressable>
  );

  let body;
  if (query.isPending) {
    body = (
      <View
        accessibilityRole="progressbar"
        accessibilityLabel={fr.common.loading}
        style={{ gap: 16, paddingTop: 16 }}
      >
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} height={52} radius={12} />
        ))}
      </View>
    );
  } else if (query.isError && notifications.length === 0) {
    body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  } else {
    body = (
      <FlatList
        data={notifications}
        keyExtractor={(n) => n.id}
        renderItem={renderItem}
        ItemSeparatorComponent={() => <View style={styles.separator} />}
        contentContainerStyle={styles.list}
        onEndReachedThreshold={0.5}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        refreshControl={
          <RefreshControl
            refreshing={query.isRefetching && !query.isFetchingNextPage}
            onRefresh={() => void query.refetch()}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        }
        ListHeaderComponent={
          unreadCount > 0 ? (
            <View style={styles.top}>
              <Button
                label={fr.notifications.markAll}
                variant="soft"
                size="sm"
                fullWidth={false}
                loading={markAll.isPending}
                onPress={() => markAll.mutate()}
              />
            </View>
          ) : null
        }
        ListEmptyComponent={
          <EmptyState
            glyph={icons.bell}
            title={fr.notifications.emptyTitle}
            message={fr.notifications.emptyMessage}
          />
        }
      />
    );
  }

  return <Screen edges={['bottom']}>{body}</Screen>;
}
