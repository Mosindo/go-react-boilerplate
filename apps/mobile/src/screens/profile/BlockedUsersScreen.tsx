import React from "react";
import { FlatList, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { keys } from "../../api/keys";
import { fetchBlocks, unblockUser } from "../../api/social";
import { Button, EmptyState, ErrorState, ListSkeleton, Screen, Text, useToast } from "../../components";
import { errorMessage } from "../../lib/format";
import { spacing } from "../../theme";

export function BlockedUsersScreen() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.blocks, queryFn: fetchBlocks });
  const unblock = useMutation({
    mutationFn: (userId: string) => unblockUser(userId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.blocks });
      await queryClient.invalidateQueries({ queryKey: keys.discover });
      toast.show("Unblocked");
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  if (query.isPending) return <Screen padded={false}><ListSkeleton rows={3} /></Screen>;
  if (query.isError) return <Screen><ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} /></Screen>;

  return (
    <Screen padded={false} testID="blocked-screen">
      <FlatList
        data={query.data}
        keyExtractor={(b) => b.userId}
        contentContainerStyle={query.data.length === 0 ? { flexGrow: 1 } : undefined}
        ListEmptyComponent={<EmptyState title="No blocked users" message="People you block will appear here." />}
        renderItem={({ item }) => (
          <View style={{ flexDirection: "row", alignItems: "center", padding: spacing.lg, gap: spacing.md }}>
            <Text style={{ flex: 1 }}>{item.firstName}</Text>
            <Button title="Unblock" variant="secondary" onPress={() => unblock.mutate(item.userId)} />
          </View>
        )}
      />
    </Screen>
  );
}
