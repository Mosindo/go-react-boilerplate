import React, { useCallback, useState } from 'react';
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native';
import type { CompositeScreenProps } from '@react-navigation/native';
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { chatApi } from '../../api/chat';
import { errorMessage } from '../../api/errors';
import type { Conversation } from '../../api/types';
import { fr } from '../../i18n/fr';
import type { MainStackParamList, TabParamList } from '../../navigation/types';
import { useMe } from '../../auth/AuthProvider';
import { useConversations } from '../../query/hooks';
import { removeConversation } from '../../realtime/cacheUpdaters';
import {
  Avatar,
  Badge,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  IconButton,
  Skeleton,
  Text,
} from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { spacing, useTheme } from '../../theme';
import { formatListTime } from '../../utils/format';

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, 'Messages'>,
  NativeStackScreenProps<MainStackParamList>
>;

const Row = React.memo(function Row({
  conversation,
  myUserId,
  onOpen,
  onMenu,
}: {
  conversation: Conversation;
  myUserId: string | undefined;
  onOpen: (c: Conversation) => void;
  onMenu: (c: Conversation) => void;
}) {
  const { colors } = useTheme();
  const { user, lastMessage, unreadCount } = conversation;
  const unread = unreadCount > 0;
  const preview = lastMessage
    ? `${lastMessage.senderId === myUserId ? fr.messages.you : ''}${lastMessage.body}`
    : fr.messages.noMessageYet;
  return (
    <Pressable
      testID={`conversation-${conversation.id}`}
      accessibilityRole="button"
      accessibilityLabel={`${user.firstName}. ${preview}.${unread ? ` ${fr.messages.unread(unreadCount)}.` : ''}`}
      onPress={() => onOpen(conversation)}
      onLongPress={() => onMenu(conversation)}
      style={({ pressed }) => [
        styles.row,
        { borderColor: colors.border },
        pressed ? { backgroundColor: colors.surfaceAlt } : null,
      ]}
    >
      <Avatar name={user.firstName} path={user.mainPhotoUrl} size={56} />
      <View style={styles.body}>
        <View style={styles.line}>
          <Text variant={unread ? 'bodyStrong' : 'body'} numberOfLines={1} style={styles.name}>
            {user.firstName}
          </Text>
          <Text variant="caption" tone={unread ? 'primary' : 'muted'}>
            {formatListTime(lastMessage?.createdAt ?? conversation.updatedAt)}
          </Text>
        </View>
        <View style={styles.line}>
          <Text
            variant="caption"
            tone={unread ? 'default' : 'muted'}
            numberOfLines={1}
            style={[styles.name, unread ? { fontWeight: '700' } : null]}
          >
            {preview}
          </Text>
          <Badge count={unreadCount} testID={`conversation-badge-${conversation.id}`} />
        </View>
      </View>
      <IconButton
        icon="ellipsis-horizontal"
        label={`${fr.messages.rowMenu} ${user.firstName}`}
        size={20}
        color={colors.textMuted}
        onPress={() => onMenu(conversation)}
        testID={`conversation-menu-${conversation.id}`}
      />
    </Pressable>
  );
});

export function MessagesScreen({ navigation }: Props) {
  const qc = useQueryClient();
  const { data, isPending, isError, error, refetch, isRefetching } = useConversations();
  const { data: me } = useMe();
  const [menuFor, setMenuFor] = useState<Conversation | null>(null);

  const remove = useMutation({
    mutationFn: (id: string) => chatApi.remove(id),
    onSuccess: (_r, id) => {
      removeConversation(qc, id);
      toast.success(fr.messages.deleted);
    },
    onSettled: () => setMenuFor(null),
  });

  const open = useCallback(
    (c: Conversation) => navigation.navigate('Chat', { conversationId: c.id, user: c.user }),
    [navigation],
  );

  if (isPending) {
    return (
      <View style={styles.skeletons} testID="messages-loading">
        {[0, 1, 2, 3, 4].map((i) => (
          <View key={i} style={styles.skeletonRow}>
            <Skeleton width={56} height={56} rounded={28} />
            <View style={styles.skeletonText}>
              <Skeleton width="50%" height={14} />
              <Skeleton width="80%" height={12} style={{ marginTop: spacing.sm }} />
            </View>
          </View>
        ))}
      </View>
    );
  }
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  return (
    <>
      <FlatList
        testID="conversation-list"
        data={data}
        keyExtractor={(c) => c.id}
        renderItem={({ item }) => (
          <Row conversation={item} myUserId={me?.id} onOpen={open} onMenu={setMenuFor} />
        )}
        contentContainerStyle={data.length === 0 ? styles.emptyContent : undefined}
        ListEmptyComponent={
          <EmptyState
            icon="chatbubbles-outline"
            title={fr.messages.emptyTitle}
            body={fr.messages.emptyBody}
            actionLabel={fr.messages.discoverCta}
            onAction={() => navigation.navigate('Discover')}
            testID="messages-empty"
          />
        }
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} />
        }
        initialNumToRender={12}
        windowSize={7}
      />
      <ConfirmDialog
        visible={menuFor !== null}
        title={fr.messages.deleteTitle}
        body={fr.messages.deleteBody}
        confirmLabel={fr.common.delete}
        destructive
        loading={remove.isPending}
        onCancel={() => setMenuFor(null)}
        onConfirm={() => menuFor && remove.mutate(menuFor.id)}
      />
    </>
  );
}

const styles = StyleSheet.create({
  emptyContent: { flexGrow: 1 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  body: { flex: 1, marginHorizontal: spacing.md },
  line: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: 22,
  },
  name: { flex: 1, marginRight: spacing.sm },
  skeletons: { padding: spacing.lg },
  skeletonRow: { flexDirection: 'row', alignItems: 'center', marginBottom: spacing.lg },
  skeletonText: { flex: 1, marginLeft: spacing.md },
});
