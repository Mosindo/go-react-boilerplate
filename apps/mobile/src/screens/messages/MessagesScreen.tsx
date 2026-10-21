import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import type { Conversation } from "../../api/types";
import { BellButton } from "../../components/BellButton";
import { useConversations } from "../../hooks/queries";
import { formatRelative } from "../../lib/format";
import type { RootStackParamList } from "../../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../../shared/feedback";
import { Header, ScreenContainer } from "../../shared/layout";
import { Avatar, Badge, ListItem, Text } from "../../shared/ui";
import { spacing } from "../../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export function MessagesScreen() {
  const navigation = useNavigation<Nav>();
  const query = useConversations();
  const all = query.data ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talking = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) => navigation.navigate("Chat", { conversationId: c.id, userId: c.user.id, name: c.user.firstName });

  let body: React.ReactNode;
  if (query.isPending) body = <LoadingView />;
  else if (query.isError) body = <ErrorView message={query.error.message} onRetry={() => void query.refetch()} />;
  else if (all.length === 0) {
    body = (
      <EmptyView
        testID="messages-empty"
        icon="chatbubbles-outline"
        title="Pas encore de match"
        message="Quand l'intérêt est réciproque, votre conversation apparaît ici."
      />
    );
  } else {
    body = (
      <FlatList
        data={talking}
        keyExtractor={(c) => c.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} />}
        ListHeaderComponent={
          fresh.length > 0 ? (
            <View style={styles.rail}>
              <Text variant="label" tone="muted">
                NOUVEAUX MATCHS
              </Text>
              <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.railRow}>
                {fresh.map((c) => (
                  <Pressable
                    key={c.id}
                    testID={`new-match-${c.user.firstName}`}
                    accessibilityRole="button"
                    accessibilityLabel={`Nouveau match : ${c.user.firstName}`}
                    onPress={() => open(c)}
                    style={styles.railItem}
                  >
                    <Avatar name={c.user.firstName} uri={c.user.thumbUrl} size={64} />
                    <Text variant="caption" numberOfLines={1} style={{ fontWeight: "600" }}>
                      {c.user.firstName}
                    </Text>
                  </Pressable>
                ))}
              </ScrollView>
            </View>
          ) : null
        }
        ListEmptyComponent={
          <Text tone="muted" center style={{ marginTop: spacing.xl }}>
            Dites bonjour à l'un de vos matchs 👋
          </Text>
        }
        renderItem={({ item }) => (
          <ListItem
            testID={`conversation-${item.user.firstName}`}
            title={item.user.firstName}
            subtitle={
              item.lastMessage ? `${item.lastMessage.senderId === item.user.id ? "" : "Vous : "}${item.lastMessage.body}` : undefined
            }
            left={<Avatar name={item.user.firstName} uri={item.user.thumbUrl} size={52} />}
            right={
              <View style={styles.meta}>
                {item.lastMessage ? (
                  <Text variant="caption" tone="muted">
                    {formatRelative(item.lastMessage.createdAt)}
                  </Text>
                ) : null}
                <Badge count={item.unreadCount} />
              </View>
            }
            onPress={() => open(item)}
          />
        )}
      />
    );
  }

  return (
    <ScreenContainer edges={["top", "left", "right"]} testID="messages-screen">
      <Header title="Messages" right={<BellButton />} />
      <View style={styles.flex}>{body}</View>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  rail: { gap: spacing.sm, marginBottom: spacing.lg },
  railRow: { gap: spacing.lg },
  railItem: { alignItems: "center", gap: spacing.xs, width: 72 },
  meta: { alignItems: "flex-end", gap: spacing.xs }
});
