import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../api/queryKeys";
import { safetyApi } from "../../api/safety";
import { errorMessage } from "../../lib/errors";
import { relativeTime } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, EmptyState, ErrorState, Header, Loading, Screen, Text, toast } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "BlockedUsers">;

export function BlockedUsersScreen({ navigation }: Props) {
  const queryClient = useQueryClient();
  const blocks = useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.blocks });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      toast("Personne débloquée");
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  return (
    <Screen edges={["top", "bottom"]} testID="blocked-screen">
      <Header title="Personnes bloquées" onBack={() => navigation.goBack()} />
      {blocks.isPending ? (
        <Loading />
      ) : blocks.isError ? (
        <ErrorState message={errorMessage(blocks.error)} onRetry={() => void blocks.refetch()} />
      ) : (
        <FlatList
          data={blocks.data.blocks}
          keyExtractor={(b) => b.userId}
          contentContainerStyle={blocks.data.blocks.length === 0 ? styles.grow : undefined}
          ListEmptyComponent={
            <EmptyState
              icon="shield-checkmark-outline"
              title="Personne n'est bloqué"
              message="Vous pouvez bloquer quelqu'un depuis son profil ou une conversation."
            />
          }
          renderItem={({ item }) => (
            <View style={styles.row}>
              <View style={styles.flex}>
                <Text variant="label">{item.firstName || "Membre"}</Text>
                <Text variant="caption" color="textMuted">
                  Bloqué·e {relativeTime(item.blockedAt).toLowerCase()}
                </Text>
              </View>
              <Button label="Débloquer" variant="secondary" compact onPress={() => unblock.mutate(item.userId)} />
            </View>
          )}
        />
      )}
      <Text variant="caption" color="textSubtle" style={styles.note}>
        Débloquer ne recrée pas d’ancien match.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  grow: { flexGrow: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.sm },
  flex: { flex: 1 },
  note: { paddingVertical: spacing.md }
});
