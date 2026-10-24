import React, { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, Pressable, StyleSheet, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { chatApi, discoverApi } from "../../api/endpoints";
import type { Message } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { formatShortTime } from "../../lib/date";
import { errorMessage } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { insertMessage, invalidateChat, queryKeys, type MessagePages } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import { Avatar, EmptyState, Icon, LoadingState, Text, useDialog, useToast } from "../../ui";
import { useSafetyActions } from "../safety/useSafetyActions";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;

const PAGE_SIZE = 40;
const MAX_LENGTH = 2000;

export function ChatScreen({ route, navigation }: Props) {
  const { conversationId, userId, name, photoId } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const dialog = useDialog();
  const toast = useToast();
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const goBack = useCallback(() => navigation.goBack(), [navigation]);
  const safety = useSafetyActions(userId, name, goBack);

  const query = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam, PAGE_SIZE),
    getNextPageParam: (last) => (last.length === PAGE_SIZE ? last[last.length - 1]?.id : undefined),
    retry: (count, e) => !(e instanceof ApiError && e.status === 404) && count < 2,
  });

  const messages = useMemo(() => query.data?.pages.flat() ?? [], [query.data]);
  const newestIncomingUnread = messages.find((m) => m.senderId !== user?.id && !m.readAt)?.id;

  // Read receipts: acknowledge incoming messages while this screen is open.
  useEffect(() => {
    if (!newestIncomingUnread) return;
    chatApi
      .markRead(conversationId)
      .then(() => invalidateChat(queryClient))
      .catch(() => undefined);
  }, [newestIncomingUnread, conversationId, queryClient]);

  const unmatch = useCallback(async () => {
    const ok = await dialog.confirm({
      title: `Se désapparier de ${name} ?`,
      message: "La conversation sera fermée pour vous deux. Vous ne vous reverrez plus.",
      confirmLabel: "Se désapparier",
      destructive: true,
    });
    if (!ok) return;
    try {
      const conversations = await chatApi.conversations(100);
      const matchId = conversations.find((c) => c.id === conversationId)?.matchId;
      if (matchId) await discoverApi.unmatch(matchId);
      invalidateChat(queryClient);
      navigation.goBack();
    } catch (e) {
      toast.show({ message: errorMessage(e), kind: "error" });
    }
  }, [dialog, name, conversationId, queryClient, navigation, toast]);

  const openMenu = useCallback(async () => {
    const choice = await dialog.choose({
      title: name,
      actions: [
        { label: "Voir le profil", value: "profile", variant: "secondary" },
        { label: "Se désapparier", value: "unmatch", variant: "secondary" },
        { label: "Signaler", value: "report", variant: "secondary" },
        { label: "Bloquer", value: "block", variant: "danger" },
        { label: "Annuler", value: "cancel", variant: "ghost" },
      ],
    });
    if (choice === "profile") navigation.navigate("ProfileDetail", { userId });
    if (choice === "unmatch") await unmatch();
    if (choice === "report") await safety.report();
    if (choice === "block") await safety.block();
  }, [dialog, name, navigation, userId, unmatch, safety]);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          testID="chat-header"
          accessibilityRole="button"
          accessibilityLabel={`Profil de ${name}`}
          onPress={() => navigation.navigate("ProfileDetail", { userId })}
          style={styles.headerTitle}
        >
          <Avatar name={name} size={32} photoId={photoId} />
          <Text variant="heading">{name}</Text>
        </Pressable>
      ),
      headerRight: () => (
        <Pressable
          testID="chat-menu"
          accessibilityRole="button"
          accessibilityLabel="Options de la conversation"
          onPress={openMenu}
          hitSlop={8}
        >
          <Icon name="ellipsis-horizontal" size={24} />
        </Pressable>
      ),
    });
  }, [navigation, name, userId, photoId, openMenu]);

  const send = async () => {
    const content = text.trim();
    if (!content || sending) return;
    setSending(true);
    setText("");
    try {
      const message = await chatApi.send(conversationId, content);
      queryClient.setQueryData<MessagePages>(queryKeys.messages(conversationId), (d) =>
        insertMessage(d, message),
      );
      invalidateChat(queryClient);
    } catch (e) {
      setText(content);
      toast.show({ message: errorMessage(e), kind: "error" });
    } finally {
      setSending(false);
    }
  };

  if (query.isLoading) return <LoadingState />;
  if (query.error instanceof ApiError && query.error.status === 404) {
    return (
      <EmptyState
        testID="chat-unavailable"
        icon="heart-dislike-outline"
        title="Conversation terminée"
        message="Ce match n'existe plus."
        actionLabel="Retour"
        onAction={() => navigation.goBack()}
      />
    );
  }

  const lastOwnId = messages.find((m) => m.senderId === user?.id)?.id;

  return (
    <View style={[styles.flex, { backgroundColor: colors.bg }]} testID="chat-screen">
      <FlatList
        testID="message-list"
        inverted
        data={messages}
        keyExtractor={(m) => m.id}
        onEndReached={() =>
          query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()
        }
        onEndReachedThreshold={0.4}
        contentContainerStyle={styles.list}
        ListEmptyComponent={
          <View style={styles.hello}>
            <Text muted center>
              C&apos;est un match avec {name}. Brisez la glace !
            </Text>
          </View>
        }
        renderItem={({ item }) => (
          <Bubble
            message={item}
            mine={item.senderId === user?.id}
            showRead={item.id === lastOwnId}
          />
        )}
      />
      <View
        style={[
          styles.composer,
          { borderTopColor: colors.border, backgroundColor: colors.surface },
        ]}
      >
        <TextInput
          testID="chat-input"
          accessibilityLabel="Votre message"
          value={text}
          onChangeText={setText}
          placeholder="Écrire un message…"
          placeholderTextColor={colors.textMuted}
          multiline
          maxLength={MAX_LENGTH}
          style={[styles.input, { color: colors.text, backgroundColor: colors.surfaceAlt }]}
        />
        <Pressable
          testID="chat-send"
          accessibilityRole="button"
          accessibilityLabel="Envoyer"
          disabled={!text.trim() || sending}
          onPress={send}
          style={[
            styles.send,
            { backgroundColor: colors.primary, opacity: !text.trim() || sending ? 0.4 : 1 },
          ]}
        >
          <Icon name="arrow-up" size={22} color={colors.onPrimary} />
        </Pressable>
      </View>
    </View>
  );
}

function Bubble({
  message,
  mine,
  showRead,
}: {
  message: Message;
  mine: boolean;
  showRead: boolean;
}) {
  const { colors } = useTheme();
  return (
    <View style={[styles.bubbleWrap, mine ? styles.mine : styles.theirs]}>
      <View
        style={[
          styles.bubble,
          {
            backgroundColor: mine ? colors.primary : colors.surface,
            borderColor: colors.border,
            borderWidth: mine ? 0 : 1,
          },
        ]}
      >
        <Text color={mine ? colors.onPrimary : colors.text} testID="message-text">
          {message.content}
        </Text>
      </View>
      <Text variant="caption" muted testID={mine && showRead ? "message-meta-last-own" : undefined}>
        {formatShortTime(message.createdAt)}
        {mine && showRead ? (message.readAt ? " · Lu" : " · Envoyé") : ""}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { padding: spacing.lg, gap: spacing.sm },
  hello: { padding: spacing.xl, transform: [{ scaleY: -1 }] },
  headerTitle: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  bubbleWrap: { maxWidth: "80%", gap: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radius.lg, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderRadius: radius.lg,
    paddingHorizontal: spacing.lg,
    paddingTop: 11,
    paddingBottom: 11,
    fontSize: 16,
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
});
