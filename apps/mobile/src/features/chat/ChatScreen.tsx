import { useHeaderHeight } from "@react-navigation/elements";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useCallback, useEffect, useLayoutEffect, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, StyleSheet, View } from "react-native";
import { getErrorMessage } from "../../api/client";
import { useCurrentUserId } from "../../hooks/useAuth";
import { useMarkConversationRead, useMessages, useSendMessage, useUnmatch } from "../../hooks/useChat";
import { formatDayLabel } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { setActiveConversation } from "../../store/activeConversation";
import { pushDeckIntent } from "../../store/deckIntent";
import { spacing, useColors } from "../../theme";
import { ConfirmDialog, EmptyState, ErrorState, IconButton, ListSkeleton, Text, toast } from "../../ui";
import { useSafetyMenu } from "../safety/useSafetyMenu";
import { Composer } from "./Composer";
import { MessageBubble } from "./MessageBubble";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;

export function ChatScreen({ navigation, route }: Props) {
  const { match } = route.params;
  const { conversationId, user } = match;
  const colors = useColors();
  const myId = useCurrentUserId();
  const headerHeight = useHeaderHeight();
  const messages = useMessages(conversationId);
  const { send, retry, discard } = useSendMessage(conversationId);
  const markRead = useMarkConversationRead(conversationId);
  const unmatch = useUnmatch();
  const [confirmUnmatch, setConfirmUnmatch] = useState(false);

  const leave = useCallback(() => navigation.popToTop(), [navigation]);

  const safety = useSafetyMenu({
    target: { userId: user.userId, displayName: user.displayName },
    extraActions: [
      {
        label: "View profile",
        icon: "person-outline",
        onPress: () => navigation.navigate("ProfileDetail", { userId: user.userId, source: "match" })
      },
      { label: "Unmatch", icon: "heart-dislike-outline", destructive: true, onPress: () => setConfirmUnmatch(true) }
    ],
    onBlocked: leave
  });

  useLayoutEffect(() => {
    navigation.setOptions({
      title: user.displayName,
      headerRight: () => <IconButton icon="ellipsis-horizontal" label="Chat options" onPress={safety.open} />
    });
  });

  // This chat is on screen: incoming messages here are read immediately.
  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      void markRead().catch(() => undefined);
      return () => setActiveConversation(null);
    }, [conversationId, markRead])
  );

  const newest = messages.items[0];
  const newestIncomingId = newest && newest.senderId !== myId ? newest.id : null;
  useEffect(() => {
    if (newestIncomingId) void markRead().catch(() => undefined);
  }, [newestIncomingId, markRead]);

  async function doUnmatch() {
    try {
      await unmatch.mutateAsync(match);
      setConfirmUnmatch(false);
      toast.success(`You unmatched ${user.displayName}`);
      pushDeckIntent({ userId: user.userId, kind: "remove" });
      leave();
    } catch (e) {
      setConfirmUnmatch(false);
      toast.error(getErrorMessage(e));
    }
  }

  let body: React.ReactNode;
  if (messages.isPending) {
    body = <ListSkeleton rows={5} />;
  } else if (messages.isError && messages.items.length === 0) {
    body = <ErrorState error={messages.error} onRetry={() => void messages.refetch()} />;
  } else if (messages.items.length === 0) {
    body = (
      <EmptyState
        icon="chatbubble-ellipses-outline"
        message="A simple hello is a great start."
        title={`Say hi to ${user.displayName}`}
      />
    );
  } else {
    body = (
      <FlatList
        ListFooterComponent={
          messages.isFetchingNextPage ? (
            <ActivityIndicator color={colors.primary} style={{ margin: spacing.md }} />
          ) : null
        }
        contentContainerStyle={styles.list}
        data={messages.items}
        inverted
        keyExtractor={(item) => item.id}
        keyboardShouldPersistTaps="handled"
        onEndReached={() => {
          if (messages.hasNextPage && !messages.isFetchingNextPage) void messages.fetchNextPage();
        }}
        onEndReachedThreshold={0.3}
        renderItem={({ item, index }) => {
          const older = messages.items[index + 1];
          const showDay = !older || formatDayLabel(older.createdAt) !== formatDayLabel(item.createdAt);
          return (
            <View>
              {showDay ? (
                <Text center style={styles.day} tone="muted" variant="caption">
                  {formatDayLabel(item.createdAt)}
                </Text>
              ) : null}
              <MessageBubble message={item} mine={item.senderId === myId} onDiscard={discard} onRetry={retry} />
            </View>
          );
        }}
      />
    );
  }

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      keyboardVerticalOffset={headerHeight}
      style={[styles.screen, { backgroundColor: colors.background }]}
    >
      <View style={styles.screen}>{body}</View>
      <Composer onSend={send} placeholder={`Message ${user.displayName}`} />
      {safety.element}
      <ConfirmDialog
        confirmLabel="Unmatch"
        destructive
        loading={unmatch.isPending}
        message="You won't be able to message each other anymore. This can't be undone."
        onCancel={() => setConfirmUnmatch(false)}
        onConfirm={doUnmatch}
        title={`Unmatch ${user.displayName}?`}
        visible={confirmUnmatch}
      />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  list: { paddingVertical: spacing.sm },
  day: { marginVertical: spacing.xs }
});
