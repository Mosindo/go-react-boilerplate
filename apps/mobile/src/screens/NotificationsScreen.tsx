import { useEffect } from "react";
import { FlatList, Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../api";
import { EmptyState, ErrorState, Loading, Text } from "../components/ui";
import { errorMessage } from "../lib/errors";
import { formatRelative } from "../lib/format";
import { keys, useNotifications } from "../state/queries";
import { spacing, useColors } from "../theme";

export function NotificationsScreen() {
  const navigation = useNavigation();
  const colors = useColors();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const unread = query.data?.unreadCount ?? 0;

  // Marque tout comme lu a l'ouverture (apres affichage de l'etat non lu).
  useEffect(() => {
    if (unread === 0) {
      return;
    }
    const timer = setTimeout(() => {
      notificationsApi
        .markRead()
        .then(() => queryClient.invalidateQueries({ queryKey: keys.notifications }))
        .catch(() => undefined);
    }, 1500);
    return () => clearTimeout(timer);
  }, [unread, queryClient]);

  if (query.isLoading) {
    return <Loading />;
  }
  if (query.isError) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }

  return (
    <FlatList
      ListEmptyComponent={
        <EmptyState
          icon="notifications-off-outline"
          message="Les nouveaux matchs et messages apparaîtront ici."
          title="Rien pour l'instant"
        />
      }
      contentContainerStyle={{ flexGrow: 1 }}
      data={query.data?.items ?? []}
      keyExtractor={(n) => n.id}
      renderItem={({ item }) => (
        <Pressable
          accessibilityRole="button"
          onPress={() => navigation.navigate("Tabs")}
          style={{
            flexDirection: "row",
            alignItems: "center",
            padding: spacing.lg,
            backgroundColor: item.readAt ? "transparent" : colors.surfaceAlt,
            borderBottomWidth: 1,
            borderBottomColor: colors.border,
          }}
        >
          <Ionicons color={colors.primary} name={item.type === "match" ? "heart" : "chatbubble"} size={22} />
          <View style={{ flex: 1, marginLeft: spacing.md }}>
            <Text variant="label">
              {item.type === "match"
                ? `Vous avez un match avec ${item.actorName || "un membre"} !`
                : `Nouveau message de ${item.actorName || "un membre"}`}
            </Text>
            <Text muted variant="caption">
              {formatRelative(item.createdAt)}
            </Text>
          </View>
        </Pressable>
      )}
    />
  );
}
