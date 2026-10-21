import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList } from "react-native";
import { fetchBlocks, unblockUser } from "../../api/safety";
import { keys } from "../../hooks/queries";
import { EmptyView, ErrorView, LoadingView, showError, showToast } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Avatar, Button, ListItem } from "../../shared/ui";

export function BlockedUsersScreen() {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.blocks, queryFn: fetchBlocks });
  const unblock = useMutation({
    mutationFn: (userId: string) => unblockUser(userId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.blocks });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      showToast("Personne débloquée.", "success");
    },
    onError: (err) => showError(err instanceof Error ? err.message : "Action impossible.")
  });

  let body: React.ReactNode;
  if (query.isPending) body = <LoadingView />;
  else if (query.isError) body = <ErrorView message={query.error.message} onRetry={() => void query.refetch()} />;
  else if (query.data.length === 0)
    body = (
      <EmptyView
        testID="blocked-empty"
        icon="shield-checkmark-outline"
        title="Personne n'est bloqué"
        message="Les personnes que vous bloquez apparaissent ici."
      />
    );
  else {
    body = (
      <FlatList
        data={query.data}
        keyExtractor={(b) => b.user.id}
        renderItem={({ item }) => (
          <ListItem
            title={item.user.firstName || "Compte supprimé"}
            left={<Avatar name={item.user.firstName || "?"} uri={item.user.thumbUrl} />}
            right={
              <Button
                testID={`unblock-${item.user.id}`}
                label="Débloquer"
                variant="secondary"
                onPress={() => unblock.mutate(item.user.id)}
              />
            }
          />
        )}
      />
    );
  }
  return (
    <ScreenContainer edges={["bottom", "left", "right"]} testID="blocked-screen">
      {body}
    </ScreenContainer>
  );
}
