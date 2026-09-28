import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Alert, FlatList, KeyboardAvoidingView, Platform, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import { errorMessage } from "../api/client";
import type { ChatMessage, ReportReason } from "../api/types";
import { ActionSheet } from "../components/ActionSheet";
import { ProfileSheet } from "../components/ProfileSheet";
import { ReportSheet } from "../components/ReportSheet";
import { buildChatRows, type ChatRow } from "../domain/chat";
import { useMe } from "../hooks/useAuth";
import { flattenMessages, useMarkRead, useMessages, useSendMessage } from "../hooks/useChat";
import { flattenMatches, useMatches, useUnmatch } from "../hooks/useMatches";
import { setOpenConversation } from "../hooks/openConversation";
import { useCandidateProfile } from "../hooks/useProfile";
import { useBlock, useReport } from "../hooks/useSafety";
import type { MainScreenProps } from "../navigation/types";
import { EmptyView, ErrorView, showToast, Skeleton } from "../shared/feedback";
import { IconButton, Loader, MessageItem, Text } from "../shared/ui";
import { radius, spacing, typography, useTheme } from "../theme";

const MAX_LENGTH = 2000;

function ThreadSkeleton() {
  return (
    <View style={styles.skeleton}>
      <Skeleton height={40} width="60%" rounded={radius.lg} />
      <Skeleton height={40} style={styles.skeletonMine} width="50%" rounded={radius.lg} />
      <Skeleton height={40} width="70%" rounded={radius.lg} />
    </View>
  );
}

