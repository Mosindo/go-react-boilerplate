import { useState } from 'react';
import { FlatList, Pressable, RefreshControl, ScrollView, View } from 'react-native';
import { errorMessage } from '../../api';
import type { Conversation } from '../../api';
import { fr } from '../../i18n';
import { formatListTime } from '../../lib/date';
import type { TabScreenProps } from '../../navigation/types';
import { useStyles, useTheme } from '../../theme';
import {
  ActionSheet,
  Avatar,
  Badge,
  EmptyState,
  ErrorState,
  IconButton,
  Screen,
  Skeleton,
  Text,
  icons,
  useToast,
} from '../../ui';
import { useCurrentUser } from '../auth/AuthContext';
import { useConversations, useHideConversation } from './hooks';

export function ConversationsScreen({ navigation }: TabScreenProps<'Messages'>) {
  const { colors } = useTheme();
  const me = useCurrentUser();
  const toast = useToast();
  const { query, conversations } = useConversations();
  const hide = useHideConversation();
  const [menuFor, setMenuFor] = useState<Conversation | null>(null);

  const styles = useStyles((t) => ({
    header: { paddingVertical: t.spacing.md, gap: t.spacing.xs },
    list: { paddingBottom: t.spacing.xl, flexGrow: 1 },
    section: { gap: t.spacing.md, paddingVertical: t.spacing.md },
    matches: { gap: t.spacing.lg, paddingRight: t.spacing.lg },
    match: { alignItems: 'center', width: 76, gap: t.spacing.xs, minHeight: 44 },
    row: {
      flexDirection: 'row',
      alignItems: 'center',
      gap: t.spacing.md,
      paddingVertical: t.spacing.md,
      minHeight: 72,
    },
    rowText: { flex: 1, gap: 2 },
    rowTop: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      alignItems: 'center',
      gap: t.spacing.sm,
    },
    rowBottom: { flexDirection: 'row', alignItems: 'center', gap: t.spacing.sm },
    separator: { height: 1, backgroundColor: t.colors.border },
  }));

  const openChat = (conversation: Conversation) => navigation.navigate('Chat', { conversation });
  const newMatches = conversations.filter((c) => c.lastMessage === null);
  const active = conversations.filter((c) => c.lastMessage !== null);

  const renderRow = ({ item }: { item: Conversation }) => {
    const last = item.lastMessage;
    const unread = item.unreadCount > 0;
    const preview = last
      ? last.senderId === me.id
        ? fr.chat.youPrefix(last.body)
        : last.body
      : '';
    return (
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={fr.chat.rowLabel(item.user.firstName, preview, item.unreadCount)}
        accessibilityHint={fr.chat.rowHint}
        onPress={() => openChat(item)}
        onLongPress={() => setMenuFor(item)}
        style={styles.row}
        testID={`conversation-${item.id}`}
      >
        <Avatar url={item.user.photoUrl} name={item.user.firstName} size={56} />
        <View style={styles.rowText}>
          <View style={styles.rowTop}>
            <Text weight={unread ? '800' : '600'} numberOfLines={1} style={{ flexShrink: 1 }}>
              {item.user.firstName}
            </Text>
            {last ? (
              <Text variant="caption" color={unread ? 'primary' : 'textMuted'}>
                {formatListTime(last.createdAt)}
              </Text>
            ) : null}
          </View>
          <View style={styles.rowBottom}>
            <Text
              variant="small"
              color={unread ? 'text' : 'textMuted'}
              weight={unread ? '700' : '400'}
              numberOfLines={1}
              style={{ flex: 1 }}
            >
              {preview}
            </Text>
            <Badge count={item.unreadCount} />
          </View>
        </View>
        <IconButton
          glyph={icons.more}
          accessibilityLabel={fr.chat.moreFor(item.user.firstName)}
          onPress={() => setMenuFor(item)}
        />
      </Pressable>
    );
  };

  const header = (
    <>
      {newMatches.length > 0 ? (
        <View style={styles.section}>
          <Text variant="label" color="textMuted">
            {fr.chat.newMatches}
          </Text>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.matches}
          >
            {newMatches.map((c) => (
              <Pressable
                key={c.id}
                style={styles.match}
                accessibilityRole="button"
                accessibilityLabel={fr.chat.newMatchLabel(c.user.firstName)}
                onPress={() => openChat(c)}
                onLongPress={() => setMenuFor(c)}
              >
                <Avatar url={c.user.photoUrl} name={c.user.firstName} size={64} />
                <Text variant="caption" weight="600" numberOfLines={1}>
                  {c.user.firstName}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>
      ) : null}
      {active.length > 0 ? (
        <Text variant="label" color="textMuted" style={{ marginTop: 8 }}>
          {fr.chat.conversations}
        </Text>
      ) : null}
    </>
  );

  let body;
  if (query.isPending) {
    body = (
      <View accessibilityRole="progressbar" accessibilityLabel={fr.common.loading}>
        {[0, 1, 2, 3, 4].map((i) => (
          <View key={i} style={styles.row}>
            <Skeleton width={56} height={56} radius={28} />
            <View style={{ flex: 1, gap: 8 }}>
              <Skeleton width="40%" height={14} />
              <Skeleton width="80%" height={12} />
            </View>
          </View>
        ))}
      </View>
    );
  } else if (query.isError && conversations.length === 0) {
    body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  } else {
    body = (
      <FlatList
        data={active}
        keyExtractor={(c) => c.id}
        renderItem={renderRow}
        ItemSeparatorComponent={() => <View style={styles.separator} />}
        ListHeaderComponent={header}
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
        ListEmptyComponent={
          newMatches.length === 0 ? (
            <EmptyState
              glyph={icons.chat}
              title={fr.chat.emptyTitle}
              message={fr.chat.emptyMessage}
            />
          ) : (
            <Text color="textMuted" align="center" style={{ paddingVertical: 32 }}>
              {fr.chat.noMessagesYet}
            </Text>
          )
        }
      />
    );
  }

  return (
    <Screen edges={['top']}>
      <View style={styles.header}>
        <Text variant="title">{fr.chat.title}</Text>
      </View>
      {body}
      <ActionSheet
        visible={menuFor !== null}
        onClose={() => setMenuFor(null)}
        title={menuFor?.user.firstName}
        actions={
          menuFor
            ? [
                {
                  label: fr.chat.viewProfile,
                  onPress: () =>
                    navigation.navigate('ProfileDetail', {
                      userId: menuFor.user.userId,
                      source: 'chat',
                    }),
                },
                {
                  label: fr.chat.hide,
                  variant: 'danger',
                  onPress: () =>
                    hide.mutate(menuFor.id, { onSuccess: () => toast.success(fr.chat.hidden) }),
                },
              ]
            : []
        }
      />
    </Screen>
  );
}
