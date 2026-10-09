import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import {
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View,
  type NativeSyntheticEvent,
  type TextInputKeyPressEventData,
} from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useFocusEffect } from '@react-navigation/native';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Ionicons } from '@expo/vector-icons';
import { chatApi } from '../../api/chat';
import { discoverApi } from '../../api/discover';
import { errorMessage } from '../../api/errors';
import type { Conversation, Message } from '../../api/types';
import { useMe } from '../../auth/AuthProvider';
import { fr } from '../../i18n/fr';
import type { MainStackParamList } from '../../navigation/types';
import { qk } from '../../query/keys';
import {
  addOptimisticMessage,
  applyMessageToConversations,
  clearConversationUnread,
  failOptimisticMessage,
  removeConversation,
  removeOptimisticMessage,
  resolveOptimisticMessage,
  type MessagesData,
} from '../../realtime/cacheUpdaters';
import { UserActionsMenu } from '../../shared/UserActionsMenu';
import { toast } from '../../shared/toastStore';
import { Avatar, ConfirmDialog, ErrorState, IconButton, LoadingView, Text } from '../../shared/ui';
import { activeConversation } from '../../store/appState';
import { fontSize, MAX_CONTENT_WIDTH, radius, spacing, useTheme } from '../../theme';
import { formatClock, formatDayHeader, isSameDay, localId } from '../../utils/format';
import { validateMessageBody } from '../../validators';

type Props = NativeStackScreenProps<MainStackParamList, 'Chat'>;

const PAGE_SIZE = 30;

type RowData = { message: Message; showDay: boolean; showRead: boolean };

const Bubble = React.memo(function Bubble({
  row,
  mine,
  onRetry,
}: {
  row: RowData;
  mine: boolean;
  onRetry: (m: Message) => void;
}) {
  const { colors } = useTheme();
  const { message, showDay, showRead } = row;
  const failed = message.localState === 'failed';
  return (
    <View>
      {showDay ? (
        <Text variant="caption" tone="muted" center style={styles.day}>
          {formatDayHeader(message.createdAt)}
        </Text>
      ) : null}
      <View style={[styles.rowWrap, mine ? styles.mineWrap : styles.theirsWrap]}>
        <Pressable
          testID={`message-${message.id}`}
          disabled={!failed}
          accessibilityRole={failed ? 'button' : 'text'}
          accessibilityLabel={`${mine ? 'Vous' : ''} ${message.body}. ${formatClock(message.createdAt)}${message.readAt && mine ? `. ${fr.messages.read}` : ''}${failed ? `. ${fr.messages.failed}` : ''}`}
          onPress={() => onRetry(message)}
          style={[
            styles.bubble,
            mine
              ? { backgroundColor: colors.bubbleMine, borderBottomRightRadius: 4 }
              : { backgroundColor: colors.bubbleTheirs, borderBottomLeftRadius: 4 },
            message.localState === 'sending' ? { opacity: 0.6 } : null,
            failed ? { borderWidth: 1, borderColor: colors.danger } : null,
          ]}
        >
          <Text style={{ color: mine ? colors.onBubbleMine : colors.onBubbleTheirs }}>
            {message.body}
          </Text>
        </Pressable>
        <Text variant="caption" tone={failed ? 'danger' : 'muted'} style={styles.meta}>
          {failed
            ? fr.messages.failed
            : message.localState === 'sending'
              ? fr.messages.sending
              : formatClock(message.createdAt)}
          {mine && showRead ? ` · ${fr.messages.read}` : ''}
        </Text>
      </View>
    </View>
  );
});

