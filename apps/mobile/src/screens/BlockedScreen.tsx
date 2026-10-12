import { useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { datingApi } from "../api/dating";
import { keys, useBlocks } from "../hooks/queries";
import { spacing } from "../theme";
import { Avatar, Button, EmptyState, ErrorState, LoadingState, Text, useToast } from "../ui";

export default function BlockedScreen() {
  const toast = useToast();
  const client = useQueryClient();
  const { data, isLoading, isError, error, refetch } = useBlocks();

  const unblock = async (userId: string) => {
    try {
      await datingApi.unblock(userId);
      await client.invalidateQueries({ queryKey: keys.blocks });
      void client.invalidateQueries({ queryKey: keys.discover });
      toast.show("Personne débloquée", "success");
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  if (isLoading) {
    return <LoadingState />;
  }
  if (isError || !data) {
    return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;
  }
  if (data.length === 0) {
    return (
      <EmptyState
        icon="shield-checkmark-outline"
        title="Personne n'est bloqué"
        message="Les personnes que vous bloquez apparaissent ici."
      />
    );
  }
  return (
    <FlatList
      data={data}
      keyExtractor={(b) => b.userId}
      contentContainerStyle={styles.list}
      renderItem={({ item }) => (
        <View style={styles.row}>
          <Avatar uri={item.thumbUrl || null} name={item.firstName || "?"} size={48} />
          <Text style={styles.flex}>{item.firstName || "Ancien membre"}</Text>
          <Button label="Débloquer" variant="secondary" onPress={() => void unblock(item.userId)} style={styles.button} />
        </View>
      )}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.md },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  flex: { flex: 1 },
  button: { minHeight: 40, paddingHorizontal: spacing.lg }
});
