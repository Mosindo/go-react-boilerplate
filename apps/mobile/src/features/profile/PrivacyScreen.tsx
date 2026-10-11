import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage, profileApi, queryKeys } from '../../api';
import type { Privacy } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { Card, Divider, ErrorState, Loading, Screen, SwitchRow, Text } from '../../ui';
import { usePrivacy } from './hooks';

export function PrivacyScreen() {
  const qc = useQueryClient();
  const onError = useErrorToast();
  const query = usePrivacy();

  const save = useMutation({
    mutationFn: (next: Privacy) => profileApi.savePrivacy(next),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: queryKeys.privacy });
      const previous = qc.getQueryData<Privacy>(queryKeys.privacy);
      qc.setQueryData<Privacy>(queryKeys.privacy, next);
      return { previous };
    },
    onError: (error, _next, context) => {
      if (context?.previous) qc.setQueryData(queryKeys.privacy, context.previous);
      onError(error);
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.privacy });
      void qc.invalidateQueries({ queryKey: queryKeys.ownProfile });
    },
  });

  if (query.isPending) return <Loading />;
  if (query.isError) {
    return (
      <Screen edges={['bottom']}>
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  const privacy = query.data;
  return (
    <Screen scroll edges={['bottom']}>
      <Text color="textMuted">{fr.privacy.intro}</Text>
      <Card padded={false}>
        <SwitchRow
          label={fr.privacy.discoverable}
          description={fr.privacy.discoverableHelp}
          value={privacy.discoverable}
          onValueChange={(discoverable) => save.mutate({ ...privacy, discoverable })}
        />
        <Divider />
        <SwitchRow
          label={fr.privacy.showDistance}
          description={fr.privacy.showDistanceHelp}
          value={privacy.showDistance}
          onValueChange={(showDistance) => save.mutate({ ...privacy, showDistance })}
        />
      </Card>
    </Screen>
  );
}
