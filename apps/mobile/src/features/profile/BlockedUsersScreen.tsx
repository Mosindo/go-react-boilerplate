import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../design/components";
import { errorMessage } from "../../lib/api/client";
import { safetyApi } from "../../lib/api/endpoints";
import { queryKeys } from "../../lib/queryClient";
import { showToast } from "../../lib/toast";

export default function BlockedUsersScreen() {
  const client = useQueryClient();
  const { data, isLoading, error, refetch } = useQuery({ queryKey: queryKeys.blocked, queryFn: safetyApi.blocked });
  if (isLoading) return <LoadingState />;
  if (error || !data) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  const unblock = async (userId: string) => {
    try {
      await safetyApi.unblock(userId);
      await client.invalidateQueries({ queryKey: queryKeys.blocked });
      showToast("Personne débloquée", "success");
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  return (
    <Screen edges={["bottom"]}>
      <FlatList
        ListEmptyComponent={<EmptyState icon="happy-outline" message="Vous n'avez bloqué personne." title="Aucun blocage" />}
        contentContainerStyle={styles.list}
        data={data}
        keyExtractor={(b) => b.user.userId}
        renderItem={({ item }) => (
          <View style={styles.row}>
            <Avatar name={item.user.firstName || "?"} size={44} uri={item.user.photo?.url} />
            <Text style={styles.name} variant="label">
              {item.user.firstName || "Membre"}
            </Text>
            <Button label="Débloquer" onPress={() => void unblock(item.user.userId)} size="sm" variant="secondary" />
          </View>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { padding: 16, gap: 12, flexGrow: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: 12 },
  name: { flex: 1 }
});