export function ChatScreen({ route, navigation }: Props) {
  const { conversationId, user } = route.params;
  const qc = useQueryClient();
  const { colors } = useTheme();
  const { data: me } = useMe();
  const myId = me?.id ?? '';
  const [text, setText] = useState('');
  const [menu, setMenu] = useState(false);
  const [confirm, setConfirm] = useState<'deactivate' | 'unmatch' | null>(null);
  const lastMarked = useRef<string | null>(null);
  const [focused, setFocused] = useState(true);

  const query = useInfiniteQuery<
    { messages: Message[]; hasMore: boolean },
    Error,
    MessagesData,
    ReturnType<typeof qk.messages>,
    string | undefined
  >({
    queryKey: qk.messages(conversationId),
    initialPageParam: undefined,
    queryFn: ({ pageParam, signal }) =>
      chatApi.messages(conversationId, pageParam, PAGE_SIZE, signal),
    getNextPageParam: (last) => (last.hasMore ? last.messages[0]?.id : undefined),
  });

  // Newest first for the inverted list: pages[0] is the newest page.
  const messages = useMemo(
    () => query.data?.pages.flatMap((p) => [...p.messages].reverse()) ?? [],
    [query.data],
  );

  const rows = useMemo<RowData[]>(() => {
    const firstReadMine = messages.find((m) => m.senderId === myId && m.readAt)?.id;
    return messages.map((message, i) => {
      const older = messages[i + 1];
      return {
        message,
        showDay: !older || !isSameDay(older.createdAt, message.createdAt),
        showRead: message.id === firstReadMine,
      };
    });
  }, [messages, myId]);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          testID="chat-header"
          accessibilityRole="button"
          accessibilityLabel={`${fr.messages.viewProfile} ${user.firstName}`}
          onPress={() => navigation.navigate('ProfileDetail', { userId: user.id, mode: 'view' })}
          style={styles.headerTitle}
        >
          <Avatar name={user.firstName} path={user.mainPhotoUrl} size={34} />
          <Text variant="heading" style={styles.headerName} numberOfLines={1}>
            {user.firstName}
          </Text>
        </Pressable>
      ),
      headerRight: () => (
        <IconButton
          icon="ellipsis-vertical"
          label={fr.messages.menu}
          onPress={() => setMenu(true)}
          testID="chat-menu"
        />
      ),
    });
  }, [navigation, user]);

  // Track which conversation is on screen (toasts / unread counters) and mark as read.
  useFocusEffect(
    useCallback(() => {
      activeConversation.set(conversationId);
      setFocused(true);
      return () => {
        setFocused(false);
        if (activeConversation.get() === conversationId) activeConversation.set(null);
      };
    }, [conversationId]),
  );

  const newestIncoming = messages.find((m) => m.senderId !== myId);
  useEffect(() => {
    if (!focused || !newestIncoming || newestIncoming.readAt) return;
    if (lastMarked.current === newestIncoming.id) return;
    lastMarked.current = newestIncoming.id;
    chatApi.markRead(conversationId).then(
      () => {
        clearConversationUnread(qc, conversationId);
        // Locally flag incoming messages as read so we do not call again.
        qc.setQueryData<MessagesData>(qk.messages(conversationId), (old) =>
          old
            ? {
                ...old,
                pages: old.pages.map((p) => ({
                  ...p,
                  messages: p.messages.map((m) =>
                    m.senderId !== myId && !m.readAt
                      ? { ...m, readAt: new Date().toISOString() }
                      : m,
                  ),
                })),
              }
            : old,
        );
        void qc.invalidateQueries({ queryKey: qk.notifications });
      },
      () => {
        lastMarked.current = null;
      },
    );
  }, [focused, newestIncoming, conversationId, qc, myId]);

  const send = useMutation({
    mutationFn: ({ body }: { tempId: string; body: string }) => chatApi.send(conversationId, body),
    meta: { silent: true },
    onSuccess: (real, { tempId }) => {
      resolveOptimisticMessage(qc, tempId, real);
      applyMessageToConversations(
        qc,
        real,
        { myUserId: myId, activeConversationId: conversationId },
        { countUnread: false },
      );
    },
    onError: (err, { tempId }) => {
      failOptimisticMessage(qc, conversationId, tempId);
      toast.error(errorMessage(err));
    },
  });

  const submitBody = useCallback(
    (body: string) => {
      const tempId = localId('tmp');
      addOptimisticMessage(qc, {
        id: tempId,
        conversationId,
        senderId: myId,
        body,
        createdAt: new Date().toISOString(),
        readAt: null,
        localState: 'sending',
      });
      send.mutate({ tempId, body });
    },
    [qc, conversationId, myId, send],
  );

  const onSend = () => {
    const body = text.trim();
    if (validateMessageBody(body) || !myId) return;
    setText('');
    submitBody(body);
  };

  const retry = useCallback(
    (m: Message) => {
      removeOptimisticMessage(qc, conversationId, m.id);
      submitBody(m.body);
    },
    [qc, conversationId, submitBody],
  );

  const onKeyPress = (e: NativeSyntheticEvent<TextInputKeyPressEventData>) => {
    const ev = e.nativeEvent as TextInputKeyPressEventData & { shiftKey?: boolean };
    if (Platform.OS === 'web' && ev.key === 'Enter' && !ev.shiftKey) {
      (e as unknown as { preventDefault?: () => void }).preventDefault?.();
      onSend();
    }
  };

  const leaveAfter = useMutation({
    mutationFn: async (kind: 'deactivate' | 'unmatch') => {
      if (kind === 'deactivate') return chatApi.remove(conversationId);
      const cached = qc
        .getQueryData<Conversation[]>(qk.conversations)
        ?.find((c) => c.id === conversationId);
      const matchId =
        cached?.matchId ??
        (await discoverApi.matches(50, 0)).find((m) => m.conversationId === conversationId)?.id;
      if (!matchId) throw new Error('match not found');
      return discoverApi.unmatch(matchId);
    },
    onSuccess: (_r, kind) => {
      removeConversation(qc, conversationId);
      void qc.invalidateQueries({ queryKey: qk.matches });
      toast.success(kind === 'unmatch' ? fr.messages.unmatched : fr.messages.deleted);
      setConfirm(null);
      navigation.goBack();
    },
    onError: () => setConfirm(null),
  });

  const renderItem = useCallback(
    ({ item }: { item: RowData }) => (
      <Bubble row={item} mine={item.message.senderId === myId} onRetry={retry} />
    ),
    [myId, retry],
  );

  let content: React.ReactNode;
  if (query.isPending) content = <LoadingView />;
  else if (query.isError)
    content = (
      <ErrorState
        title={fr.messages.loadFailed}
        message={errorMessage(query.error)}
        onRetry={() => void query.refetch()}
      />
    );
  else {
    content = (
      <>
        {messages.length === 0 ? (
          <View style={styles.emptyOverlay} pointerEvents="none">
            <Text tone="muted" center testID="chat-empty">
              {fr.messages.chatEmpty(user.firstName)}
            </Text>
          </View>
        ) : null}
        <FlatList
          testID="message-list"
          inverted
          data={rows}
          keyExtractor={(r) => r.message.id}
          renderItem={renderItem}
          contentContainerStyle={styles.listContent}
          onEndReached={() =>
            query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()
          }
          onEndReachedThreshold={0.4}
          ListFooterComponent={
            query.isFetchingNextPage ? <LoadingView label={fr.messages.olderLoading} /> : null
          }
          keyboardShouldPersistTaps="handled"
          initialNumToRender={20}
          windowSize={9}
        />
      </>
    );
  }

  return (
    <KeyboardAvoidingView
      style={[styles.flex, { backgroundColor: colors.background }]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={Platform.OS === 'ios' ? 90 : 0}
    >
      <View style={styles.flex}>{content}</View>
      <View
        style={[styles.composer, { backgroundColor: colors.surface, borderColor: colors.border }]}
      >
        <TextInput
          testID="chat-input"
          accessibilityLabel={fr.messages.placeholder}
          placeholder={fr.messages.placeholder}
          placeholderTextColor={colors.textMuted}
          value={text}
          onChangeText={setText}
          onKeyPress={onKeyPress}
          multiline
          maxLength={2000}
          style={[
            styles.input,
            { color: colors.text, backgroundColor: colors.background, borderColor: colors.border },
          ]}
        />
        <Pressable
          testID="chat-send"
          accessibilityRole="button"
          accessibilityLabel={fr.messages.send}
          accessibilityState={{ disabled: !text.trim() }}
          disabled={!text.trim()}
          onPress={onSend}
          style={[styles.send, { backgroundColor: colors.primary, opacity: text.trim() ? 1 : 0.4 }]}
        >
          <Ionicons name="send" size={20} color={colors.onPrimary} />
        </Pressable>
      </View>

      <UserActionsMenu
        visible={menu}
        onClose={() => setMenu(false)}
        user={{ id: user.id, firstName: user.firstName }}
        onBlocked={() => navigation.popToTop()}
        extraItems={[
          {
            key: 'profile',
            label: fr.messages.viewProfile,
            icon: 'person-outline',
            onPress: () => navigation.navigate('ProfileDetail', { userId: user.id, mode: 'view' }),
          },
          {
            key: 'deactivate',
            label: fr.messages.deleteConversation,
            icon: 'trash-outline',
            onPress: () => setConfirm('deactivate'),
          },
          {
            key: 'unmatch',
            label: fr.messages.unmatch,
            icon: 'heart-dislike-outline',
            destructive: true,
            onPress: () => setConfirm('unmatch'),
          },
        ]}
      />
      <ConfirmDialog
        visible={confirm !== null}
        title={
          confirm === 'unmatch' ? fr.messages.unmatchTitle(user.firstName) : fr.messages.deleteTitle
        }
        body={confirm === 'unmatch' ? fr.messages.unmatchBody : fr.messages.deleteBody}
        confirmLabel={confirm === 'unmatch' ? fr.messages.unmatch : fr.common.delete}
        destructive
        loading={leaveAfter.isPending}
        onCancel={() => setConfirm(null)}
        onConfirm={() => confirm && leaveAfter.mutate(confirm)}
      />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  listContent: { padding: spacing.md, paddingBottom: spacing.sm },
  headerTitle: { flexDirection: 'row', alignItems: 'center', maxWidth: MAX_CONTENT_WIDTH - 140 },
  headerName: { marginLeft: spacing.sm, flexShrink: 1 },
  day: { marginVertical: spacing.md },
  rowWrap: { marginBottom: spacing.sm, maxWidth: '82%' },
  mineWrap: { alignSelf: 'flex-end', alignItems: 'flex-end' },
  theirsWrap: { alignSelf: 'flex-start', alignItems: 'flex-start' },
  bubble: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm + 2,
    borderRadius: radius.lg,
  },
  meta: { marginTop: 2, marginHorizontal: spacing.xs },
  emptyOverlay: {
    ...StyleSheet.absoluteFillObject,
    alignItems: 'center',
    justifyContent: 'center',
    padding: spacing.xl,
  },
  composer: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    borderRadius: 22,
    borderWidth: 1,
    paddingHorizontal: spacing.lg,
    paddingTop: 11,
    paddingBottom: 11,
    fontSize: fontSize.md,
  },
  send: {
    width: 44,
    height: 44,
    borderRadius: 22,
    alignItems: 'center',
    justifyContent: 'center',
    marginLeft: spacing.sm,
  },
});
