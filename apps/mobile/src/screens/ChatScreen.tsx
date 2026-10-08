import React, { useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useIsFocused } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { describeError } from "../api/client";
import type { Message } from "../api/types";
import { useUserActions } from "../components/UserActions";
import { useMarkRead, useMe, useMessages, useSendMessage } from "../hooks/data";
import { useRealtime } from "../hooks/useRealtime";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { formatDay, sameDay } from "../shared/format";
import { Avatar, IconButton, MessageItem, Notice, Sheet, Text, radii, spacing, typography } from "../shared/ui";
import { useTheme } from "../theme";

const MAX_LENGTH = 2000;

type Row = { kind: "message"; message: Message } | { kind: "day"; key: string; label: string };

/** Newest-first messages (as served) with a day separator after the last message of each day. */
function withDaySeparators(messages: Message[]): Row[] {
  const rows: Row[] = [];
  messages.forEach((message, index) => {
    rows.push({ kind: "message", message });
    const older = messages[index + 1];
    if (!older || !sameDay(older.createdAt, message.createdAt)) {
      rows.push({ kind: "day", key: `day-${message.id}`, label: formatDay(message.createdAt) });
    }
  });
  return rows;
}

export default function ChatScreen({ navigation, route }: NativeStackScreenProps<MainStackParamList, "Chat">) {
  const { conversationId, user, matchId } = route.params;
  const { colors } = useTheme();
  const { data: me } = useMe();
  const { connected } = useRealtime();
  const focused = useIsFocused();
  const actions = useUserActions();
  const query = useMessages(conversationId);
  const send = useSendMessage(conversationId);
  const markRead = useMarkRead(conversationId);
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);

  const messages = useMemo(() => query.data?.pages.flatMap((page) => page.messages) ?? [], [query.data]);
  const otherReadAt = query.data?.pages[0]?.otherLastReadAt ?? "";
  const newestId = messages[0]?.id;
  const newestFromOther = messages[0] ? messages[0].senderId !== me?.user.id : false;
  const rows = useMemo(() => withDaySeparators(messages), [messages]);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          accessibilityLabel={`Voir le profil de ${user.firstName}`}
          accessibilityRole="button"
          onPress={() => navigation.navigate("ProfileDetail", { userId: user.id, matchId, conversationId })}
          style={styles.title}
        >
          <Avatar name={user.firstName} path={user.photoPath} size="sm" />
          <Text weight="semibold">{user.firstName}</Text>
        </Pressable>
      ),
      headerRight: () => <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenu(true)} />
    });
  }, [navigation, user, matchId, conversationId]);

  // Mark as read when opened and whenever a new message from the other person lands while visible.
  useEffect(() => {
    if (focused && query.isSuccess && (newestFromOther || messages.length === 0)) markRead.mutate();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focused, query.isSuccess, newestId, newestFromOther]);

  const submit = async () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    setDraft("");
    try {
      await send.mutateAsync(body);
    } catch (error) {
      setDraft(body);
      showToast(describeError(error), "error");
    }
  };

  const menuActions = [
    {
      label: "Voir le profil",
      icon: "person-outline" as const,
      onPress: () => navigation.navigate("ProfileDetail", { userId: user.id, matchId, conversationId })
    },
    {
      label: "Supprimer la conversation",
      icon: "trash-outline" as const,
      onPress: () => void actions.hideConversation(conversationId).then((done) => done && navigation.goBack())
    },
    ...(matchId
      ? [
          {
            label: "Annuler le match",
            icon: "heart-dislike-outline" as const,
            destructive: true,
            onPress: () => void actions.unmatchUser(matchId, user.firstName).then((done) => done && navigation.popToTop())
          }
        ]
      : []),
    { label: "Signaler", icon: "flag-outline" as const, onPress: () => navigation.navigate("Report", { userId: user.id, firstName: user.firstName }) },
    {
      label: "Bloquer",
      icon: "ban-outline" as const,
      destructive: true,
      onPress: () => void actions.blockUser(user.id, user.firstName).then((done) => done && navigation.popToTop())
    }
  ];

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      keyboardVerticalOffset={90}
      style={[styles.flex, { backgroundColor: colors.background }]}
    >
      <View style={[styles.flex, styles.page]}>
        {!connected ? <Notice message="Reconnexion en cours… vos messages seront synchronisés." /> : null}
        {query.isLoading ? (
          <LoadingView fullScreen />
        ) : query.isError ? (
          <ErrorView message={describeError(query.error)} onAction={() => void query.refetch()} />
        ) : messages.length === 0 ? (
          <EmptyView
            icon="sparkles-outline"
            message={`Vous vous plaisez mutuellement. Écrivez le premier message à ${user.firstName} !`}
            title="C'est un match !"
          />
        ) : (
          <FlatList
            contentContainerStyle={styles.list}
            data={rows}
            inverted
            keyExtractor={(row) => (row.kind === "message" ? row.message.id : row.key)}
            onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
            onEndReachedThreshold={0.4}
            renderItem={({ item }) =>
              item.kind === "day" ? (
                <Text align="center" style={styles.day} tone="subtle" variant="caption">
                  {item.label}
                </Text>
              ) : (
                <MessageItem
                  body={item.message.body}
                  createdAt={item.message.createdAt}
                  mine={item.message.senderId === me?.user.id}
                  read={Boolean(otherReadAt) && new Date(item.message.createdAt).getTime() <= new Date(otherReadAt).getTime()}
                />
              )
            }
          />
        )}

        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>
          <TextInput
            accessibilityLabel="Votre message"
            maxLength={MAX_LENGTH}
            multiline
            onChangeText={setDraft}
            onKeyPress={(event) => {
              // Enter sends on web; Shift+Enter inserts a line break.
              const native = event.nativeEvent as unknown as { key: string; shiftKey?: boolean };
              if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                event.preventDefault();
                void submit();
              }
            }}
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textSubtle}
            style={[styles.input, typography.body, { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border }]}
            value={draft}
          />
          <IconButton
            background={draft.trim() ? colors.primary : colors.surfaceMuted}
            color={draft.trim() ? colors.onPrimary : colors.textSubtle}
            disabled={!draft.trim() || send.isPending}
            icon="send"
            label="Envoyer"
            onPress={() => void submit()}
          />
        </View>
      </View>
      <Sheet actions={menuActions} onClose={() => setMenu(false)} title={user.firstName} visible={menu} />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  page: { width: "100%", maxWidth: 560, alignSelf: "center" },
  title: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: { paddingVertical: spacing.md },
  day: { marginVertical: spacing.sm },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm, padding: spacing.md, borderTopWidth: StyleSheet.hairlineWidth },
  input: { flex: 1, minHeight: 44, maxHeight: 140, borderWidth: 1, borderRadius: radii.xl, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm }
});
