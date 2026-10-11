import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  View,
} from 'react-native';
import { useIsFocused } from '@react-navigation/native';
import { useQueryClient } from '@tanstack/react-query';
import { chatApi, errorMessage, queryKeys } from '../../api';
import type { Message } from '../../api';
import { fr } from '../../i18n';
import { buildChatItems } from '../../lib/date';
import type { ChatListItem } from '../../lib/date';
import { useErrorToast } from '../../lib/useErrorToast';
import { tempId } from '../../lib/uniqueId';
import type { RootScreenProps } from '../../navigation/types';
import { setActiveConversation } from '../../realtime/activeConversation';
import { conversationsWithMessage, messagesWith } from '../../realtime/cache';
import type { ConversationsData, MessagesData } from '../../realtime/cache';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useStyles, useTheme } from '../../theme';
import { Avatar, EmptyState, ErrorState, IconButton, Skeleton, Text, icons } from '../../ui';
import { useCurrentUser } from '../auth/AuthContext';
import { UserActionsSheet } from '../safety/UserActionsSheet';
import { MessageBubble } from './MessageBubble';
import { MessageComposer } from './MessageComposer';
import { useMarkRead, useMessages } from './hooks';

interface PendingMessage {
  localId: string;
  body: string;
  createdAt: string;
  failed: boolean;
}

type Row = ChatListItem | { kind: 'pending'; key: string; pending: PendingMessage };

