import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { ConversationItem } from "../api/types";
import { useUserActions } from "../components/UserActions";
import { useConversations } from "../hooks/data";
import { useMe } from "../hooks/data";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { formatRelative } from "../shared/format";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Avatar, Badge, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";

function Row({ item, mineId, onPress, onLongPress }: { item: ConversationItem; mineId: string; onPress: () => void; onLongPress: () => void }) {
  const { colors } = useTheme();
  const unread = item.unreadCount > 0;
  const last = item.lastMessage;
  const preview = last ? `${last.senderId === mineId ? "Vous : " : ""}${last.body}` : "";
  return (
    <Pressable
      accessibilityHint="Appui long pour supprimer la conversation"
      accessibilityLabel={`${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}. ${preview}`}
      accessibilityRole="button"
      onLongPress={onLongPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, { borderBottomColor: colors.border }, pressed ? { opacity: 0.7 } : null]}
    >
      <Avatar name={item.user.firstName} path={item.user.photos[0]?.thumbUrl} size="lg" />
      <View style={styles.body}>
        <View style={styles.top}>
          <Text numberOfLines={1} style={styles.name} weight={unread ? "bold" : "semibold"}>
            {item.user.firstName}
          </Text>
          <Text tone={unread ? "primary" : "subtle"} variant="caption">
            {formatRelative(item.lastMessageAt)}
          </Text>
        </View>
        <View style={styles.top}>
          <Text numberOfLines={1} style={styles.name} tone={unread ? "default" : "muted"} weight={unread ? "semibold" : "regular"}>
            {preview}
          </Text>
          <Badge count={item.unreadCount} />
        </View>
      </View>
    </Pressable>
  );
}

export default function ConversationsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { data: me } = useMe();
  const actions = useUserActions();
  const query = useConversations();
  const items = query.data?.pages.flatMap((page) => page.conversations) ?? [];

  return (
    <SafeAreaLayout edges={["top"]}>
      <Header title="Messages" />
      {query.isLoading ? (
        <LoadingView fullScreen />
      ) : query.isError ? (
        <ErrorView message="Impossible de charger vos conversations." onAction={() => void query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyView icon="chatbubbles-outline" message="Écrivez à l'un de vos matchs pour lancer la conversation." title="Aucune conversation" />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={items}
          keyExtractor={(item) => item.id}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          onRefresh={() => void query.refetch()}
          refreshing={query.isRefetching && !query.isFetchingNextPage}
          renderItem={({ item }) => (
            <Row
              item={item}
              mineId={me?.user.id ?? ""}
              onLongPress={() => void actions.hideConversation(item.id)}
              onPress={() =>
                navigation.navigate("Chat", {
                  conversationId: item.id,
                  matchId: item.matchId,
                  user: { id: item.user.id, firstName: item.user.firstName, photoPath: item.user.photos[0]?.thumbUrl }
                })
              }
            />
          )}
        />
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  list: { paddingHorizontal: spacing.lg, width: "100%", maxWidth: 560, alignSelf: "center" },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.md, borderBottomWidth: StyleSheet.hairlineWidth },
  body: { flex: 1, gap: 2 },
  top: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.sm },
  name: { flex: 1 }
});
