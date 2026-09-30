import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { Button, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { notificationsApi } from "../../lib/api/endpoints";
import type { AppNotification } from "../../lib/api/types";
import { formatRelative } from "../../lib/format";
import { queryKeys } from "../../lib/queryClient";
import type { AppStackParamList } from "../../navigation/types";
import { useNotifications } from "./hooks";

export default function NotificationsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const client = useQueryClient();
  const { colors, radii } = useTheme();
  const { data, isLoading, error, refetch, isRefetching } = useNotifications();

  const refresh = () => void client.invalidateQueries({ queryKey: queryKeys.notifications });

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      await notificationsApi.markRead(n.id).catch(() => undefined);
      refresh();
    }
    if (n.data.conversationId) {
      navigation.navigate("Chat", { conversationId: n.data.conversationId });
    }
  };

  if (isLoading) return <LoadingState />;
  if (error || !data) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  return (
    <Screen testID="notifications-screen">
      <View style={styles.header}>
        <Text variant="title">Notifications</Text>
        {data.unreadCount > 0 ? (
          <Button label="Tout marquer comme lu" onPress={() => void notificationsApi.markAllRead().then(refresh)} size="sm" variant="ghost" />
        ) : null}
      </View>
      <FlatList
        ListEmptyComponent={<EmptyState icon="notifications-outline" message="Vos nouveaux matchs et messages apparaîtront ici." title="Rien de nouveau" />}
        contentContainerStyle={styles.list}
        data={data.notifications}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl onRefresh={() => void refetch()} refreshing={isRefetching} tintColor={colors.primary} />}
        renderItem={({ item }) => (
          <Pressable
            accessibilityHint={item.data.conversationId ? "Ouvre la conversation" : undefined}
            accessibilityRole="button"
            onPress={() => void open(item)}
            style={({ pressed }) => [
              styles.row,
              { backgroundColor: pressed ? colors.surfaceMuted : item.isRead ? "transparent" : colors.primarySoft, borderRadius: radii.md }
            ]}
            testID={`notification-${item.type}`}
          >
            <View style={[styles.icon, { backgroundColor: colors.surface }]}>
              <Ionicons color={colors.primary} name={item.type === "match" ? "heart" : "chatbubble"} size={18} />
            </View>
            <View style={styles.text}>
              <Text variant="label">{item.title}</Text>
              <Text tone="muted" variant="caption">
                {item.body}
              </Text>
            </View>
            <Text tone="subtle" variant="caption">
              {formatRelative(item.createdAt)}
            </Text>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingHorizontal: 20, paddingVertical: 8 },
  list: { padding: 12, gap: 6, flexGrow: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: 12, padding: 12 },
  icon: { width: 38, height: 38, borderRadius: 19, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 }
});
