import React, { useEffect, useLayoutEffect, useState } from "react";
import { FlatList, Pressable, StyleSheet, TextInput, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { chatApi, matchingApi, safetyApi } from "../api/endpoints";
import { ReportSheet } from "../components/ReportSheet";
import { errorMessage, keys, useConversations } from "../hooks/queries";
import { timeLabel } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { useSession } from "../state/session";
import { discoverStore } from "../state/discoverStore";
import { ErrorView, LoadingView, Screen, Sheet, SheetAction, Text, radii, spacing, useTheme, useToast } from "../ui";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;

export default function ChatScreen({ navigation, route }: Props) {
  const { conversationId, userId, name } = route.params;
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { user } = useSession();
  const conversations = useConversations();
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);

  const messagesQuery = useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    getNextPageParam: (last) => last.nextBefore
  });

  // Newest first, as the list is inverted.
  const messages = (messagesQuery.data?.pages ?? []).flatMap((page) => [...page.messages].reverse());
  const newest = messages[0];
  const matchId = conversations.data?.find((c) => c.id === conversationId)?.matchId;

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => (
        <Pressable
          accessibilityLabel="Plus d'options"
          accessibilityRole="button"
          onPress={() => setMenu(true)}
          style={styles.menuButton}
          testID="chat-menu"
        >
          <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      )
    });
  }, [navigation, name, colors.text]);

  const newestId = newest?.id;
  const newestSender = newest?.senderId;
  useEffect(() => {
    if (newestId && newestSender !== user?.id) {
      chatApi
        .markRead(conversationId)
        .then(() => queryClient.invalidateQueries({ queryKey: keys.conversations }))
        .catch(() => undefined);
    }
  }, [newestId, newestSender, conversationId, user?.id, queryClient]);

  const send = async () => {
    const body = draft.trim();
    if (!body || sending) return;
    setSending(true);
    try {
      await chatApi.send(conversationId, body);
      setDraft("");
      await queryClient.invalidateQueries({ queryKey: keys.messages(conversationId) });
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
    } catch (error) {
      toast.show(errorMessage(error, "Message non envoyé."), "error");
    } finally {
      setSending(false);
    }
  };

  const leave = async (action: () => Promise<void>, message: string) => {
    setMenu(false);
    try {
      await action();
      discoverStore.refresh();
      await queryClient.invalidateQueries({ queryKey: keys.conversations });
      toast.show(message);
      navigation.popToTop();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  if (messagesQuery.isLoading) return <LoadingView />;
  if (messagesQuery.isError)
    return <ErrorView message="Cette conversation n'est plus disponible." onRetry={() => navigation.goBack()} />;

  const lastOwnIndex = messages.findIndex((m) => m.senderId === user?.id);

  return (
    <Screen padded={false} testID="chat-screen">
      <FlatList
        contentContainerStyle={styles.list}
        data={messages}
        inverted
        keyExtractor={(m) => m.id}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Text tone="muted">C'est un match ! Dites bonjour à {name} 👋</Text>
          </View>
        }
        onEndReached={() =>
          messagesQuery.hasNextPage && !messagesQuery.isFetchingNextPage && void messagesQuery.fetchNextPage()
        }
        onEndReachedThreshold={0.4}
        renderItem={({ item, index }) => {
          const mine = item.senderId === user?.id;
          return (
            <View style={[styles.bubbleWrap, mine ? styles.mine : styles.theirs]}>
              <View
                style={[
                  styles.bubble,
                  {
                    backgroundColor: mine ? colors.primary : colors.surface,
                    borderColor: colors.border,
                    borderWidth: mine ? 0 : 1
                  }
                ]}
                testID={mine ? "msg-mine" : "msg-theirs"}
              >
                <Text style={{ color: mine ? colors.primaryText : colors.text }}>{item.body}</Text>
              </View>
              <Text tone="muted" variant="caption">
                {timeLabel(item.createdAt)}
                {mine && index === lastOwnIndex && item.readByOther ? " · Lu" : ""}
              </Text>
            </View>
          );
        }}
      />

      <View style={[styles.composer, { borderColor: colors.border, backgroundColor: colors.background }]}>
        <TextInput
          accessibilityLabel="Votre message"
          maxLength={2000}
          multiline
          onChangeText={setDraft}
          placeholder="Écrire un message…"
          placeholderTextColor={colors.textMuted}
          style={[styles.input, { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border }]}
          testID="chat-input"
          value={draft}
        />
        <Pressable
          accessibilityLabel="Envoyer"
          accessibilityRole="button"
          disabled={!draft.trim() || sending}
          onPress={send}
          style={[styles.send, { backgroundColor: colors.primary, opacity: !draft.trim() || sending ? 0.5 : 1 }]}
          testID="chat-send"
        >
          <Ionicons color={colors.primaryText} name="arrow-up" size={22} />
        </Pressable>
      </View>

      <Sheet onClose={() => setMenu(false)} title={name} visible={menu}>
        <SheetAction
          label="Voir le profil"
          onPress={() => {
            setMenu(false);
            navigation.navigate("ProfileDetail", { userId });
          }}
        />
        <SheetAction
          label="Masquer la conversation"
          onPress={() => void leave(() => chatApi.hide(conversationId), "Conversation masquée.")}
        />
        {matchId ? (
          <SheetAction
            danger
            label="Se désapparier"
            onPress={() => void leave(() => matchingApi.unmatch(matchId), "Match supprimé.")}
          />
        ) : null}
        <SheetAction
          label="Signaler"
          onPress={() => {
            setMenu(false);
            setReporting(true);
          }}
          testID="chat-report"
        />
        <SheetAction
          danger
          label="Bloquer"
          onPress={() => void leave(() => safetyApi.block(userId), "Utilisateur bloqué.")}
          testID="chat-block"
        />
      </Sheet>
      <ReportSheet
        name={name}
        onClose={() => setReporting(false)}
        onDone={() => {
          discoverStore.refresh();
          navigation.popToTop();
        }}
        userId={userId}
        visible={reporting}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.sm, flexGrow: 1 },
  empty: { alignItems: "center", padding: spacing.xl, transform: [{ scaleY: -1 }] },
  bubbleWrap: { maxWidth: "80%", gap: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm + 2, borderRadius: radii.lg },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    borderWidth: 1,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    fontSize: 16
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  menuButton: { width: 44, height: 44, alignItems: "center", justifyContent: "center" }
});
