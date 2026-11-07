import React from "react";
import { FlatList } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AppText, Avatar, Button, EmptyState, ErrorState, Loading, Row, Screen } from "../../components/ui";
import { safetyApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { spacing } from "../../theme/theme";

export default function BlockedScreen() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["blocks"], queryFn: safetyApi.blocks });
  const unblock = useMutation({
    mutationFn: (id: string) => safetyApi.unblock(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["blocks"] }),
  });
  return (
    <Screen>
      {q.isLoading ? (
        <Loading />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      ) : (q.data ?? []).length === 0 ? (
        <EmptyState
          title="Personne n'est bloqué"
          message="Les personnes que vous bloquez ne peuvent plus vous voir ni vous écrire."
        />
      ) : (
        <FlatList
          data={q.data}
          keyExtractor={(b) => b.userId}
          contentContainerStyle={{ gap: spacing.md, paddingVertical: spacing.lg }}
          renderItem={({ item }) => (
            <Row style={{ justifyContent: "space-between" }}>
              <Row>
                <Avatar name={item.firstName || "?"} uri={item.photoUrl} />
                <AppText style={{ fontWeight: "700" }}>{item.firstName || "Compte supprimé"}</AppText>
              </Row>
              <Button
                title="Débloquer"
                variant="secondary"
                loading={unblock.isPending}
                onPress={() => unblock.mutate(item.userId)}
              />
            </Row>
          )}
        />
      )}
    </Screen>
  );
}
