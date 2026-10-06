import React from "react";
import { FlatList, StyleSheet } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listBlocks, unblockUser } from "../api/platform";
import { describeError } from "../lib/errors";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Avatar, Button, ListItem, spacing } from "../shared/ui";

export default function BlockedUsersScreen() {
  const queryClient = useQueryClient();
  const blocks = useQuery({ queryKey: ["blocks"], queryFn: listBlocks });
  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["blocks"] });
      showToast("Personne débloquée", "success");
    },
    onError: (err) => showToast(describeError(err), "danger")
  });

  if (blocks.isLoading) {
    return <LoadingView fullScreen label="Chargement…" />;
  }
  if (blocks.isError) {
    return <ErrorView message={describeError(blocks.error)} onAction={() => void blocks.refetch()} />;
  }
  if ((blocks.data ?? []).length === 0) {
    return <EmptyView icon="shield-checkmark-outline" message="Vous n'avez bloqué personne." title="Aucune personne bloquée" />;
  }
  return (
    <FlatList
      contentContainerStyle={styles.list}
      data={blocks.data}
      keyExtractor={(item) => item.userId}
      renderItem={({ item }) => (
        <ListItem
          leading={<Avatar name={item.firstName || "?"} size={40} />}
          title={item.firstName || "Compte supprimé"}
          trailing={<Button fullWidth={false} label="Débloquer" onPress={() => unblock.mutate(item.userId)} size="sm" variant="outline" />}
        />
      )}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.xl }
});
