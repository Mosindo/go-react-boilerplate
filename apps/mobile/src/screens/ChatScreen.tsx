import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { chatApi, isApiError, type Message } from "../api";
import { useFeedback } from "../components/Feedback";
import { Avatar } from "../components/Photo";
import { ActionSheet } from "../components/Sheets";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { Text } from "../components/Text";
import { useAuth } from "../auth/AuthContext";
import { useMessages } from "../hooks/useQueries";
import { useSafetyActions } from "../hooks/useSafetyActions";
import { insertMessage, type MessagePage } from "../lib/chatCache";
import { timeLabel } from "../lib/format";
import { queryKeys } from "../lib/queryKeys";
import type { RootStackParamList } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList, "Chat">;

export function ChatScreen() {
  const theme = useTheme();
  const navigation = useNavigation<Nav>();
  const { params } = useRoute<RouteProp<RootStackParamList, "Chat">>();
  const { matchId, profile } = params;
  const { user } = useAuth();
  const me = user?.id ?? "";
  const queryClient = useQueryClient();
  const { toast, confirm } = useFeedback();
  const messages = useMessages(matchId);
  const [draft, setDraft] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const lastReadId = useRef(0);

  const leave = useCallback(() => navigation.popToTop(), [navigation]);
  const safety = useSafetyActions(profile, matchId, leave);

  const list: Message[] = messages.data?.pages.flatMap((p) => p.messages) ?? [];
  const newestIncoming = list.find((m) => m.senderId !== me);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          onPress={() => navigation.navigate("ProfileDetail", { profile, matchId })}
          style={styles.headerTitle}
          accessibilityRole="button"
          accessibilityLabel={`Profil de ${profile.firstName}`}
        >
          <Avatar url={profile.photos[0]?.url} name={profile.firstName} size={34} />
          <Text variant="bodyStrong">{profile.firstName}</Text>
        </Pressable>
      ),
      headerRight: () => (
        <Pressable
          onPress={() => setMenuOpen(true)}
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          hitSlop={10}
          testID="chat-menu"
        >
          <Text variant="title">⋯</Text>
        </Pressable>
      )
    });
  }, [navigation, profile, matchId]);

  // Mark incoming messages as read while the conversation is on screen.
  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(matchId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    }
  });
  useEffect(() => {
    if (newestIncoming && newestIncoming.id > lastReadId.current && !newestIncoming.readAt) {
      lastReadId.current = newestIncoming.id;
      markRead.mutate();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [newestIncoming?.id, newestIncoming?.readAt]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(matchId, body),
    onSuccess: (msg) => {
      queryClient.setQueryData<InfiniteData<MessagePage, number | undefined>>(
        queryKeys.messages(matchId),
        (d) => insertMessage(d, msg)
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (err, body) => {
      setDraft((cur) => cur || body);
      toast(err instanceof Error ? err.message : "Message non envoyé.", "error");
    }
  });

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    setDraft("");
    send.mutate(body);
  };

  const clearConversation = async () => {
    const ok = await confirm({
      title: "Supprimer la conversation ?",
      message: "L'historique sera effacé de votre côté uniquement. Votre match reste actif.",
      confirmLabel: "Supprimer",
      destructive: true
    });
    if (!ok) return;
    try {
      await chatApi.clear(matchId);
      queryClient.removeQueries({ queryKey: queryKeys.messages(matchId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      leave();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Suppression impossible.", "error");
    }
  };

  if (messages.isError && isApiError(messages.error, 404)) {
    return (
      <SafeAreaView style={[styles.flex, { backgroundColor: theme.bg }]}>
        <EmptyState
          title="Cette conversation n'existe plus"
          message="Le match a été retiré."
          actionLabel="Retour"
          onAction={leave}
        />
      </SafeAreaView>
    );
  }

  const lastOwnReadId = list.find((m) => m.senderId === me && m.readAt)?.id;

  return (
    <SafeAreaView edges={["bottom", "left", "right"]} style={[styles.flex, { backgroundColor: theme.bg }]}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={90}
      >
        {messages.isPending ? (
          <LoadingState />
        ) : messages.isError ? (
          <ErrorState message={messages.error.message} onRetry={() => void messages.refetch()} />
        ) : list.length === 0 ? (
          <EmptyState
            emoji="👋"
            title={`Dites bonjour à ${profile.firstName}`}
            message="Vous vous plaisez mutuellement : lancez la conversation !"
          />
        ) : (
          <FlatList
            inverted
            data={list}
            keyExtractor={(m) => String(m.id)}
            contentContainerStyle={styles.list}
            onEndReached={() =>
              messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()
            }
            onEndReachedThreshold={0.3}
            renderItem={({ item }) => {
              const mine = item.senderId === me;
              return (
                <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
                  <View
                    style={[
                      styles.bubble,
                      mine
                        ? { backgroundColor: theme.primary }
                        : { backgroundColor: theme.surface, borderColor: theme.border, borderWidth: 1 }
                    ]}
                  >
                    <Text color={mine ? theme.onPrimary : theme.text}>{item.body}</Text>
                  </View>
                  <Text variant="caption" muted>
                    {timeLabel(item.createdAt)}
                    {mine && item.id === lastOwnReadId ? " · Lu" : ""}
                  </Text>
                </View>
              );
            }}
          />
        )}

        <View style={[styles.composer, { borderTopColor: theme.border, backgroundColor: theme.bg }]}>
          <TextInput
            value={draft}
            onChangeText={setDraft}
            placeholder="Votre message…"
            placeholderTextColor={theme.textMuted}
            accessibilityLabel="Votre message"
            multiline
            maxLength={2000}
            style={[
              styles.input,
              { backgroundColor: theme.surface, borderColor: theme.border, color: theme.text }
            ]}
            onSubmitEditing={Platform.OS === "web" ? submit : undefined}
            testID="message-input"
          />
          <Pressable
            onPress={submit}
            disabled={!draft.trim() || send.isPending}
            accessibilityRole="button"
            accessibilityLabel="Envoyer"
            testID="send-message"
            style={[
              styles.send,
              { backgroundColor: theme.primary, opacity: !draft.trim() || send.isPending ? 0.5 : 1 }
            ]}
          >
            <Text variant="bodyStrong" color={theme.onPrimary}>
              Envoyer
            </Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>

      <ActionSheet
        visible={menuOpen}
        title={profile.firstName}
        onClose={() => setMenuOpen(false)}
        actions={[
          {
            label: "Voir le profil",
            onPress: () => navigation.navigate("ProfileDetail", { profile, matchId })
          },
          {
            label: "Supprimer la conversation",
            onPress: () => void clearConversation(),
            testID: "action-clear"
          },
          {
            label: "Retirer ce match",
            onPress: () => void safety.unmatch(),
            destructive: true,
            testID: "action-unmatch"
          },
          { label: "Signaler", onPress: safety.openReport, destructive: true, testID: "action-report" },
          { label: "Bloquer", onPress: () => void safety.block(), destructive: true, testID: "action-block" }
        ]}
      />
      {safety.reportSheet}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  headerTitle: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: { padding: spacing.lg, gap: spacing.sm },
  bubbleRow: { maxWidth: "82%", gap: 2, marginVertical: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radius.lg, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm + 2 },
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
    borderRadius: radius.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm + 2,
    fontSize: 16
  },
  send: {
    minHeight: 44,
    paddingHorizontal: spacing.lg,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center"
  }
});
