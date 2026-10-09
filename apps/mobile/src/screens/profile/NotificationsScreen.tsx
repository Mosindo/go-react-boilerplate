import React, { useCallback, useLayoutEffect } from 'react';
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Ionicons } from '@expo/vector-icons';
import { errorMessage } from '../../api/errors';
import { notificationsApi } from '../../api/notifications';
import type { AppNotification, Conversation } from '../../api/types';
import { fr } from '../../i18n/fr';
import type { MainStackParamList } from '../../navigation/types';
import { qk } from '../../query/keys';
import { useNotifications } from '../../query/hooks';
import {
  markAllNotificationsReadLocal,
  markNotificationReadLocal,
} from '../../realtime/cacheUpdaters';
import { EmptyState, ErrorState, LoadingView, Text } from '../../shared/ui';
import { IconButton } from '../../shared/ui';
import { spacing, useTheme } from '../../theme';
import { formatListTime } from '../../utils/format';

type Props = NativeStackScreenProps<MainStackParamList, 'Notifications'>;

export function notificationTitle(n: AppNotification): string {
  if (n.type === 'match') return fr.notifications.typeMatch;
  if (n.type === 'message') return fr.notifications.typeMessage;
  return n.title || fr.notifications.typeOther;
}

export function notificationBody(n: AppNotification): string {
  if (n.type === 'match') return fr.notifications.matchBody;
  return n.body;
}

const Item = React.memo(function Item({
  n,
  onPress,
}: {
  n: AppNotification;
  onPress: (n: AppNotification) => void;
}) {
  const { colors } = useTheme();
  const icon = n.type === 'match' ? 'heart' : n.type === 'message' ? 'chatbubble' : 'notifications';
  return (
    <Pressable
      testID={`notification-${n.id}`}
      accessibilityRole="button"
      accessibilityLabel={`${notificationTitle(n)}. ${notificationBody(n)}${n.isRead ? '' : '. Non lue'}`}
      onPress={() => onPress(n)}
      style={[
        styles.item,
        {
          borderColor: colors.border,
          backgroundColor: n.isRead ? 'transparent' : colors.primarySoft,
        },
      ]}
    >
      <View style={[styles.icon, { backgroundColor: colors.surfaceAlt }]}>
        <Ionicons name={icon} size={20} color={colors.primary} />
      </View>
      <View style={styles.text}>
        <Text variant={n.isRead ? 'body' : 'bodyStrong'}>{notificationTitle(n)}</Text>
        {notificationBody(n) ? (
          <Text variant="caption" tone="muted" numberOfLines={2}>
            {notificationBody(n)}
          </Text>
        ) : null}
      </View>
      <Text variant="caption" tone="muted">
        {formatListTime(n.createdAt)}
      </Text>
    </Pressable>
  );
});

export function NotificationsScreen({ navigation }: Props) {
  const qc = useQueryClient();
  const { data, isPending, isError, error, refetch, isRefetching } = useNotifications();

  const markRead = useMutation({
    mutationFn: (id: string) => notificationsApi.markRead(id),
    onMutate: (id) => markNotificationReadLocal(qc, id),
    onError: () => void qc.invalidateQueries({ queryKey: qk.notifications }),
  });
  const markAll = useMutation({
    mutationFn: () => notificationsApi.markAllRead(),
    onSuccess: () => markAllNotificationsReadLocal(qc),
  });

  const unread = data?.unreadCount ?? 0;

  useLayoutEffect(() => {
    navigation.setOptions({
      headerRight: () =>
        unread > 0 ? (
          <IconButton
            icon="checkmark-done"
            label={fr.notifications.markAll}
            onPress={() => markAll.mutate()}
            testID="notifications-mark-all"
          />
        ) : null,
    });
  }, [navigation, unread, markAll]);

  const onPress = useCallback(
    (n: AppNotification) => {
      if (!n.isRead) markRead.mutate(n.id);
      const conversationId = n.data?.conversationId;
      if (!conversationId) return;
      const conv = qc
        .getQueryData<Conversation[]>(qk.conversations)
        ?.find((c) => c.id === conversationId);
      if (conv) navigation.navigate('Chat', { conversationId, user: conv.user });
      else navigation.navigate('Tabs', { screen: 'Messages' });
    },
    [markRead, navigation, qc],
  );

  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  return (
    <FlatList
      testID="notifications-list"
      data={data.notifications}
      keyExtractor={(n) => n.id}
      renderItem={({ item }) => <Item n={item} onPress={onPress} />}
      ListHeaderComponent={
        unread > 0 ? (
          <Text variant="label" tone="muted" style={styles.header} testID="notifications-unread">
            {fr.notifications.unread(unread)}
          </Text>
        ) : null
      }
      ListEmptyComponent={
        <EmptyState
          icon="notifications-off-outline"
          title={fr.notifications.empty}
          testID="notifications-empty"
        />
      }
      contentContainerStyle={data.notifications.length === 0 ? styles.emptyContent : undefined}
      refreshControl={<RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} />}
    />
  );
}

const styles = StyleSheet.create({
  emptyContent: { flexGrow: 1 },
  header: { padding: spacing.lg, paddingBottom: spacing.sm },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: 'center', justifyContent: 'center' },
  text: { flex: 1, marginHorizontal: spacing.md },
});
