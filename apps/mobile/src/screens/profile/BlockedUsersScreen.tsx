import React from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../../api/dating";
import { keys, useBlocks } from "../../hooks/queries";
import { EmptyView, ErrorView, LoadingView, showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, ListRow } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";
import { formatRelative } from "../../utils/format";

export default function BlockedUsersScreen() {
  const query = useBlocks();
  const client = useQueryClient();
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.blocks });
      showToast("Personne débloquée");
    },
    onError: (e) => showToast(errorMessage(e), "danger")
  });

  if (query.isLoading) {
    return <LoadingView />;
  }
  if (query.isError) {
    return <ErrorView onRetry={() => void query.refetch()} />;
  }
  return (
    <Screen testID="blocked-screen">
      <ScreenHeader subtitle="Débloquer ne restaure pas un ancien match." title="Personnes bloquées" />
      {query.data?.length === 0 ? <EmptyView glyph="✓" title="Personne n'est bloqué" /> : null}
      {query.data?.map((b) => (
        <ListRow
          key={b.userId}
          subtitle={`Bloqué·e ${formatRelative(b.blockedAt)}`}
          title={b.firstName || "Membre"}
          trailing={
            <Button
              label="Débloquer"
              loading={unblock.isPending && unblock.variables === b.userId}
              onPress={() => unblock.mutate(b.userId)}
              size="sm"
              variant="secondary"
            />
          }
        />
      ))}
    </Screen>
  );
}
