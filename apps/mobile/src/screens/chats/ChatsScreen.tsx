import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { Conversation } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { AuthImage, ConversationList, ErrorState, ListSkeleton, Screen, Text } from "../../components";
import { useConversations } from "../../hooks/queries";
import { errorMessage } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";

export function ChatsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { user } = useAuth();
  const query = useConversations();
  const all = query.data?.pages.flatMap((p) => p.conversations) ?? [];
  const fresh = all.filter((c) => c.lastMessage === null);
  const talking = all.filter((c) => c.lastMessage !== null);

  const open = (c: Conversation): void =>
    navigation.navigate("Chat", { conversationId: c.id, name: c.user.firstName, userId: c.user.userId });

  if (query.isPending) {
    return <Screen padded={false} edges={[]}><ListSkeleton /></Screen>;
  }
  if (query.isError) {
    return <Screen edges={[]}><ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} /></Screen>;
  }

  const strip =
    fresh.length > 0 ? (
      <View style={styles.strip} testID="new-matches">
        <Text variant="label" tone="muted" style={styles.stripTitle}>
          New matches
        </Text>
        <FlatList
          horizontal
          data={fresh}
          keyExtractor={(c) => c.id}
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={{ gap: spacing.md, paddingHorizontal: spacing.lg }}
          renderItem={({ item }) => (
            <Pressable accessibilityRole="button" accessibilityLabel={`New match ${item.user.firstName}`} onPress={() => open(item)} testID={`new-match-${item.id}`} style={styles.match}>
              <AuthImage url={item.user.photoUrl} style={styles.matchPhoto} fallbackInitial={item.user.firstName} />
              <Text variant="caption" numberOfLines={1}>
                {item.user.firstName}
              </Text>
            </Pressable>
          )}
        />
        {talking.length > 0 ? (
          <Text variant="label" tone="muted" style={styles.stripTitle}>
            Messages
          </Text>
        ) : null}
      </View>
    ) : null;

  return (
    <Screen padded={false} edges={[]} testID="chats-screen">
      <ConversationList
        conversations={talking}
        myUserId={user?.id ?? ""}
        onOpen={open}
        header={strip}
        refreshing={query.isRefetching && !query.isFetchingNextPage}
        onRefresh={() => void query.refetch()}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  strip: { paddingTop: spacing.md, gap: spacing.sm },
  stripTitle: { paddingHorizontal: spacing.lg },
  match: { width: 72, alignItems: "center", gap: 4 },
  matchPhoto: { width: 68, height: 68, borderRadius: 34 }
});
