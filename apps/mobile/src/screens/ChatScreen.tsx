import { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { Alert, FlatList, KeyboardAvoidingView, Platform, Pressable, TextInput, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { chatApi, discoverApi, moderationApi } from "../api";
import type { Message, MessagePage } from "../api/types";
import { useToast } from "../components/feedback";
import { EmptyState, ErrorState, Loading, Text } from "../components/ui";
import { errorMessage } from "../lib/errors";
import { formatTime } from "../lib/format";
import { isRead, mergePages, upsertMessage } from "../lib/messages";
import type { RootStackParamList } from "../navigation/types";
import { useAuth } from "../state/auth";
import { keys, useMatches } from "../state/queries";
import { radius, spacing, useColors } from "../theme";

type Props = NativeStackScreenProps<RootStackParamList, "Chat">;

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId, userId, name } = route.params;
  const { user } = useAuth();
  const colors = useColors();
  const toast = useToast();
  const queryClient = useQueryClient();
  const matches = useMatches();
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);

  const query = useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam || undefined),
    initialPageParam: "",
    getNextPageParam: (last) => last.nextCursor || undefined,
  });

  const messages = useMemo(() => mergePages((query.data?.pages ?? []).map((p) => p.items)), [query.data]);
  const otherReadAt = query.data?.pages[0]?.otherLastReadAt ?? "1970-01-01T00:00:00Z";
  const lastOwnId = messages.find((m) => m.senderId === user?.id)?.id;
  const newestId = messages[0]?.id;

  const markRead = useCallback(() => {
    chatApi
      .markRead(conversationId)
      .then(() => queryClient.invalidateQueries({ queryKey: keys.conversations }))
      .catch(() => undefined);
  }, [conversationId, queryClient]);

  // Marque comme lu a l'ouverture et chaque fois qu'un nouveau message arrive pendant que l'ecran est ouvert.
  useEffect(() => {
    if (newestId !== undefined) {
      markRead();
    }
  }, [newestId, markRead]);

  const moderate = useCallback(() => {
    const matchId = matches.data?.find((m) => m.conversationId === conversationId)?.id;
    Alert.alert(name, undefined, [
      { text: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { userId }) },
      {
        text: "Supprimer la conversation",
        onPress: () =>
          Alert.alert(
            "Supprimer la conversation ?",
            "Elle disparaîtra de votre liste. Votre correspondant la conserve.",
            [
              { text: "Annuler", style: "cancel" },
              {
                text: "Supprimer",
                style: "destructive",
                onPress: () =>
                  chatApi
                    .clear(conversationId)
                    .then(() => {
                      void queryClient.invalidateQueries({ queryKey: keys.conversations });
                      navigation.goBack();
                    })
                    .catch((e) => toast.show(errorMessage(e), "error")),
              },
            ],
          ),
      },
      ...(matchId
        ? [
            {
              text: "Annuler le match",
              style: "destructive" as const,
              onPress: () =>
                discoverApi
                  .unmatch(matchId)
                  .then(() => {
                    void queryClient.invalidateQueries({ queryKey: keys.matches });
                    void queryClient.invalidateQueries({ queryKey: keys.conversations });
                    navigation.goBack();
                  })
                  .catch((e) => toast.show(errorMessage(e), "error")),
            },
          ]
        : []),
      { text: "Signaler", onPress: () => navigation.navigate("Report", { userId, name }) },
      {
        text: "Bloquer",
        style: "destructive",
        onPress: () =>
          Alert.alert(`Bloquer ${name} ?`, "Le match et la conversation seront supprimés.", [
            { text: "Annuler", style: "cancel" },
            {
              text: "Bloquer",
              style: "destructive",
              onPress: () =>
                moderationApi
                  .block(userId)
                  .then(() => {
                    void queryClient.invalidateQueries({ queryKey: keys.matches });
                    void queryClient.invalidateQueries({ queryKey: keys.conversations });
                    toast.show(`${name} est bloqué(e).`, "success");
                    navigation.goBack();
                  })
                  .catch((e) => toast.show(errorMessage(e), "error")),
            },
          ]),
      },
      { text: "Fermer", style: "cancel" },
    ]);
  }, [conversationId, matches.data, name, navigation, queryClient, toast, userId]);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => (
        <Pressable
          accessibilityLabel="Options de la conversation"
          accessibilityRole="button"
          hitSlop={10}
          onPress={moderate}
        >
          <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      ),
    });
  }, [navigation, name, moderate, colors.text]);

  const send = async () => {
    const body = draft.trim();
    if (!body || sending) {
      return;
    }
    setSending(true);
    setDraft("");
    try {
      const message = await chatApi.send(conversationId, body);
      queryClient.setQueryData<InfiniteData<MessagePage>>(keys.messages(conversationId), (data) => {
        if (!data || data.pages.length === 0) {
          return data;
        }
        const [first, ...rest] = data.pages;
        return { ...data, pages: [{ ...first, items: upsertMessage(first.items, message) }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
    } catch (e) {
      setDraft(body); // ne perd pas le texte saisi
      toast.show(errorMessage(e), "error");
    } finally {
      setSending(false);
    }
  };

  if (query.isLoading) {
    return <Loading />;
  }
  if (query.isError) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }

  const renderItem = ({ item }: { item: Message }) => {
    const mine = item.senderId === user?.id;
    return (
      <View style={{ alignItems: mine ? "flex-end" : "flex-start", marginBottom: spacing.sm }}>
        <View
          style={{
            maxWidth: "80%",
            backgroundColor: mine ? colors.primary : colors.surface,
            borderColor: mine ? colors.primary : colors.border,
            borderWidth: 1,
            borderRadius: radius.lg,
            paddingVertical: spacing.sm + 2,
            paddingHorizontal: spacing.lg,
          }}
        >
          <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{item.body}</Text>
        </View>
        <Text muted style={{ marginTop: 2, marginHorizontal: spacing.xs }} variant="caption">
          {formatTime(item.createdAt)}
          {mine && item.id === lastOwnId && isRead(item, otherReadAt) ? " · Lu" : ""}
        </Text>
      </View>
    );
  };

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      style={{ flex: 1, backgroundColor: colors.background }}
    >
      <FlatList
        ListEmptyComponent={
          <View style={{ transform: [{ scaleY: -1 }] }}>
            <EmptyState
              icon="hand-left-outline"
              message={`Écrivez le premier message à ${name}.`}
              title="C'est un match !"
            />
          </View>
        }
        contentContainerStyle={{ padding: spacing.lg, flexGrow: 1 }}
        data={messages}
        inverted
        keyExtractor={(m) => m.id}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) {
            void query.fetchNextPage();
          }
        }}
        renderItem={renderItem}
      />
      <View
        style={{
          flexDirection: "row",
          alignItems: "flex-end",
          padding: spacing.md,
          borderTopWidth: 1,
          borderTopColor: colors.border,
          backgroundColor: colors.surface,
        }}
      >
        <TextInput
          accessibilityLabel="Votre message"
          maxLength={2000}
          multiline
          onChangeText={setDraft}
          placeholder="Votre message…"
          placeholderTextColor={colors.textMuted}
          style={{
            flex: 1,
            maxHeight: 120,
            minHeight: 44,
            color: colors.text,
            backgroundColor: colors.background,
            borderRadius: radius.lg,
            paddingHorizontal: spacing.lg,
            paddingTop: 12,
            paddingBottom: 12,
            fontSize: 16,
          }}
          value={draft}
        />
        <Pressable
          accessibilityLabel="Envoyer"
          accessibilityRole="button"
          disabled={!draft.trim() || sending}
          onPress={() => void send()}
          style={{
            marginLeft: spacing.sm,
            width: 44,
            height: 44,
            borderRadius: 22,
            backgroundColor: colors.primary,
            alignItems: "center",
            justifyContent: "center",
            opacity: !draft.trim() || sending ? 0.5 : 1,
          }}
          testID="chat-send"
        >
          <Ionicons color={colors.onPrimary} name="send" size={20} />
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}
