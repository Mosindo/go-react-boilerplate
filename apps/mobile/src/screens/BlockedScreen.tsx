import React from "react";
import { FlatList, StyleSheet } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { describeError } from "../api/client";
import { keys } from "../api/keys";
import { unblockUser } from "../api/platform";
import { useBlocked } from "../hooks/data";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Button, ListItem, spacing } from "../shared/ui";

export default function BlockedScreen() {
  const client = useQueryClient();
  const query = useBlocked();
  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.blocked });
      showToast("Utilisateur débloqué.", "success");
    },
    onError: (error) => showToast(describeError(error), "error")
  });

  if (query.isLoading) return <LoadingView fullScreen />;
  if (query.isError) return <ErrorView message="Impossible de charger la liste." onAction={() => void query.refetch()} />;
  if (!query.data?.length)
    return <EmptyView icon="shield-checkmark-outline" message="Les personnes que vous bloquez apparaissent ici." title="Personne n'est bloqué" />;

  return (
    <FlatList
      contentContainerStyle={styles.list}
      data={query.data}
      keyExtractor={(item) => item.userId}
      renderItem={({ item }) => (
        <ListItem
          chevron={false}
          right={
            <Button disabled={unblock.isPending} onPress={() => unblock.mutate(item.userId)} size="sm" variant="outline">
              Débloquer
            </Button>
          }
          subtitle={`Bloqué le ${new Date(item.blockedAt).toLocaleDateString("fr-FR")}`}
          title={item.firstName || "Utilisateur"}
        />
      )}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, width: "100%", maxWidth: 560, alignSelf: "center" }
});
