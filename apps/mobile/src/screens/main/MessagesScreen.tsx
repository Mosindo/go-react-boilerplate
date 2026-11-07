import React from "react";
import { FlatList, Pressable, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { AppText, Avatar, EmptyState, ErrorState, Loading, Row, Screen } from "../../components/ui";
import { chatApi, discoveryApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { formatTime } from "../../utils/format";
import { radius, spacing, useTheme } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";

export default function MessagesScreen() {
  const { colors } = useTheme();
  const nav = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const convs = useQuery({ queryKey: ["conversations"], queryFn: chatApi.conversations });
  const matches = useQuery({ queryKey: ["matches"], queryFn: discoveryApi.matches });

  const talked = new Set((convs.data ?? []).filter((c) => c.lastMessage).map((c) => c.id));
  const fresh = (matches.data ?? []).filter((m) => !talked.has(m.conversationId));

  const header = (
    <View style={{ gap: spacing.md, marginBottom: spacing.md }}>
      {fresh.length > 0 ? (
        <>
          <AppText variant="heading">Nouveaux matchs</AppText>
          <FlatList
            horizontal
            data={fresh}
            keyExtractor={(m) => m.id}
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={{ gap: spacing.md }}
            renderItem={({ item }) => (
              <Pressable
                testID={`new-match-${item.user.firstName}`}
                accessibilityRole="button"
                accessibilityLabel={`Nouveau match : ${item.user.firstName}`}
                style={{ alignItems: "center", gap: spacing.xs }}
                onPress={() =>
                  nav.navigate("Chat", {
                    conversationId: item.conversationId,
                    peerName: item.user.firstName,
                    peerId: item.user.userId,
                  })
                }
              >
                <Avatar name={item.user.firstName} uri={item.user.photos[0]?.url} size={68} />
                <AppText variant="small">{item.user.firstName}</AppText>
              </Pressable>
            )}
          />
        </>
      ) : null}
      <AppText variant="heading">Conversations</AppText>
    </View>
  );

  return (
    <Screen>
      <AppText variant="title" style={{ paddingVertical: spacing.sm }}>
        Messages
      </AppText>
      {convs.isLoading ? (
        <Loading />
      ) : convs.isError ? (
        <ErrorState message={errorMessage(convs.error)} onRetry={() => void convs.refetch()} />
      ) : (
        <FlatList
          data={(convs.data ?? []).filter((c) => c.lastMessage)}
          keyExtractor={(c) => c.id}
          ListHeaderComponent={header}
          refreshing={convs.isRefetching}
          onRefresh={() => {
            void convs.refetch();
            void matches.refetch();
          }}
          ListEmptyComponent={
            <EmptyState
              title={fresh.length ? "Brisez la glace" : "Aucune conversation"}
              message={
                fresh.length
                  ? "Écrivez le premier message à l'un de vos nouveaux matchs."
                  : "Likez des profils : dès qu'un like est réciproque, vous pourrez discuter ici."
              }
            />
          }
          renderItem={({ item }) => (
            <Pressable
              testID={`conversation-${item.peer.firstName}`}
              accessibilityRole="button"
              onPress={() =>
                nav.navigate("Chat", {
                  conversationId: item.id,
                  peerName: item.peer.firstName,
                  peerId: item.peer.userId,
                })
              }
              style={{ paddingVertical: spacing.md, borderBottomWidth: 1, borderColor: colors.border }}
            >
              <Row>
                <Avatar name={item.peer.firstName} uri={item.peer.photoUrl} />
                <View style={{ flex: 1, gap: 2 }}>
                  <Row style={{ justifyContent: "space-between" }}>
                    <AppText style={{ fontWeight: "700" }}>{item.peer.firstName}</AppText>
                    <AppText variant="muted">
                      {item.lastMessage ? formatTime(item.lastMessage.createdAt) : ""}
                    </AppText>
                  </Row>
                  <Row style={{ justifyContent: "space-between" }}>
                    <AppText
                      variant="muted"
                      numberOfLines={1}
                      style={{ flex: 1, fontWeight: item.unreadCount ? "700" : "400" }}
                    >
                      {item.lastMessage?.body}
                    </AppText>
                    {item.unreadCount > 0 ? (
                      <View
                        style={{
                          backgroundColor: colors.primary,
                          borderRadius: radius.pill,
                          minWidth: 22,
                          paddingHorizontal: 6,
                          alignItems: "center",
                        }}
                      >
                        <AppText variant="small" style={{ color: colors.primaryText, fontWeight: "800" }}>
                          {item.unreadCount}
                        </AppText>
                      </View>
                    ) : null}
                  </Row>
                </View>
              </Row>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}
