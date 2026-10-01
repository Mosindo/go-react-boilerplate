import React from "react";
import { FlatList, StyleSheet } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Button, ListItem, spacing } from "../shared/ui";

export default function BlockedUsersScreen() {
  const queryClient = useQueryClient();
  const { data, isLoading, isError, refetch } = useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.blocks });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      showToast("Personne débloquée.");
    }
  });

  if (isLoading) {
    return <LoadingView fullScreen />;
  }
  if (isError || !data) {
    return <ErrorView message="Impossible de charger la liste." onAction={() => void refetch()} />;
  }
  return (
    <FlatList
      ListEmptyComponent={
        <EmptyView
          icon="shield-checkmark-outline"
          message="Les personnes que vous bloquez n'apparaîtront plus dans votre découverte ni vos messages."
          title="Personne n'est bloqué"
        />
      }
      contentContainerStyle={data.length === 0 ? styles.empty : undefined}
      data={data}
      keyExtractor={(b) => b.userId}
      renderItem={({ item }) => (
        <ListItem
          right={
            <Button
              disabled={unblock.isPending}
              fullWidth={false}
              label="Débloquer"
              onPress={() => unblock.mutate(item.userId)}
              size="sm"
              variant="outline"
            />
          }
          subtitle={`Bloqué le ${new Date(item.blockedAt).toLocaleDateString("fr-FR")}`}
          title={item.firstName || "Membre"}
        />
      )}
    />
  );
}

const styles = StyleSheet.create({ empty: { flexGrow: 1, padding: spacing.lg } });
