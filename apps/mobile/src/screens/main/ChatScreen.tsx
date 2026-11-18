import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useFocusEffect } from "@react-navigation/native";
import { errorMessage, isApiError } from "../../api/errors";
import type { Message } from "../../api/types";
import { SafetySheet } from "../../components/SafetySheet";
import { useFeedback } from "../../feedback";
import { useMarkRead, useMessages, useSendMessage } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { useRealtime } from "../../hooks/useRealtime";
import type { AppScreenProps } from "../../navigation/types";
import { layout, makeStyles, radii, spacing, typography, useTheme } from "../../theme";
import { Avatar, EmptyState, ErrorState, IconButton, LoadingState, Text } from "../../ui";
import { formatDayHeader, formatTime, sameDay } from "../../utils/dates";

const MAX_LENGTH = 2000;

export default function ChatScreen({ navigation, route }: AppScreenProps<"Chat">) {
  const { userId, firstName, photoUrl } = route.params;
  const styles = useStyles();
  const { colors } = useTheme();
  const { user } = useAuth();
  const { toast } = useFeedback();
  const { connected, setActiveConversation } = useRealtime();
  const thread = useMessages(userId, connected ? false : 10_000);
  const send = useSendMessage(userId);
  const markRead = useMarkRead(userId);
  const [draft, setDraft] = useState("");
  const [safetyOpen, setSafetyOpen] = useState(false);
  const inputRef = useRef<TextInput>(null);

  // Newest first, matching the inverted list.
  const messages = useMemo(() => (thread.data?.pages ?? []).flatMap((p) => [...p.messages].reverse()), [thread.data]);
  const otherLastReadAt = thread.data?.pages[0]?.otherLastReadAt ?? null;
  const newestIncoming = messages.find((m) => m.senderUserId !== user?.id)?.id;
  const lastOwn = messages.find((m) => m.senderUserId === user?.id);
  const seen = !!lastOwn && !!otherLastReadAt && new Date(otherLastReadAt) >= new Date(lastOwn.createdAt);

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(userId);
      return () => setActiveConversation(null);
    }, [setActiveConversation, userId])
  );

  // Mark as read on open and whenever a new incoming message arrives.
  const { mutate: markAsRead } = markRead;
  useEffect(() => {
    if (thread.isSuccess) {
      markAsRead();
    }
  }, [markAsRead, newestIncoming, thread.isSuccess]);

  const gone = thread.isError && isApiError(thread.error, "not_found");
  useEffect(() => {
    if (gone) {
      toast(`La conversation avec ${firstName} n’est plus disponible.`, "info");
    }
  }, [firstName, gone, toast]);

  const submit = async () => {
    const content = draft.trim();
    if (!content || send.isPending) {
      return;
    }
    setDraft("");
    try {
      await send.mutateAsync(content);
    } catch (e) {
      setDraft(content);
      toast(errorMessage(e), "error");
    }
  };

  const renderItem = ({ item, index }: { item: Message; index: number }) => {
    const mine = item.senderUserId === user?.id;
    const older = messages[index + 1];
    const showDay = !older || !sameDay(older.createdAt, item.createdAt);
    return (
      <View>
        {showDay ? (
          <Text align="center" color="textSubtle" style={styles.day} variant="caption">
            {formatDayHeader(item.createdAt)}
          </Text>
        ) : null}
        <View style={[styles.bubbleRow, mine ? styles.mineRow : null]}>
          <View
            accessibilityLabel={`${mine ? "Vous" : firstName}, ${formatTime(item.createdAt)} : ${item.content}`}
            style={[
              styles.bubble,
              mine ? { backgroundColor: colors.accent } : { backgroundColor: colors.surface, borderColor: colors.border, borderWidth: 1 }
            ]}
          >
            <Text color={mine ? "textOnAccent" : "text"} selectable>
              {item.content}
            </Text>
            <Text color={mine ? "textOnAccent" : "textSubtle"} style={styles.time} variant="caption">
              {formatTime(item.createdAt)}
            </Text>
          </View>
        </View>
        {mine && item.id === lastOwn?.id ? (
          <Text color="textSubtle" style={styles.receipt} testID="chat-receipt" variant="caption">
            {seen ? "Vu" : "Envoyé"}
          </Text>
        ) : null}
      </View>
    );
  };

  let body: React.ReactNode;
  if (thread.isLoading) {
    body = <LoadingState />;
  } else if (gone) {
    body = <EmptyState icon="chatbubbles-outline" message="Le match a été annulé." title="Conversation fermée" />;
  } else if (thread.isError) {
    body = <ErrorState message={errorMessage(thread.error)} onRetry={() => void thread.refetch()} />;
  } else if (messages.length === 0) {
    body = (
      <EmptyState
        icon="chatbubble-ellipses-outline"
        message={`Lancez la conversation avec ${firstName} : une question sur son profil marche toujours mieux qu’un simple « salut ».`}
        title="Dites bonjour"
      />
    );
  } else {
    body = (
      <FlatList
        contentContainerStyle={styles.list}
        data={messages}
        inverted
        keyExtractor={(m) => m.id}
        keyboardShouldPersistTaps="handled"
        onEndReached={() => {
          if (thread.hasNextPage && !thread.isFetchingNextPage) {
            void thread.fetchNextPage();
          }
        }}
        onEndReachedThreshold={0.3}
        renderItem={renderItem}
        testID="chat-messages"
      />
    );
  }

  return (
    <SafeAreaView edges={["top", "bottom"]} style={styles.root} testID="chat-screen">
      <View style={styles.header}>
        <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} />
        <Pressable
          accessibilityLabel={`Voir le profil de ${firstName}`}
          accessibilityRole="button"
          onPress={() => navigation.navigate("ProfileDetail", { userId, source: "match" })}
          style={styles.who}
        >
          <Avatar name={firstName} size={40} uri={photoUrl} />
          <Text variant="heading">{firstName}</Text>
        </Pressable>
        <IconButton icon="ellipsis-horizontal" label="Options de sécurité" onPress={() => setSafetyOpen(true)} testID="chat-safety" />
      </View>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={8} style={styles.flex}>
        <View style={styles.flex}>{body}</View>
        {!gone ? (
          <View style={styles.composer}>
            <TextInput
              ref={inputRef}
              accessibilityLabel="Votre message"
              maxLength={MAX_LENGTH}
              multiline
              onChangeText={setDraft}
              onKeyPress={(e) => {
                const native = e.nativeEvent as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                  e.preventDefault();
                  void submit();
                }
              }}
              placeholder="Votre message…"
              placeholderTextColor={colors.textSubtle}
              style={[styles.input, { color: colors.text }]}
              testID="chat-input"
              value={draft}
            />
            <IconButton
              disabled={!draft.trim() || send.isPending}
              icon="arrow-up"
              label="Envoyer"
              onPress={() => void submit()}
              size={44}
              testID="chat-send"
              tone="accent"
            />
          </View>
        ) : null}
      </KeyboardAvoidingView>
      <SafetySheet
        firstName={firstName}
        isMatch
        onClose={() => setSafetyOpen(false)}
        onRemoved={() => navigation.goBack()}
        userId={userId}
        visible={safetyOpen}
      />
    </SafeAreaView>
  );
}

const useStyles = makeStyles((c) => ({
  root: { flex: 1, backgroundColor: c.background },
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: c.border
  },
  who: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: { padding: spacing.md, gap: 6, width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" },
  day: { marginVertical: spacing.sm },
  bubbleRow: { flexDirection: "row" },
  mineRow: { justifyContent: "flex-end" },
  bubble: { maxWidth: "80%", borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.xs + 2, gap: 2 },
  time: { alignSelf: "flex-end", opacity: 0.8, fontSize: 11 },
  receipt: { alignSelf: "flex-end", marginTop: 2, marginRight: 4 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: c.border,
    backgroundColor: c.background
  },
  input: {
    flex: 1,
    ...typography.body,
    maxHeight: 140,
    minHeight: 44,
    borderRadius: 22,
    paddingHorizontal: spacing.md,
    paddingTop: 11,
    paddingBottom: 11,
    backgroundColor: c.surface,
    borderWidth: 1,
    borderColor: c.border
  }
}));
