import { FlatList, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { moderationApi } from "../api";
import { useToast } from "../components/feedback";
import { Button, EmptyState, ErrorState, Loading, Text } from "../components/ui";
import { errorMessage } from "../lib/errors";
import { keys } from "../state/queries";
import { spacing, useColors } from "../theme";

export function BlockedScreen() {
  const colors = useColors();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.blocks, queryFn: moderationApi.blocks });
  const unblock = useMutation({
    mutationFn: (userId: string) => moderationApi.unblock(userId),
    onSuccess: () => {
      toast.show("Profil débloqué", "success");
      void queryClient.invalidateQueries({ queryKey: keys.blocks });
    },
    onError: (e) => toast.show(errorMessage(e), "error"),
  });

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
          icon="shield-checkmark-outline"
          message="Vous n'avez bloqué personne."
          title="Aucun profil bloqué"
        />
      }
      contentContainerStyle={{ flexGrow: 1, padding: spacing.lg }}
      data={query.data}
      keyExtractor={(b) => b.userId}
      renderItem={({ item }) => (
        <View
          style={{
            flexDirection: "row",
            alignItems: "center",
            justifyContent: "space-between",
            paddingVertical: spacing.md,
            borderBottomWidth: 1,
            borderBottomColor: colors.border,
          }}
        >
          <Text variant="label">{item.firstName || "Profil supprimé"}</Text>
          <Button
            loading={unblock.isPending}
            onPress={() => unblock.mutate(item.userId)}
            title="Débloquer"
            variant="secondary"
          />
        </View>
      )}
    />
  );
}
