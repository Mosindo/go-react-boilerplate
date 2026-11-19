import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text, useFeedback } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "BlockedUsers">;

export function BlockedUsersScreen({ navigation }: Props) {
  const client = useQueryClient();
  const { toast, confirm } = useFeedback();
  const query = useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.blocks });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
      toast("Personne débloquée", "success");
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  return (
    <Screen>
      <Header title="Personnes bloquées" onBack={navigation.goBack} />
      {query.isLoading ? (
        <LoadingState />
      ) : query.error ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          data={query.data}
          keyExtractor={(b) => b.userId}
          contentContainerStyle={styles.list}
          ListEmptyComponent={
            <EmptyState icon="shield-checkmark-outline" title="Aucun blocage" message="Les personnes que vous bloquez apparaîtront ici." />
          }
          renderItem={({ item }) => (
            <View style={styles.row}>
              <Text style={styles.name}>{item.firstName}</Text>
              <Button
                label="Débloquer"
                variant="secondary"
                onPress={async () => {
                  const ok = await confirm({
                    title: `Débloquer ${item.firstName} ?`,
                    message: "Vous pourrez de nouveau vous croiser dans la découverte. Votre ancien match ne sera pas rétabli."
                  });
                  if (ok) unblock.mutate(item.userId);
                }}
              />
            </View>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { flexGrow: 1, gap: spacing.xs },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.xs },
  name: { flex: 1 }
});
