import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useFocusEffect } from "@react-navigation/native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { chatApi } from "../../api/chat";
import { errorMessage } from "../../api/client";
import type { Message } from "../../api/types";
import { SafetyActions } from "../../components/SafetyActions";
import { queryKeys } from "../../hooks/queryClient";
import { useConversation, useMessages, useSendMessage } from "../../hooks/queries";
import { setActiveConversation } from "../../hooks/useRealtime";
import { useSession } from "../../hooks/useSession";
import type { AppScreenProps } from "../../navigation/types";
import { colors, layout, radii, spacing, typography } from "../../theme";
import { Avatar, Dialog, EmptyState, ErrorState, IconButton, LoadingState, Text, showToast } from "../../ui";
import { formatDayLabel, formatMessageTime, isSameDay } from "../../utils/format";

const MAX_LENGTH = 2000;

export function ChatScreen({ navigation, route }: AppScreenProps<"Chat">) {
  const { conversationId } = route.params;
  const { user } = useSession();
  const qc = useQueryClient();
  const conversation = useConversation(conversationId);
  const messages = useMessages(conversationId);
  const send = useSendMessage(conversationId);
  const [draft, setDraft] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);

  const list = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);
  const peerLastReadAt = messages.data?.pages[0]?.peerLastReadAt ?? null;
  const peer = conversation.data?.peer;
  const latestPeerMessageId = list.find((m) => m.senderId !== user?.id)?.id;

  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.chatUnread });
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
    }
  });
  const markReadRef = useRef(markRead.mutate);
  markReadRef.current = markRead.mutate;

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      return () => setActiveConversation(null);
    }, [conversationId])
  );

  // Mark as read whenever a new message from the peer is displayed.
  useEffect(() => {
    if (latestPeerMessageId) {
      markReadRef.current();
    }
  }, [latestPeerMessageId]);

  const clear = useMutation({
    mutationFn: () => chatApi.clear(conversationId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      showToast("Conversation effacée de votre côté.", "success");
      navigation.goBack();
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) {
      return;
    }
    send.mutate(body, {
      onSuccess: () => setDraft(""),
      onError: (error) => showToast(errorMessage(error, "Message non envoyé."), "error")
    });
  };

  const lastMineId = list.find((m) => m.senderId === user?.id)?.id;

  const renderItem = ({ item, index }: { item: Message; index: number }) => {
    const mine = item.senderId === user?.id;
    const older = list[index + 1];
    const showDay = !older || !isSameDay(older.createdAt, item.createdAt);
    const read = mine && peerLastReadAt !== null && new Date(peerLastReadAt) >= new Date(item.createdAt);
    return (
      <View>
        {showDay ? (
          <Text variant="caption" color="textSubtle" align="center" style={styles.day}>
            {formatDayLabel(item.createdAt)}
          </Text>
        ) : null}
        <View style={[styles.bubbleRow, mine ? styles.mineRow : styles.theirsRow]}>
          <View style={[styles.bubble, mine ? styles.mine : styles.theirs]} testID={mine ? "message-mine" : "message-theirs"}>
            <Text variant="body" color={mine ? "onPrimary" : "text"} selectable>
              {item.body}
            </Text>
          </View>
          <Text variant="caption" color="textSubtle" style={styles.meta}>
            {formatMessageTime(item.createdAt)}
            {item.id === lastMineId ? (read ? " · Lu" : " · Envoyé") : ""}
          </Text>
        </View>
      </View>
    );
  };

  let body: React.ReactNode;
  if (messages.isLoading || conversation.isLoading) {
    body = <LoadingState />;
  } else if (messages.error || conversation.error) {
    body = <ErrorState message={errorMessage(messages.error ?? conversation.error)} onRetry={() => navigation.goBack()} />;
  } else if (list.length === 0) {
    body = (
      <EmptyState
        icon="chatbubbles-outline"
        title={`Dites bonjour à ${peer?.firstName ?? ""}`}
        body="Une question sur un centre d'intérêt commun est souvent un bon début."
      />
    );
  } else {
    body = (
      <FlatList
        inverted
        data={list}
        keyExtractor={(m) => m.id}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()}
        onEndReachedThreshold={0.3}
        ListFooterComponent={messages.isFetchingNextPage ? <ActivityIndicator color={colors.primary} style={styles.loader} /> : null}
        keyboardShouldPersistTaps="handled"
        testID="message-list"
      />
    );
  }

  return (
    <SafeAreaView edges={["top", "bottom"]} style={styles.root} testID="chat-screen">
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <View style={styles.column}>
          <View style={styles.header}>
            <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} testID="chat-back" />
            {peer ? (
              <Pressable
                style={styles.peer}
                accessibilityRole="button"
                accessibilityLabel={`Voir le profil de ${peer.firstName}`}
                onPress={() => navigation.navigate("ProfileDetail", { userId: peer.userId, matchId: conversation.data?.matchId })}
              >
                <Avatar path={peer.photo?.url} name={peer.firstName} size={38} />
                <Text variant="heading" numberOfLines={1}>
                  {peer.firstName}
                </Text>
              </Pressable>
            ) : (
              <View style={styles.flex} />
            )}
            {peer ? (
              <IconButton
                icon="ellipsis-horizontal"
                label="Options de la conversation"
                onPress={() => setMenuOpen(true)}
                testID="chat-menu"
              />
            ) : null}
          </View>
          <View style={styles.flex}>{body}</View>
          <View style={styles.composer}>
            <TextInput
              value={draft}
              onChangeText={setDraft}
              placeholder="Votre message…"
              placeholderTextColor={colors.textSubtle}
              selectionColor={colors.primary}
              multiline
              maxLength={MAX_LENGTH}
              style={styles.input}
              accessibilityLabel="Votre message"
              testID="chat-input"
              onSubmitEditing={Platform.OS === "web" ? submit : undefined}
              blurOnSubmit={false}
            />
            <IconButton
              icon="arrow-up"
              label="Envoyer"
              onPress={submit}
              disabled={!draft.trim() || send.isPending || !peer}
              background={colors.primary}
              color={colors.onPrimary}
              testID="chat-send"
            />
          </View>
        </View>
      </KeyboardAvoidingView>

      {peer ? (
        <SafetyActions
          visible={menuOpen}
          onClose={() => setMenuOpen(false)}
          userId={peer.userId}
          firstName={peer.firstName}
          matchId={conversation.data?.matchId}
          onClearConversation={() => {
            setMenuOpen(false);
            setConfirmClear(true);
          }}
          onDone={() => navigation.navigate("Tabs", { screen: "Matches" })}
        />
      ) : null}
      <Dialog
        visible={confirmClear}
        title="Effacer la conversation ?"
        message="Les messages seront masqués uniquement pour vous. Le match est conservé."
        confirmLabel="Effacer"
        destructive
        loading={clear.isPending}
        onConfirm={() => clear.mutate()}
        onCancel={() => setConfirmClear(false)}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: colors.background
  },
  flex: {
    flex: 1
  },
  column: {
    flex: 1,
    width: "100%",
    maxWidth: layout.maxContentWidth,
    alignSelf: "center"
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderBottomWidth: 1,
    borderBottomColor: colors.border
  },
  peer: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  list: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    gap: spacing.xxs
  },
  loader: {
    marginVertical: spacing.md
  },
  day: {
    marginVertical: spacing.sm
  },
  bubbleRow: {
    maxWidth: "80%",
    marginVertical: 2
  },
  mineRow: {
    alignSelf: "flex-end",
    alignItems: "flex-end"
  },
  theirsRow: {
    alignSelf: "flex-start",
    alignItems: "flex-start"
  },
  bubble: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2,
    borderRadius: radii.lg
  },
  mine: {
    backgroundColor: colors.primary,
    borderBottomRightRadius: 6
  },
  theirs: {
    backgroundColor: colors.surfaceHigh,
    borderBottomLeftRadius: 6
  },
  meta: {
    marginTop: 2,
    marginHorizontal: spacing.xxs
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderTopWidth: 1,
    borderTopColor: colors.border
  },
  input: {
    ...typography.body,
    flex: 1,
    maxHeight: 140,
    minHeight: 44,
    color: colors.text,
    backgroundColor: colors.surface,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.md,
    paddingTop: 11,
    paddingBottom: 11
  }
});
