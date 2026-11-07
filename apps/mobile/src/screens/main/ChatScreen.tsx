import React, { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AppText, Banner, Button, ErrorState, Loading, Row, Screen } from "../../components/ui";
import { ReportSheet, Sheet } from "../../components/Modals";
import { chatApi, discoveryApi, safetyApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { useAuth } from "../../auth/AuthContext";
import { formatTime, isReadByPeer } from "../../utils/format";
import { font, radius, spacing, useTheme } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";
import type { Message, MessagesPage } from "../../api/types";

export default function ChatScreen({ route, navigation }: NativeStackScreenProps<MainStackParams, "Chat">) {
  const { conversationId, peerName, peerId } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const qc = useQueryClient();
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);

  const q = useInfiniteQuery({
    queryKey: ["messages", conversationId],
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last: MessagesPage) => last.nextBefore ?? undefined,
  });

  // Newest first, as expected by an inverted list.
  const messages = useMemo<Message[]>(
    () => (q.data?.pages ?? []).flatMap((p) => [...p.messages].reverse()),
    [q.data],
  );
  const peerReadAt = q.data?.pages[0]?.peerReadAt ?? null;
  const lastMineRead = messages.find((m) => m.senderId === user?.id && isReadByPeer(m.createdAt, peerReadAt));

  const latestFromPeer = messages.find((m) => m.senderId !== user?.id)?.id;
  const markRead = useCallback(() => {
    chatApi
      .markRead(conversationId)
      .then(() => qc.invalidateQueries({ queryKey: ["conversations"] }))
      .catch(() => undefined);
  }, [conversationId, qc]);
  useEffect(() => {
    if (latestFromPeer !== undefined) markRead();
  }, [latestFromPeer, markRead]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (msg) => {
      qc.setQueryData<{ pages: MessagesPage[]; pageParams: unknown[] }>(
        ["messages", conversationId],
        (old) => {
          const first = old?.pages[0];
          if (!old || !first || first.messages.some((m) => m.id === msg.id)) return old;
          return { ...old, pages: [{ ...first, messages: [...first.messages, msg] }, ...old.pages.slice(1)] };
        },
      );
      void qc.invalidateQueries({ queryKey: ["conversations"] });
    },
    onError: (e, body) => {
      setText(body);
      setError(errorMessage(e));
    },
  });

  const submit = () => {
    const body = text.trim();
    if (!body || send.isPending) return;
    setError(null);
    setText("");
    send.mutate(body);
  };

  const conversations = useQuery({ queryKey: ["conversations"], queryFn: chatApi.conversations });
  const matchId = conversations.data?.find((c) => c.id === conversationId)?.matchId;

  const done = () => {
    void qc.invalidateQueries({ queryKey: ["conversations"] });
    void qc.invalidateQueries({ queryKey: ["matches"] });
    navigation.goBack();
  };
  const clear = useMutation({
    mutationFn: () => chatApi.clear(conversationId),
    onSuccess: done,
    onError: (e) => setError(errorMessage(e)),
  });
  const unmatch = useMutation({
    mutationFn: () =>
      matchId ? discoveryApi.unmatch(matchId) : Promise.reject(new Error("match introuvable")),
    onSuccess: done,
    onError: (e) => setError(errorMessage(e)),
  });
  const block = useMutation({
    mutationFn: () => safetyApi.block(peerId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["blocks"] });
      done();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  useLayoutEffect(() => {
    navigation.setOptions({
      title: peerName,
      headerRight: () => (
        <Pressable
          testID="chat-menu"
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          onPress={() => setMenu(true)}
          style={{ padding: spacing.sm }}
        >
          <AppText style={{ fontSize: 22, fontWeight: "800" }}>⋯</AppText>
        </Pressable>
      ),
    });
  }, [navigation, peerName]);

  if (q.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (q.isError)
    return (
      <Screen>
        <ErrorState message="Cette conversation n'est plus disponible." onRetry={() => navigation.goBack()} />
      </Screen>
    );

  return (
    <Screen padded={false} edges={["bottom"]}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={90}
      >
        {error ? (
          <View style={{ padding: spacing.md }}>
            <Banner message={error} />
          </View>
        ) : null}
        <FlatList
          testID="message-list"
          inverted
          data={messages}
          keyExtractor={(m) => String(m.id)}
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm }}
          onEndReached={() => q.hasNextPage && !q.isFetchingNextPage && void q.fetchNextPage()}
          onEndReachedThreshold={0.4}
          ListEmptyComponent={
            <View style={{ transform: [{ scaleY: -1 }], padding: spacing.xl }}>
              <AppText variant="muted" style={{ textAlign: "center" }}>
                C&apos;est un match ! Envoyez le premier message à {peerName}.
              </AppText>
            </View>
          }
          renderItem={({ item }) => {
            const mine = item.senderId === user?.id;
            return (
              <View style={{ alignSelf: mine ? "flex-end" : "flex-start", maxWidth: "80%", gap: 2 }}>
                <View
                  style={{
                    backgroundColor: mine ? colors.primary : colors.surfaceAlt,
                    borderRadius: radius.lg,
                    paddingHorizontal: spacing.md,
                    paddingVertical: spacing.sm,
                  }}
                >
                  <AppText style={{ color: mine ? colors.primaryText : colors.text }}>{item.body}</AppText>
                </View>
                <AppText variant="muted" style={{ fontSize: font.xs, textAlign: mine ? "right" : "left" }}>
                  {formatTime(item.createdAt)}
                  {mine && lastMineRead?.id === item.id ? " · Lu" : ""}
                </AppText>
              </View>
            );
          }}
        />
        <Row
          style={{
            padding: spacing.md,
            borderTopWidth: 1,
            borderColor: colors.border,
            backgroundColor: colors.bg,
          }}
        >
          <TextInput
            testID="message-input"
            accessibilityLabel="Votre message"
            value={text}
            onChangeText={setText}
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textMuted}
            multiline
            maxLength={2000}
            onSubmitEditing={submit}
            style={{
              flex: 1,
              minHeight: 44,
              maxHeight: 120,
              borderRadius: radius.lg,
              borderWidth: 1,
              borderColor: colors.border,
              backgroundColor: colors.surface,
              color: colors.text,
              paddingHorizontal: spacing.md,
              paddingVertical: spacing.sm,
              fontSize: font.md,
            }}
          />
          <Button
            testID="send-message"
            title="Envoyer"
            onPress={submit}
            disabled={!text.trim()}
            style={{ minHeight: 44, paddingHorizontal: spacing.lg }}
          />
        </Row>
      </KeyboardAvoidingView>

      <Sheet visible={menu} onClose={() => setMenu(false)}>
        <AppText variant="heading">{peerName}</AppText>
        <Button
          title="Voir le profil"
          variant="secondary"
          onPress={() => {
            setMenu(false);
            navigation.navigate("ProfileDetail", { userId: peerId });
          }}
        />
        <Button
          testID="clear-conversation"
          title="Supprimer la conversation (pour moi)"
          variant="secondary"
          loading={clear.isPending}
          onPress={() => {
            setMenu(false);
            clear.mutate();
          }}
        />
        <Button
          testID="unmatch"
          title="Annuler le match"
          variant="secondary"
          loading={unmatch.isPending}
          onPress={() => {
            setMenu(false);
            unmatch.mutate();
          }}
        />
        <Button
          testID="block-from-chat"
          title="Bloquer"
          variant="secondary"
          loading={block.isPending}
          onPress={() => {
            setMenu(false);
            block.mutate();
          }}
        />
        <Button
          testID="report-from-chat"
          title="Signaler"
          variant="danger"
          onPress={() => {
            setMenu(false);
            setReporting(true);
          }}
        />
        <Button title="Fermer" variant="ghost" onPress={() => setMenu(false)} />
      </Sheet>
      <ReportSheet
        visible={reporting}
        userId={peerId}
        name={peerName}
        onClose={() => setReporting(false)}
        onDone={(blocked) => {
          setReporting(false);
          if (blocked) done();
          else setError("Merci, votre signalement a été envoyé.");
        }}
      />
    </Screen>
  );
}
