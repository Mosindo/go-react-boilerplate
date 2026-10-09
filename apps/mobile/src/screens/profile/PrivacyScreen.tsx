import React from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage } from '../../api/errors';
import { profileApi, profileToInput } from '../../api/profile';
import type { MyProfile } from '../../api/types';
import { fr } from '../../i18n/fr';
import { qk } from '../../query/keys';
import { useMyProfile } from '../../query/hooks';
import { ErrorState, LoadingView, Screen, ToggleRow } from '../../shared/ui';
import { bumpDiscoverEpoch } from '../../store/appState';

export function PrivacyScreen() {
  const qc = useQueryClient();
  const { data, isPending, isError, error, refetch } = useMyProfile();

  const update = useMutation({
    mutationFn: (patch: Partial<Pick<MyProfile, 'discoverable' | 'showDistance'>>) =>
      profileApi.saveProfile(profileToInput(data!, patch)),
    onMutate: async (patch) => {
      await qc.cancelQueries({ queryKey: qk.myProfile });
      const previous = qc.getQueryData<MyProfile>(qk.myProfile);
      qc.setQueryData<MyProfile>(qk.myProfile, (old) => (old ? { ...old, ...patch } : old));
      return { previous };
    },
    onError: (_e, _v, ctx) => ctx?.previous && qc.setQueryData(qk.myProfile, ctx.previous),
    onSuccess: (saved) => {
      qc.setQueryData(qk.myProfile, saved);
      bumpDiscoverEpoch();
    },
  });

  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  return (
    <Screen scroll testID="privacy-screen">
      <ToggleRow
        testID="toggle-discoverable"
        label={fr.profile.discoverable}
        help={fr.profile.discoverableHelp}
        value={data.discoverable}
        onChange={(discoverable) => update.mutate({ discoverable })}
      />
      <ToggleRow
        testID="toggle-showDistance"
        label={fr.profile.showDistance}
        help={fr.profile.showDistanceHelp}
        value={data.showDistance}
        onChange={(showDistance) => update.mutate({ showDistance })}
      />
    </Screen>
  );
}
