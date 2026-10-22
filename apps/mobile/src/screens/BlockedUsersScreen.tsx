import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { fetchBlocked, unblockUser } from "../api/safety";
import { ScreenBar } from "../components/ScreenBar";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, ListItem, spacing } from "../shared/ui";

export default function BlockedUsersScreen() {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["blocks"], queryFn: fetchBlocked });
  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["blocks"] });
      await queryClient.invalidateQueries({ queryKey: ["discover"] });
    }
  });

  return (
    <ScreenContainer contentMaxWidth={560} testID="blocked-screen">
      <ScreenBar title="Comptes bloqués" />
      {query.isLoading ? (
        <LoadingView label="Chargement…" />
      ) : query.isError ? (
        <ErrorView actionLabel="Réessayer" message={errorMessage(query.error)} onAction={() => void query.refetch()} />
      ) : (
        <FlatList
          ItemSeparatorComponent={() => <ListSeparator />}
          ListEmptyComponent={<EmptyView message="Les personnes que vous bloquez apparaissent ici." title="Aucun compte bloqué" />}
          contentContainerStyle={styles.list}
          data={query.data}
          keyExtractor={(item) => item.userId}
          renderItem={({ item }) => (
            <ListItem
              title={item.firstName || "Compte"}
              trailing={
                <Button
                  label="Débloquer"
                  loading={unblock.isPending && unblock.variables === item.userId}
                  onPress={() => unblock.mutate(item.userId)}
                  size="sm"
                  variant="outline"
                />
              }
            />
          )}
        />
      )}
    </ScreenContainer>
  );
}

function ListSeparator() {
  return <View style={styles.gap} />;
}

const styles = StyleSheet.create({ list: { paddingBottom: spacing.xl }, gap: { height: spacing.sm } });