export default function ChatScreen({ navigation, route }: MainScreenProps<"Chat">) {
  const theme = useTheme();
  const { conversationId } = route.params;
  const meData = useMe().data;
  const myId = meData?.id ?? "";
  const myInterestIds = meData?.profile?.interests.map((interest) => interest.id) ?? [];
  const matchesQuery = useMatches();
  const match = useMemo(
    () => flattenMatches(matchesQuery.data).find((item) => item.conversationId === conversationId),
    [matchesQuery.data, conversationId]
  );
  const name = match?.user.firstName ?? route.params.name ?? "Conversation";
  const matchId = match?.matchId ?? route.params.matchId;
  const otherUserId = match?.user.userId ?? route.params.userId;

  const messagesQuery = useMessages(conversationId);
  const send = useSendMessage(conversationId, myId);
  const markRead = useMarkRead(conversationId);
  const unmatch = useUnmatch();
  const block = useBlock();
  const report = useReport();

  const [draft, setDraft] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [reporting, setReporting] = useState(false);
  const [viewing, setViewing] = useState(false);
  const profileQuery = useCandidateProfile(otherUserId, viewing);

  const messages = flattenMessages(messagesQuery.data);
  const rows = useMemo(() => buildChatRows(messages), [messages]);
  const loaded = messagesQuery.isSuccess;

  useFocusEffect(
    useCallback(() => {
      setOpenConversation(conversationId);
      return () => setOpenConversation(null);
    }, [conversationId])
  );

  useEffect(() => {
    if (viewing && profileQuery.isError) {
      setViewing(false);
      showToast("This profile is not available.", "info");
    }
  }, [viewing, profileQuery.isError]);

  useEffect(() => {
    if (loaded) {
      void markRead();
    }
  }, [loaded, markRead]);

  useEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => <IconButton glyph="⋯" label="Conversation options" onPress={() => setMenuOpen(true)} />
    });
  }, [name, navigation]);

  const submit = () => {
    const body = draft.trim();
    if (!body || !loaded) {
      return;
    }
    setDraft("");
    void send(body);
  };

  const confirmUnmatch = () => {
    if (!matchId) {
      return;
    }
    Alert.alert(`Unmatch ${name}?`, "The conversation and all messages are deleted for both of you.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Unmatch",
        style: "destructive",
        onPress: () =>
          unmatch.mutate(matchId, {
            onSuccess: () => {
              showToast(`You unmatched ${name}.`, "success");
              navigation.goBack();
            },
            onError: (error) => showToast(errorMessage(error), "error")
          })
      }
    ]);
  };

  const confirmBlock = () => {
    if (!otherUserId) {
      return;
    }
    Alert.alert(`Block ${name}?`, "You will no longer see each other anywhere in the app.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Block",
        style: "destructive",
        onPress: () =>
          block.mutate(otherUserId, {
            onSuccess: () => {
              showToast(`${name} was blocked.`, "success");
              navigation.goBack();
            },
            onError: (error) => showToast(errorMessage(error), "error")
          })
      }
    ]);
  };

  const submitReport = (reason: ReportReason, details: string) => {
    if (!otherUserId) {
      return;
    }
    report.mutate(
      { userId: otherUserId, reason, details: details || undefined },
      {
        onSuccess: () => {
          setReporting(false);
          showToast("Thanks. Your report was sent.", "success");
        },
        onError: (error) => showToast(errorMessage(error), "error")
      }
    );
  };

  const renderRow = ({ item }: { item: ChatRow }) => {
    if (item.kind === "day") {
      return (
        <Text center style={styles.day} tone="muted" variant="caption">
          {item.label}
        </Text>
      );
    }
    return (
      <MessageItem
        message={item.message}
        mine={item.message.senderId === myId}
        onRetry={(message: ChatMessage) => void send(message.body, message)}
        showTime={item.showTime}
        startsGroup={item.startsGroup}
      />
    );
  };

  const menu = [
    ...(matchId ? [{ label: "Unmatch", destructive: true, onPress: confirmUnmatch }] : []),
    ...(otherUserId
      ? [
          { label: "View profile", onPress: () => setViewing(true) },
          { label: "Block", destructive: true, onPress: confirmBlock },
          { label: "Report", onPress: () => setReporting(true) }
        ]
      : [])
  ];

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      keyboardVerticalOffset={Platform.OS === "ios" ? 88 : 0}
      style={[styles.root, { backgroundColor: theme.background }]}
    >
      {messagesQuery.isPending ? (
        <ThreadSkeleton />
      ) : messagesQuery.isError && messages.length === 0 ? (
        <ErrorView message={errorMessage(messagesQuery.error)} onRetry={() => void messagesQuery.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={rows.length === 0 ? styles.emptyList : styles.list}
          data={rows}
          inverted={rows.length > 0}
          keyExtractor={(row) => row.key}
          keyboardShouldPersistTaps="handled"
          ListEmptyComponent={
            <EmptyView message={`Break the ice. A simple hello goes a long way.`} title={`Say hi to ${name}`} />
          }
          ListFooterComponent={messagesQuery.isFetchingNextPage ? <Loader /> : null}
          onEndReached={() => {
            if (messagesQuery.hasNextPage && !messagesQuery.isFetchingNextPage) {
              void messagesQuery.fetchNextPage();
            }
          }}
          onEndReachedThreshold={0.4}
          renderItem={renderRow}
        />
      )}
      <View style={[styles.composer, { borderTopColor: theme.border, backgroundColor: theme.surface }]}>
        <TextInput
          accessibilityLabel="Message"
          maxLength={MAX_LENGTH}
          multiline
          onChangeText={setDraft}
          placeholder="Write a message"
          placeholderTextColor={theme.textMuted}
          style={[
            styles.input,
            typography.body,
            { color: theme.text, backgroundColor: theme.background, borderColor: theme.border }
          ]}
          value={draft}
        />
        <IconButton disabled={!draft.trim() || !loaded} filled glyph="➤" label="Send message" onPress={submit} />
      </View>
      <ActionSheet actions={menu} onClose={() => setMenuOpen(false)} title={name} visible={menuOpen} />
      {viewing && profileQuery.data ? (
        <ProfileSheet
          candidate={profileQuery.data}
          myInterestIds={myInterestIds}
          onBlocked={() => navigation.goBack()}
          onClose={() => setViewing(false)}
        />
      ) : null}
      <ReportSheet
        name={name}
        onClose={() => setReporting(false)}
        onSubmit={submitReport}
        submitting={report.isPending}
        visible={reporting}
      />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  list: { paddingVertical: spacing.md },
  emptyList: { flexGrow: 1 },
  day: { marginVertical: spacing.md },
  skeleton: { padding: spacing.lg, gap: spacing.md },
  skeletonMine: { alignSelf: "flex-end" },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderWidth: 1,
    borderRadius: radius.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm
  }
});