export function ChatScreen({ navigation, route }: RootScreenProps<'Chat'>) {
  const { conversation } = route.params;
  const peer = conversation.user;
  const me = useCurrentUser();
  const qc = useQueryClient();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const onError = useErrorToast();
  const focused = useIsFocused();
  const markRead = useMarkRead(conversation.id);
  const { query, messages, otherLastReadAt } = useMessages(conversation.id);
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [menuOpen, setMenuOpen] = useState(false);
  const sending = useRef(new Set<string>());

  const styles = useStyles((t) => ({
    flex: { flex: 1, backgroundColor: t.colors.bg },
    list: { paddingHorizontal: t.spacing.lg, paddingVertical: t.spacing.md },
    day: {
      alignSelf: 'center',
      marginVertical: t.spacing.md,
      paddingHorizontal: t.spacing.md,
      paddingVertical: 4,
      borderRadius: t.radius.pill,
      backgroundColor: t.colors.surfaceAlt,
    },
    composer: { paddingHorizontal: t.spacing.lg },
    headerTitle: { flexDirection: 'row', alignItems: 'center', gap: t.spacing.sm, minHeight: 44 },
    skeletons: { padding: t.spacing.lg, gap: t.spacing.md },
  }));

  // Header: peer avatar + name (opens profile) and the "…" menu.
  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          style={styles.headerTitle}
          accessibilityRole="button"
          accessibilityLabel={fr.chat.viewProfileOf(peer.firstName)}
          onPress={() =>
            navigation.navigate('ProfileDetail', { userId: peer.userId, source: 'chat' })
          }
        >
          <Avatar url={peer.photoUrl} name={peer.firstName} size={36} />
          <Text weight="700" numberOfLines={1}>
            {peer.firstName}
          </Text>
        </Pressable>
      ),
      headerRight: () => (
        <IconButton
          glyph={icons.more}
          accessibilityLabel={fr.chat.menu}
          onPress={() => setMenuOpen(true)}
        />
      ),
    });
  }, [navigation, peer, styles.headerTitle]);

  // Tell the realtime layer which chat is open (no unread badge for it).
  useEffect(() => {
    if (!focused) return;
    setActiveConversation(conversation.id);
    return () => setActiveConversation(null);
  }, [focused, conversation.id]);

  // Mark as read on focus and whenever a new incoming message lands while we look at it.
  const latestIncomingId = messages.find((m) => m.senderId !== me.id)?.id;
  useEffect(() => {
    if (focused && !query.isPending) markRead();
  }, [focused, latestIncomingId, query.isPending, markRead]);

  const recordSent = useCallback(
    (message: Message) => {
      qc.setQueryData<MessagesData>(queryKeys.messages(conversation.id), (old) =>
        old ? messagesWith(old, message) : old,
      );
      qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
        old ? conversationsWithMessage(old, message, me.id, true) : old,
      );
    },
    [qc, conversation.id, me.id],
  );

  const deliver = useCallback(
    (item: PendingMessage) => {
      if (sending.current.has(item.localId)) return;
      sending.current.add(item.localId);
      setPending((list) =>
        list.map((p) => (p.localId === item.localId ? { ...p, failed: false } : p)),
      );
      chatApi
        .send(conversation.id, item.body)
        .then((message) => {
          recordSent(message);
          setPending((list) => list.filter((p) => p.localId !== item.localId));
        })
        .catch((error: unknown) => {
          setPending((list) =>
            list.map((p) => (p.localId === item.localId ? { ...p, failed: true } : p)),
          );
          onError(error);
        })
        .finally(() => sending.current.delete(item.localId));
    },
    [conversation.id, recordSent, onError],
  );

  const send = useCallback(
    (body: string) => {
      const item: PendingMessage = {
        localId: tempId('msg'),
        body,
        createdAt: new Date().toISOString(),
        failed: false,
      };
      setPending((list) => [item, ...list]);
      deliver(item);
    },
    [deliver],
  );

  // Own latest message gets the delivery line ("Lu" once the other side read it).
  const lastMine = messages.find((m) => m.senderId === me.id);
  const lastMineId = lastMine?.id;
  const lastMineRead = !!(lastMine && otherLastReadAt && lastMine.createdAt <= otherLastReadAt);

  const rows = useMemo<Row[]>(() => {
    const pendingRows: Row[] = pending.map((p) => ({
      kind: 'pending',
      key: p.localId,
      pending: p,
    }));
    return [...pendingRows, ...buildChatItems(messages)];
  }, [pending, messages]);

  const renderItem = ({ item }: { item: Row }) => {
    if (item.kind === 'day') {
      return (
        <View style={styles.day}>
          <Text variant="caption" color="textMuted" weight="600">
            {item.label}
          </Text>
        </View>
      );
    }
    if (item.kind === 'pending') {
      const p = item.pending;
      return (
        <MessageBubble
          mine
          body={p.body}
          createdAt={p.createdAt}
          status={p.failed ? 'failed' : 'sending'}
          onRetry={() => deliver(p)}
        />
      );
    }
    const m = item.message;
    const mine = m.senderId === me.id;
    return (
      <MessageBubble
        mine={mine}
        body={m.body}
        createdAt={m.createdAt}
        status={mine && m.id === lastMineId ? (lastMineRead ? 'read' : 'sent') : undefined}
      />
    );
  };

  let body;
  if (query.isPending) {
    body = (
      <View
        style={styles.skeletons}
        accessibilityRole="progressbar"
        accessibilityLabel={fr.common.loading}
      >
        <Skeleton width="60%" height={40} radius={18} />
        <Skeleton width="45%" height={40} radius={18} style={{ alignSelf: 'flex-end' }} />
        <Skeleton width="70%" height={40} radius={18} />
      </View>
    );
  } else if (query.isError && messages.length === 0) {
    body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  } else {
    body = (
      <FlatList
        inverted
        data={rows}
        keyExtractor={(row) => row.key}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        keyboardShouldPersistTaps="handled"
        onEndReachedThreshold={0.4}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        ListFooterComponent={
          query.isFetchingNextPage ? (
            <ActivityIndicator color={colors.primary} style={{ padding: 12 }} />
          ) : null
        }
        ListEmptyComponent={
          // Inverted lists flip the empty component too; flip it back.
          <View style={{ transform: [{ scaleY: -1 }] }}>
            <EmptyState
              glyph={icons.chat}
              title={fr.chat.startTitle(peer.firstName)}
              message={fr.chat.startMessage}
            />
          </View>
        }
      />
    );
  }

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={insets.top + 44}
    >
      <View style={{ flex: 1 }}>{body}</View>
      <View style={[styles.composer, { paddingBottom: insets.bottom }]}>
        <MessageComposer onSend={send} />
      </View>
      <UserActionsSheet
        visible={menuOpen}
        onClose={() => setMenuOpen(false)}
        target={{
          userId: peer.userId,
          firstName: peer.firstName,
          matchId: conversation.matchId,
          conversationId: conversation.id,
        }}
        leading={[
          {
            label: fr.chat.viewProfile,
            onPress: () =>
              navigation.navigate('ProfileDetail', { userId: peer.userId, source: 'chat' }),
          },
        ]}
        onGone={() => navigation.goBack()}
      />
    </KeyboardAvoidingView>
  );
}
