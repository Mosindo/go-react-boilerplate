import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, Text, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { errorMessage } from "../api/client";
import type { Message } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { ReportModal } from "../features/safety/ReportModal";
import { formatTime } from "../lib/format";
import type { MainStackParams } from "../navigation/types";
import { useRealtimeEvents } from "../realtime/RealtimeContext";
import { radius, spacing, useTheme } from "../theme";
import { confirmAction, notify } from "../lib/dialogs";
import { ActionSheet } from "../ui/ActionSheet";
import { ErrorText, Loading } from "../ui/components";

type Props = NativeStackScreenProps<MainStackParams, "Chat">;

export function ChatScreen({ route, navigation }: Props) {
  const { conversationId, matchId, user } = route.params;
  const theme = useTheme();
  const { user: me } = useAuth();
  const queryClient = useQueryClient();
  const [messages, setMessages] = useState<Message[]>([]); // oldest first
  const [otherRead, setOtherRead] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [reporting, setReporting] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const loadingOlder = useRef(false);

  const markRead = useCallback(() => {
    void api
      .markRead(conversationId)
      .then(() => queryClient.invalidateQueries({ queryKey: ["conversations"] }))
      .catch(() => undefined);
  }, [conversationId, queryClient]);

  useEffect(() => {
    let active = true;
    api
      .messages(conversationId)
      .then((page) => {
        if (!active) {
          return;
        }
        setMessages(page.messages);
        setOtherRead(page.otherLastReadMessageId);
        setHasMore(page.messages.length >= 40);
        markRead();
      })
      .catch((e) => active && setError(errorMessage(e)))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [conversationId, markRead]);

  useRealtimeEvents((event) => {
    if (event.type === "message" && event.data.conversationId === conversationId) {
      setMessages((current) => (current.some((m) => m.id === event.data.id) ? current : [...current, event.data]));
      if (event.data.senderId !== me?.id) {
        markRead();
      }
    } else if (event.type === "read" && event.data.conversationId === conversationId) {
      setOtherRead(event.data.upToMessageId);
    } else if (event.type === "unmatch" && event.data.matchId === matchId) {
      navigation.popToTop();
    }
  });

  const leave = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["conversations"] });
    navigation.popToTop();
  }, [navigation, queryClient]);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: user.firstName,
      headerRight: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Options de la conversation"
          testID="chat-menu"
          onPress={() => setMenuOpen(true)}
        >
          <Text style={{ color: theme.text, fontSize: 24, paddingHorizontal: spacing.sm }}>⋯</Text>
        </Pressable>
      )
    });
  }, [navigation, user, theme.text]);

  const loadOlder = async () => {
    if (loadingOlder.current || !hasMore || messages.length === 0) {
      return;
    }
    loadingOlder.current = true;
    try {
      const page = await api.messages(conversationId, messages[0].id);
      setMessages((current) => [...page.messages, ...current]);
      setHasMore(page.messages.length >= 40);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      loadingOlder.current = false;
    }
  };

  const send = async () => {
    const body = text.trim();
    if (!body || sending) {
      return;
    }
    setSending(true);
    setError(null);
    try {
      const sent = await api.sendMessage(conversationId, body);
      setText("");
      setMessages((current) => (current.some((m) => m.id === sent.id) ? current : [...current, sent]));
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSending(false);
    }
  };

  const lastOwnId = [...messages].reverse().find((m) => m.senderId === me?.id)?.id;
  const inverted = [...messages].reverse();

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      style={[styles.flex, { backgroundColor: theme.background }]}
      keyboardVerticalOffset={90}
    >
      {loading ? (
        <Loading />
      ) : (
        <FlatList
          inverted
          data={inverted}
          keyExtractor={(m) => String(m.id)}
          onEndReached={() => void loadOlder()}
          onEndReachedThreshold={0.3}
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm }}
          ListEmptyComponent={
            <Text
              style={{ color: theme.textMuted, textAlign: "center", transform: [{ scaleY: -1 }] }}
            >{`Vous avez matché avec ${user.firstName}. Dites bonjour !`}</Text>
          }
          renderItem={({ item }) => {
            const mine = item.senderId === me?.id;
            return (
              <View style={[styles.bubbleRow, { justifyContent: mine ? "flex-end" : "flex-start" }]}>
                <View style={[styles.bubble, { backgroundColor: mine ? theme.primary : theme.surface, borderColor: theme.border }]}>
                  <Text style={{ color: mine ? theme.primaryText : theme.text, fontSize: 16 }}>{item.body}</Text>
                  <Text style={{ color: mine ? theme.primaryText : theme.textMuted, fontSize: 11, opacity: 0.8, alignSelf: "flex-end" }}>
                    {formatTime(item.createdAt)}
                    {mine && item.id === lastOwnId ? (item.id <= otherRead ? " · Lu" : " · Envoyé") : ""}
                  </Text>
                </View>
              </View>
            );
          }}
        />
      )}
      <ErrorText message={error} />
      <View style={[styles.composer, { backgroundColor: theme.surface, borderTopColor: theme.border }]}>
        <TextInput
          accessibilityLabel="Votre message"
          testID="message-input"
          value={text}
          onChangeText={setText}
          placeholder="Écrire un message"
          placeholderTextColor={theme.textMuted}
          multiline
          maxLength={2000}
          style={[styles.input, { color: theme.text, backgroundColor: theme.background, borderColor: theme.border }]}
        />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Envoyer"
          disabled={sending || !text.trim()}
          onPress={() => void send()}
          testID="send-message"
          style={[styles.send, { backgroundColor: theme.primary, opacity: sending || !text.trim() ? 0.5 : 1 }]}
        >
          <Text style={{ color: theme.primaryText, fontWeight: "800" }}>Envoyer</Text>
        </Pressable>
      </View>
      <ActionSheet
        visible={menuOpen}
        title={user.firstName}
        onClose={() => setMenuOpen(false)}
        actions={[
          { label: "Voir le profil", onPress: () => navigation.navigate("UserProfile", { user, conversationId, matchId }) },
          {
            label: "Supprimer la conversation",
            onPress: () =>
              confirmAction(
                "Supprimer la conversation ?",
                "Elle disparaît pour vous seulement.",
                "Supprimer",
                () => void api.hideConversation(conversationId).then(leave)
              )
          },
          {
            label: "Défaire le match",
            onPress: () =>
              confirmAction(
                "Défaire le match ?",
                "La conversation sera supprimée pour vous deux.",
                "Défaire",
                () => void api.unmatch(matchId).then(leave)
              )
          },
          { label: "Signaler", onPress: () => setReporting(true) },
          {
            label: "Bloquer",
            destructive: true,
            onPress: () =>
              confirmAction(
                `Bloquer ${user.firstName} ?`,
                "Vous ne verrez plus ce profil et il ne verra plus le vôtre.",
                "Bloquer",
                () => void api.block(user.id).then(leave)
              )
          }
        ]}
      />
      <ReportModal
        userId={reporting ? user.id : null}
        name={user.firstName}
        onClose={(sent) => {
          setReporting(false);
          if (sent) {
            notify("Merci", "Votre signalement a été envoyé.");
          }
        }}
      />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  bubbleRow: { flexDirection: "row" },
  bubble: {
    maxWidth: "80%",
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderWidth: StyleSheet.hairlineWidth,
    gap: 2
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderWidth: 1,
    borderRadius: radius.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    fontSize: 16
  },
  send: { minHeight: 44, borderRadius: radius.pill, paddingHorizontal: spacing.lg, alignItems: "center", justifyContent: "center" }
});
